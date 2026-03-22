import { Component, Input, forwardRef } from '@angular/core';
import { ControlValueAccessor, NG_VALUE_ACCESSOR, ReactiveFormsModule } from '@angular/forms';

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
        (blur)="onTouched()"
        class="w-full px-4 py-3 rounded-xl text-sm outline-none transition-all duration-200 border"
        style="
          background-color: var(--color-bg-elevated);
          border-color: var(--color-border-strong);
          color: var(--color-text-primary);
        "
      />
      @if (error) {
        <span class="text-xs" style="color: var(--color-error)">{{ error }}</span>
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

  value = '';
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
}
