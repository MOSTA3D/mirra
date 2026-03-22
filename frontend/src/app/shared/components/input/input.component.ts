import { Component, Input, forwardRef, signal } from '@angular/core';
import { ControlValueAccessor, NG_VALUE_ACCESSOR, ReactiveFormsModule, AbstractControl } from '@angular/forms';

@Component({
  selector: 'app-input',
  standalone: true,
  imports: [ReactiveFormsModule],
  providers: [{
    provide: NG_VALUE_ACCESSOR,
    useExisting: forwardRef(() => InputComponent),
    multi: true,
  }],
  template: `
    <div class="flex flex-col gap-1.5">
      @if (label) {
        <label class="text-sm font-medium" style="color: var(--color-text-secondary)">{{ label }}</label>
      }
      <input
        [type]="type"
        [placeholder]="placeholder"
        [disabled]="disabled"
        [value]="value"
        (input)="onInput($event)"
        (blur)="onBlurEvent()"
        class="w-full px-4 py-3 rounded-xl text-sm outline-none transition-all duration-200 border"
        [style]="inputStyle()"
      />
      @if (showError()) {
        <span class="text-xs flex items-center gap-1" style="color: var(--color-error)">
          <svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor">
            <path d="M6 1a5 5 0 100 10A5 5 0 006 1zm0 2.5a.5.5 0 01.5.5v2a.5.5 0 01-1 0V4a.5.5 0 01.5-.5zm0 5a.75.75 0 110-1.5.75.75 0 010 1.5z"/>
          </svg>
          {{ errorMessage() }}
        </span>
      }
    </div>
  `,
})
export class InputComponent implements ControlValueAccessor {
  @Input() label = '';
  @Input() type = 'text';
  @Input() placeholder = '';
  @Input() error = '';
  @Input() disabled = false;
  @Input() control: AbstractControl | null = null;

  value = '';
  touched = signal(false);
  onChange = (_: any) => {};
  onTouched = () => {};

  writeValue(val: string) { this.value = val ?? ''; }
  registerOnChange(fn: any) { this.onChange = fn; }
  registerOnTouched(fn: any) { this.onTouched = fn; }
  setDisabledState(disabled: boolean) { this.disabled = disabled; }

  onInput(e: Event) {
    this.value = (e.target as HTMLInputElement).value;
    this.onChange(this.value);
  }

  onBlurEvent() {
    this.touched.set(true);
    this.onTouched();
  }

  showError(): boolean {
    if (this.error) return true;
    if (!this.control) return false;
    return this.control.invalid && (this.control.dirty || this.control.touched || this.touched());
  }

  errorMessage(): string {
    if (this.error) return this.error;
    if (!this.control?.errors) return '';
    const errors = this.control.errors;
    if (errors['required']) return 'This field is required';
    if (errors['email']) return 'Enter a valid email address';
    if (errors['minlength']) return `Minimum ${errors['minlength'].requiredLength} characters`;
    if (errors['maxlength']) return `Maximum ${errors['maxlength'].requiredLength} characters`;
    return 'Invalid value';
  }

  inputStyle(): string {
    const hasError = this.showError();
    return `
      background-color: var(--color-bg-elevated);
      border-color: ${hasError ? 'var(--color-error)' : 'var(--color-border-strong)'};
      color: var(--color-text-primary);
      ${hasError ? 'box-shadow: 0 0 0 1px var(--color-error);' : ''}
    `;
  }
}
