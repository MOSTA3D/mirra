package pipeline

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/mirra-ai/mirra/backend/internal/llm"
	"github.com/mirra-ai/mirra/backend/internal/store"
)

// Runner orchestrates the full pipeline: ingest → extract → distill → [LLM enrich] → score → export.
type Runner struct {
	ingestor  *Ingestor
	extractor *Extractor
	distiller *Distiller
	exporter  *Exporter
	stores    store.Stores
	llm       llm.Provider // optional — falls back to rule-based if nil/unavailable
}

func NewRunner(stores store.Stores, llmProvider llm.Provider) *Runner {
	return &Runner{
		ingestor:  NewIngestor(),
		extractor: NewExtractor(),
		distiller: NewDistiller(),
		exporter:  NewExporter(),
		stores:    stores,
		llm:       llmProvider,
	}
}

// ProcessResult holds the pipeline output.
type ProcessResult struct {
	Profile     *PersonaProfile
	Markdown    string
	Confidence  map[string]float64
	Suggestions []string
}

// Run processes all sources for a persona and updates its status.
func (r *Runner) Run(ctx context.Context, personaID string, personaName string, sources []*store.Source) {
	jobID := uuid.NewString()
	now := time.Now().UTC()

	persona, err := r.stores.Personas.GetByID(ctx, personaID)
	if err != nil {
		log.Printf("pipeline: failed to get persona %s: %v", personaID, err)
		return
	}

	job := &store.Job{
		ID:          jobID,
		PersonaID:   personaID,
		OwnerID:     persona.OwnerID,
		Status:      "running",
		CurrentStep: "ingest",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := r.stores.Jobs.Create(ctx, job); err != nil {
		log.Printf("pipeline: failed to create job: %v", err)
		return
	}

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

	persona.Status = "ready"
	persona.Confidence = result.Confidence
	persona.Suggestions = result.Suggestions
	persona.CachedMarkdown = result.Markdown
	persona.UpdatedAt = time.Now().UTC()
	r.stores.Personas.Update(ctx, persona)

	job.Status = "done"
	job.CurrentStep = "done"
	job.UpdatedAt = time.Now().UTC()
	r.stores.Jobs.Update(ctx, job)

	log.Printf("pipeline: job %s done for persona %s (llm=%v)", jobID, personaID, r.llm != nil && r.llm.Available())
}

func (r *Runner) runPipeline(ctx context.Context, job *store.Job, personaName string, sources []*store.Source) (*ProcessResult, error) {
	// Step 1: Ingest
	r.updateStep(ctx, job, "ingest")
	var chunks []*Chunk
	for _, src := range sources {
		chunks = append(chunks, r.ingestor.Ingest(src.Type, src.Content, src.SpeakerName))
		src.Status = "processed"
	}
	if len(chunks) == 0 {
		return nil, fmt.Errorf("no content to process")
	}

	// Step 2: Extract
	r.updateStep(ctx, job, "extract")
	signals := r.extractor.Extract(chunks)
	if signals.TotalWords < 10 {
		return nil, fmt.Errorf("insufficient content: only %d words", signals.TotalWords)
	}

	// Step 3: Distill (rule-based baseline)
	r.updateStep(ctx, job, "distill")
	profile := r.distiller.Distill(personaName, signals)

	// Step 4: LLM enrichment (replaces rule-based summary/voice if available)
	if r.llm != nil && r.llm.Available() {
		r.updateStep(ctx, job, "llm_enrich")
		log.Printf("pipeline: calling LLM (%s) for persona enrichment", r.llm.Name())

		llmInput := llm.SynthesisInput{
			Name:           personaName,
			HumorScore:     profile.Dimensions["humor"].Score,
			ToneScore:      1 - profile.Dimensions["tone"].Score, // convert back to formality
			OpinionScore:   profile.Dimensions["opinions"].Score,
			EmotionScore:   profile.Dimensions["emotion"].Score,
			AvgSentenceLen: signals.AvgSentenceLen,
			TopInterests:   profile.Interests,
			SampleQuotes:   profile.SamplePhrases,
			CommonWords:    profile.VocabStyle.CommonWords,
			Locations:      profile.Locations,
			Foods:          profile.Foods,
			Quirks:         profile.Quirks,
		}

		llmOutput, err := r.llm.Synthesize(ctx, llmInput)
		if err != nil {
			// LLM failure is non-fatal — log and continue with rule-based
			log.Printf("pipeline: LLM enrichment failed (using rule-based fallback): %v", err)
		} else {
			// Enrich profile with LLM output
			profile.Summary = llmOutput.Summary
			profile.VoiceGuide = llmOutput.VoiceGuide
			profile.LLMSystemPrompt = llmOutput.SystemPrompt
			if len(llmOutput.CoreTraits) > 0 {
				profile.CoreBeliefs = llmOutput.CoreTraits
			}
			profile.LLMEnriched = true
			log.Printf("pipeline: LLM enrichment successful")
		}
	}

	// Step 5: Score
	r.updateStep(ctx, job, "score")
	confidence := make(map[string]float64)
	for key, dim := range profile.Dimensions {
		confidence[key] = dim.Score
	}

	// Step 6: Export
	r.updateStep(ctx, job, "export")
	markdown := r.exporter.ToMarkdown(profile)

	return &ProcessResult{
		Profile:     profile,
		Markdown:    markdown,
		Confidence:  confidence,
		Suggestions: signals.Diagnostics.SuggestedActions,
	}, nil
}

func (r *Runner) updateStep(ctx context.Context, job *store.Job, step string) {
	job.CurrentStep = step
	job.UpdatedAt = time.Now().UTC()
	r.stores.Jobs.Update(ctx, job)
}
