import { Component, signal } from '@angular/core';
import { RouterLink, Router } from '@angular/router';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { AuthService } from '../../../core/services/auth.service';
import { ButtonComponent } from '../../../shared/components/button/button.component';
import { InputComponent } from '../../../shared/components/input/input.component';

type Step = 'email' | 'verify' | 'password';

@Component({
  selector: 'app-register',
  standalone: true,
  imports: [RouterLink, ReactiveFormsModule, ButtonComponent, InputComponent],
  templateUrl: './register.component.html',
})
export class RegisterComponent {
  step = signal<Step>('email');
  stepIndex = () => (['email','verify','password'] as Step[]).indexOf(this.step());
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

  sendCode() {
    if (this.emailForm.invalid || this.loading()) return;
    this.loading.set(true);
    this.error.set('');

    const email = this.emailForm.value.email!;
    this.auth.sendVerificationCode(email).subscribe({
      next: () => {
        this.email.set(email);
        this.step.set('verify');
        this.loading.set(false);
        this.startResendCooldown();
      },
      error: (err) => {
        this.error.set(err?.error?.error?.message ?? 'Failed to send code. Try again.');
        this.loading.set(false);
      },
    });
  }

  verifyCode() {
    if (this.codeForm.invalid || this.loading()) return;
    this.loading.set(true);
    this.error.set('');

    this.auth.verifyCode(this.email(), this.codeForm.value.code!).subscribe({
      next: () => {
        this.step.set('password');
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err?.error?.error?.message ?? 'Invalid code. Please try again.');
        this.loading.set(false);
      },
    });
  }

  resendCode() {
    if (this.resendCooldown() > 0) return;
    this.loading.set(true);
    this.error.set('');

    this.auth.sendVerificationCode(this.email()).subscribe({
      next: () => {
        this.loading.set(false);
        this.startResendCooldown();
      },
      error: () => this.loading.set(false),
    });
  }

  register() {
    if (this.passwordForm.invalid || this.loading()) return;
    this.loading.set(true);
    this.error.set('');

    this.auth.register(this.email(), this.passwordForm.value.password!).subscribe({
      next: () => this.router.navigate(['/dashboard']),
      error: (err) => {
        this.error.set(err?.error?.error?.message ?? 'Something went wrong. Please try again.');
        this.loading.set(false);
      },
    });
  }

  private startResendCooldown() {
    this.resendCooldown.set(60);
    const interval = setInterval(() => {
      this.resendCooldown.update(v => {
        if (v <= 1) { clearInterval(interval); return 0; }
        return v - 1;
      });
    }, 1000);
  }
}
