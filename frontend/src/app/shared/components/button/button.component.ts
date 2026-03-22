import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-button',
  standalone: true,
  imports: [CommonModule],
  template: `
    <button
      [type]="type"
      [disabled]="disabled || loading"
      [class]="computedClass"
    >
      @if (loading) {
        <span class="inline-block w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin mr-2"></span>
      }
      <ng-content />
    </button>
  `,
})
export class ButtonComponent {
  @Input() type: 'button' | 'submit' = 'button';
  @Input() variant: 'primary' | 'secondary' | 'ghost' = 'primary';
  @Input() disabled = false;
  @Input() loading = false;
  @Input() fullWidth = false;

  get computedClass(): string {
    const base = 'inline-flex items-center justify-center px-6 py-3 rounded-xl font-semibold text-sm transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed';
    const width = this.fullWidth ? 'w-full' : '';
    const variants: Record<string, string> = {
      primary: 'gradient-accent text-white hover:opacity-90',
      secondary: 'border text-sm font-medium hover:opacity-80',
      ghost: 'hover:opacity-70',
    };
    return [base, width, variants[this.variant]].filter(Boolean).join(' ');
  }
}
