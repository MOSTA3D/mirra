package pipeline

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/mirra-ai/mirra/backend/internal/store"
)

// Runner orchestrates the full pipeline: ingest → extract → distill → score → export.
type Runner struct {
	ingestor  *Ingestor
	extractor *Extractor
	distiller *Distiller
	exporter  *Exporter
	stores    store.Stores
}

func NewRunner(stores store.Stores) *Runner {
	return &Runner{
		ingestor:  NewIngestor(),
		extractor: NewExtractor(),
		distiller: NewDistiller(),
		exporter:  NewExporter(),
		stores:    stores,
	}
}

// ProcessResult holds the pipeline output.
type ProcessResult struct {
	Profile    *PersonaProfile
	Markdown   string
	Confidence map[string]float64
}

// Run processes all sources for a persona and updates its status.
// This runs asynchronously — call it in a goroutine.
func (r *Runner) Run(ctx context.Context, personaID string, personaName string, sources []*store.Source) {
	jobID := uuid.NewString()
	now := time.Now().UTC()

	job := &store.Job{
		ID:          jobID,
		PersonaID:   personaID,
		Status:      "running",
		CurrentStep: "ingest",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Get the persona to find owner
	persona, err := r.stores.Personas.GetByID(ctx, personaID)
	if err != nil {
		log.Printf("pipeline: failed to get persona %s: %v", personaID, err)
		return
	}
	job.OwnerID = persona.OwnerID

	if err := r.stores.Jobs.Create(ctx, job); err != nil {
		log.Printf("pipeline: failed to create job: %v", err)
		return
	}

	// Update persona status to processing
	persona.Status = "processing"
	persona.UpdatedAt = time.Now().UTC()
	r.stores.Personas.Update(ctx, persona)

	result, err := r.runPipeline(ctx, job, personaName, sources)
	if err != nil {
		log.Printf("pipeline: job %s failed: %v", jobID, err)
		job.Status = "failed"
		job.ErrorLog = err.Error()
		job.UpdatedAt = time.Now().UTC()
		r.stores.Jobs.Update(ctx, job)

		persona.Status = "draft"
		persona.UpdatedAt = time.Now().UTC()
		r.stores.Personas.Update(ctx, persona)
		return
	}

	// Update persona with results
	persona.Status = "ready"
	persona.Confidence = result.Confidence
	persona.UpdatedAt = time.Now().UTC()
	r.stores.Personas.Update(ctx, persona)

	// Store the markdown export result in the job
	job.Status = "done"
	job.CurrentStep = "done"
	job.ErrorLog = result.Markdown // reuse field for now — replace with dedicated field in postgres phase
	job.UpdatedAt = time.Now().UTC()
	r.stores.Jobs.Update(ctx, job)

	log.Printf("pipeline: job %s done for persona %s", jobID, personaID)
}

func (r *Runner) runPipeline(ctx context.Context, job *store.Job, personaName string, sources []*store.Source) (*ProcessResult, error) {
	// Step 1: Ingest
	r.updateStep(ctx, job, "ingest")
	var chunks []*Chunk
	for _, src := range sources {
		chunk := r.ingestor.Ingest(src.Type, src.Content, src.SpeakerName)
		chunks = append(chunks, chunk)
		src.Status = "processed"
	}

	if len(chunks) == 0 {
		return nil, fmt.Errorf("no content to process")
	}

	// Step 2: Extract
	r.updateStep(ctx, job, "extract")
	signals := r.extractor.Extract(chunks)

	if signals.TotalWords < 10 {
		return nil, fmt.Errorf("insufficient content: only %d words found across all sources", signals.TotalWords)
	}

	// Step 3: Distill
	r.updateStep(ctx, job, "distill")
	profile := r.distiller.Distill(personaName, signals)

	// Step 4: Score
	r.updateStep(ctx, job, "score")
	confidence := make(map[string]float64)
	for key, dim := range profile.Dimensions {
		confidence[key] = dim.Score
	}

	// Step 5: Export
	r.updateStep(ctx, job, "export")
	markdown := r.exporter.ToMarkdown(profile)

	return &ProcessResult{
		Profile:    profile,
		Markdown:   markdown,
		Confidence: confidence,
	}, nil
}

func (r *Runner) updateStep(ctx context.Context, job *store.Job, step string) {
	job.CurrentStep = step
	job.UpdatedAt = time.Now().UTC()
	r.stores.Jobs.Update(ctx, job)
}
