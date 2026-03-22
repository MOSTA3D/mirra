import { Component, signal } from '@angular/core';
import { Router, RouterLink } from '@angular/router';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { PersonaService } from '../../../core/services/persona.service';
import { ButtonComponent } from '../../../shared/components/button/button.component';
import { InputComponent } from '../../../shared/components/input/input.component';

type Step = 'name' | 'sources' | 'review';
type SourceType = 'url' | 'text' | 'pdf';

interface DraftSource {
  type: SourceType;
  content: string;
}

@Component({
  selector: 'app-create-persona',
  standalone: true,
  imports: [RouterLink, ReactiveFormsModule, ButtonComponent, InputComponent],
  templateUrl: './create.component.html',
})
export class CreatePersonaComponent {
  step = signal<Step>('name');
  loading = signal(false);
  error = signal('');

  personaId = signal('');
  personaName = signal('');
  sources = signal<DraftSource[]>([]);

  activeSourceType = signal<SourceType>('text');
  sourceContent = signal('');
  sourceTypes: SourceType[] = ['text', 'url', 'pdf'];

  nameForm!: ReturnType<FormBuilder['group']>;

  steps: Step[] = ['name', 'sources', 'review'];

  constructor(
    private router: Router,
    private personaService: PersonaService,
    fb: FormBuilder,
  ) {
    this.nameForm = fb.group({
      name: ['', [Validators.required, Validators.minLength(2)]],
      visibility: ['private'],
    });
  }

  get stepIndex() { return this.steps.indexOf(this.step()); }

  createPersona() {
    if (this.nameForm.invalid || this.loading()) return;
    this.loading.set(true);
    this.error.set('');

    const { name, visibility } = this.nameForm.value;
    this.personaService.create({ name: name!, visibility: visibility as any }).subscribe({
      next: (res) => {
        this.personaId.set(res.data.id);
        this.personaName.set(res.data.name);
        this.step.set('sources');
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err?.error?.error?.message ?? 'Failed to create persona');
        this.loading.set(false);
      },
    });
  }

  addSource() {
    const content = this.sourceContent().trim();
    if (!content) return;

    this.sources.update(s => [...s, { type: this.activeSourceType(), content }]);
    this.sourceContent.set('');
  }

  removeSource(index: number) {
    this.sources.update(s => s.filter((_, i) => i !== index));
  }

  goToReview() {
    this.step.set('review');
  }

  async submitSources() {
    if (this.loading()) return;
    this.loading.set(true);
    this.error.set('');

    const id = this.personaId();
    const srcs = this.sources();

    try {
      // Submit all sources sequentially
      for (const src of srcs) {
        await new Promise<void>((resolve, reject) => {
          this.personaService.addSource(id, { type: src.type, content: src.content }).subscribe({
            next: () => resolve(),
            error: (e) => reject(e),
          });
        });
      }

      // Trigger processing pipeline
      await new Promise<void>((resolve) => {
        this.personaService.process(id).subscribe({
          next: () => resolve(),
          error: () => resolve(), // Non-fatal — navigate anyway
        });
      });

      this.router.navigate(['/persona', id]);
    } catch (err: any) {
      this.error.set(err?.error?.error?.message ?? 'Failed to submit sources');
      this.loading.set(false);
    }
  }

  sourceTypeLabel(type: SourceType): string {
    return { url: '🔗 URL', text: '📝 Text', pdf: '📄 PDF' }[type];
  }
}
