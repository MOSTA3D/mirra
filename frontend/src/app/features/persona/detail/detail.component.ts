import { Component, OnInit, OnDestroy, signal } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { DatePipe } from '@angular/common';
import { PersonaService, Persona } from '../../../core/services/persona.service';
import { ButtonComponent } from '../../../shared/components/button/button.component';

@Component({
  selector: 'app-persona-detail',
  standalone: true,
  imports: [RouterLink, ButtonComponent, DatePipe],
  templateUrl: './detail.component.html',
})
export class PersonaDetailComponent implements OnInit, OnDestroy {
  persona = signal<Persona | null>(null);
  loading = signal(true);
  exportContent = signal('');
  exporting = signal(false);
  copied = signal(false);

  private personaId = '';
  private pollInterval: ReturnType<typeof setInterval> | null = null;

  constructor(
    private route: ActivatedRoute,
    private personaService: PersonaService,
  ) {}

  ngOnInit() {
    this.personaId = this.route.snapshot.paramMap.get('id')!;
    this.loadPersona();
  }

  ngOnDestroy() {
    this.stopPolling();
  }

  loadPersona() {
    this.personaService.get(this.personaId).subscribe({
      next: (res) => {
        this.persona.set(res.data);
        this.loading.set(false);

        if (res.data.status === 'processing') {
          this.startPolling();
        } else if (res.data.status === 'ready') {
          this.stopPolling();
          this.exportPersona(); // auto-load export when ready
        }
      },
      error: () => this.loading.set(false),
    });
  }

  startPolling() {
    if (this.pollInterval) return;
    this.pollInterval = setInterval(() => this.loadPersona(), 2000);
  }

  stopPolling() {
    if (this.pollInterval) {
      clearInterval(this.pollInterval);
      this.pollInterval = null;
    }
  }

  exportPersona() {
    if (!this.persona() || this.exporting()) return;
    this.exporting.set(true);

    this.personaService.export(this.persona()!.id).subscribe({
      next: (res) => {
        this.exportContent.set(res.data.content);
        this.exporting.set(false);
      },
      error: () => this.exporting.set(false),
    });
  }

  copyToClipboard() {
    navigator.clipboard.writeText(this.exportContent()).then(() => {
      this.copied.set(true);
      setTimeout(() => this.copied.set(false), 2000);
    });
  }

  confidenceDimensions(confidence: Record<string, number>) {
    return Object.entries(confidence).map(([key, val]) => ({ key, val }));
  }

  confidenceColor(val: number): string {
    if (val >= 0.7) return 'var(--color-success)';
    if (val >= 0.4) return 'var(--color-warning)';
    return 'var(--color-error)';
  }
}
