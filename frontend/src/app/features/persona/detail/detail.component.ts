import { Component, OnInit, OnDestroy, signal } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { DatePipe } from '@angular/common';
import { PersonaService, Persona, Source } from '../../../core/services/persona.service';
import { ButtonComponent } from '../../../shared/components/button/button.component';
import { FileUploadComponent } from '../../../shared/components/file-upload/file-upload.component';

type SourceType = 'text' | 'url' | 'whatsapp' | 'telegram' | 'twitter' | 'instagram' | 'pdf';

const CHAT_FORMATS: SourceType[] = ['whatsapp', 'telegram'];
const FILE_FORMATS: SourceType[] = ['pdf', 'twitter', 'instagram', 'telegram'];

const SOURCE_LABELS: Record<string, string> = {
  text: '📝 Text', url: '🔗 URL', pdf: '📄 PDF',
  whatsapp: '💬 WhatsApp', telegram: '✈️ Telegram',
  twitter: '🐦 Twitter/X', instagram: '📸 Instagram',
};

@Component({
  selector: 'app-persona-detail',
  standalone: true,
  imports: [RouterLink, ButtonComponent, DatePipe, FileUploadComponent],
  templateUrl: './detail.component.html',
})
export class PersonaDetailComponent implements OnInit, OnDestroy {
  persona = signal<Persona | null>(null);
  sources = signal<Source[]>([]);
  loading = signal(true);
  exportContent = signal('');
  exporting = signal(false);
  copied = signal(false);

  // Add source panel
  showAddSource = signal(false);
  activeSourceType = signal<SourceType>('text');
  sourceContent = signal('');
  speakerName = signal('');
  addingSource = signal(false);
  addSourceError = signal('');
  sourceTypes: SourceType[] = ['text', 'url', 'whatsapp', 'telegram', 'twitter', 'instagram', 'pdf'];

  // Rebuild
  rebuilding = signal(false);

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

  ngOnDestroy() { this.stopPolling(); }

  loadPersona() {
    this.personaService.get(this.personaId).subscribe({
      next: (res) => {
        this.persona.set(res.data);
        this.loading.set(false);
        if (res.data.status === 'processing') {
          this.startPolling();
        } else if (res.data.status === 'ready') {
          this.stopPolling();
          if (!this.exportContent()) this.exportPersona();
        }
        this.loadSources();
      },
      error: () => this.loading.set(false),
    });
  }

  loadSources() {
    this.personaService.getSources(this.personaId).subscribe({
      next: (res) => this.sources.set(res.data ?? []),
      error: () => {},
    });
  }

  startPolling() {
    if (this.pollInterval) return;
    this.pollInterval = setInterval(() => this.loadPersona(), 2500);
  }

  stopPolling() {
    if (this.pollInterval) { clearInterval(this.pollInterval); this.pollInterval = null; }
  }

  exportPersona() {
    if (!this.persona() || this.exporting()) return;
    this.exporting.set(true);
    this.personaService.export(this.persona()!.id).subscribe({
      next: (res) => { this.exportContent.set(res.data.content); this.exporting.set(false); },
      error: () => this.exporting.set(false),
    });
  }

  copyToClipboard() {
    navigator.clipboard.writeText(this.exportContent()).then(() => {
      this.copied.set(true);
      setTimeout(() => this.copied.set(false), 2000);
    });
  }

  addSource() {
    const content = this.sourceContent().trim();
    if (!content || this.addingSource()) return;
    this.addingSource.set(true);
    this.addSourceError.set('');

    this.personaService.addSource(this.personaId, {
      type: this.activeSourceType() as any,
      content,
      speakerName: this.speakerName().trim() || undefined,
    }).subscribe({
      next: () => {
        this.sourceContent.set('');
        this.speakerName.set('');
        this.addingSource.set(false);
        this.showAddSource.set(false);
        this.loadSources();
      },
      error: (err) => {
        this.addSourceError.set(err?.error?.error?.message ?? 'Failed to add source');
        this.addingSource.set(false);
      },
    });
  }

  rebuild() {
    if (this.rebuilding()) return;
    this.rebuilding.set(true);
    this.exportContent.set('');

    this.personaService.process(this.personaId).subscribe({
      next: () => {
        this.rebuilding.set(false);
        this.loadPersona();
        this.startPolling();
      },
      error: (err) => {
        this.addSourceError.set(err?.error?.error?.message ?? 'Failed to start processing');
        this.rebuilding.set(false);
      },
    });
  }

  onFileLoaded(file: { content: string }) { this.sourceContent.set(file.content); }

  isChatFormat = (t: SourceType) => CHAT_FORMATS.includes(t);
  isFileFormat = (t: SourceType) => FILE_FORMATS.includes(t);
  sourceLabel = (t: SourceType) => SOURCE_LABELS[t] ?? t;

  confidenceDimensions(confidence: Record<string, number>) {
    return Object.entries(confidence).map(([key, val]) => ({ key, val }));
  }

  confidenceColor(val: number): string {
    if (val >= 0.7) return 'var(--color-success)';
    if (val >= 0.4) return 'var(--color-warning)';
    return 'var(--color-error)';
  }
}
