import { Component, Output, EventEmitter, signal } from '@angular/core';

export interface UploadedFile {
  name: string;
  content: string; // extracted text content
  size: number;
}

@Component({
  selector: 'app-file-upload',
  standalone: true,
  template: `
    <div
      class="w-full rounded-xl border-2 border-dashed transition-all cursor-pointer flex flex-col items-center justify-center gap-3 py-10"
      [style.border-color]="dragging() ? 'var(--color-accent-solid)' : 'var(--color-border-strong)'"
      [style.background-color]="dragging() ? 'rgba(139,92,246,0.05)' : 'var(--color-bg-elevated)'"
      (click)="fileInput.click()"
      (dragover)="$event.preventDefault(); dragging.set(true)"
      (dragleave)="dragging.set(false)"
      (drop)="onDrop($event)">

      @if (uploading()) {
        <div class="w-6 h-6 border-2 rounded-full animate-spin"
             style="border-color: var(--color-border-strong); border-top-color: var(--color-accent-solid)"></div>
        <p class="text-sm" style="color: var(--color-text-muted)">Reading file...</p>
      } @else if (file()) {
        <div class="text-center">
          <p class="text-2xl mb-1">📄</p>
          <p class="text-sm font-medium" style="color: var(--color-text-primary)">{{ file()!.name }}</p>
          <p class="text-xs mt-1" style="color: var(--color-text-muted)">{{ formatSize(file()!.size) }} · Click to replace</p>
        </div>
      } @else {
        <p class="text-3xl">📎</p>
        <div class="text-center">
          <p class="text-sm font-medium" style="color: var(--color-text-primary)">Drop file here or click to browse</p>
          <p class="text-xs mt-1" style="color: var(--color-text-muted)">.txt, .md, .pdf (text will be extracted)</p>
        </div>
      }

      <input #fileInput type="file" accept=".txt,.md,.pdf,.json" class="hidden"
             (change)="onFileChange($event)" />
    </div>

    @if (error()) {
      <p class="mt-2 text-xs" style="color: var(--color-error)">{{ error() }}</p>
    }
  `,
})
export class FileUploadComponent {
  @Output() fileLoaded = new EventEmitter<UploadedFile>();

  dragging = signal(false);
  uploading = signal(false);
  file = signal<UploadedFile | null>(null);
  error = signal('');

  onDrop(e: DragEvent) {
    e.preventDefault();
    this.dragging.set(false);
    const f = e.dataTransfer?.files[0];
    if (f) this.processFile(f);
  }

  onFileChange(e: Event) {
    const f = (e.target as HTMLInputElement).files?.[0];
    if (f) this.processFile(f);
  }

  processFile(f: File) {
    this.uploading.set(true);
    this.error.set('');

    const reader = new FileReader();
    reader.onload = (ev) => {
      const content = ev.target?.result as string;
      const uploaded: UploadedFile = { name: f.name, content, size: f.size };
      this.file.set(uploaded);
      this.fileLoaded.emit(uploaded);
      this.uploading.set(false);
    };
    reader.onerror = () => {
      this.error.set('Failed to read file. Please try again.');
      this.uploading.set(false);
    };
    reader.readAsText(f);
  }

  formatSize(bytes: number): string {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / 1024 / 1024).toFixed(1) + ' MB';
  }
}
