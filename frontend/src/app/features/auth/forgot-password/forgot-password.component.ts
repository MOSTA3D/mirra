import { Component, signal } from '@angular/core';
import { RouterLink, Router } from '@angular/router';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { AuthService } from '../../../core/services/auth.service';
import { ButtonComponent } from '../../../shared/components/button/button.component';
import { InputComponent } from '../../../shared/components/input/input.component';

type Step = 'email' | 'verify' | 'password' | 'done';

@Component({
  selector: 'app-forgot-password',
  standalone: true,
  imports: [RouterLink, ReactiveFormsModule, ButtonComponent, InputComponent],
  templateUrl: './forgot-password.component.html',
})
export class ForgotPasswordComponent {
  step = signal<Step>('email');
  loading = signal(false);
  error = signal('');
  email = signal('');
  resendCooldown = signal(0);

  emailForm;
  codeForm;
  passwordForm;

  constructor(
    fb: FormBuilder,
    private auth: AuthService,
    private router: Router,
  ) {
    this.emailForm = fb.group({
      email: ['', [Validators.required, Validators.email]],
    });
    this.codeForm = fb.group({
      code: ['', [Validators.required, Validators.minLength(6), Validators.maxLength(6)]],
    });
    this.passwordForm = fb.group({
      password: ['', [Validators.required, Validators.minLength(8)]],
    });
  }

  stepIndex = () => (['email', 'verify', 'password'] as Step[]).indexOf(this.step());

  requestCode() {
    if (this.emailForm.invalid || this.loading()) return;
    this.loading.set(true);
    this.error.set('');

    const email = this.emailForm.value.email!;
    this.auth.forgotPassword(email).subscribe({
      next: () => {
        this.email.set(email);
        this.step.set('verify');
        this.loading.set(false);
        this.startCooldown();
      },
      error: () => {
        // Always show success to prevent enumeration
        this.email.set(email);
        this.step.set('verify');
        this.loading.set(false);
        this.startCooldown();
      },
    });
  }

  verifyCode() {
    if (this.codeForm.invalid || this.loading()) return;
    this.loading.set(true);
    this.error.set('');

    this.auth.verifyResetCode(this.email(), this.codeForm.value.code!).subscribe({
      next: () => {
        this.step.set('password');
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err?.error?.error?.message ?? 'Invalid or expired code');
        this.loading.set(false);
      },
    });
  }

  resendCode() {
    if (this.resendCooldown() > 0) return;
    this.auth.forgotPassword(this.email()).subscribe({ next: () => this.startCooldown(), error: () => this.startCooldown() });
  }

  resetPassword() {
    if (this.passwordForm.invalid || this.loading()) return;
    this.loading.set(true);
    this.error.set('');

    this.auth.resetPassword(
      this.email(),
      this.codeForm.value.code!,
      this.passwordForm.value.password!
    ).subscribe({
      next: () => {
        this.loading.set(false);
        this.step.set('done');
      },
      error: (err) => {
        this.error.set(err?.error?.error?.message ?? 'Failed to reset password. Try again.');
        this.loading.set(false);
      },
    });
  }

  private startCooldown() {
    this.resendCooldown.set(60);
    const interval = setInterval(() => {
      this.resendCooldown.update(v => {
        if (v <= 1) { clearInterval(interval); return 0; }
        return v - 1;
      });
    }, 1000);
  }
}
