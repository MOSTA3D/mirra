import { Component, signal } from '@angular/core';
import { RouterLink, Router } from '@angular/router';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { AuthService } from '../../../core/services/auth.service';
import { ButtonComponent } from '../../../shared/components/button/button.component';
import { InputComponent } from '../../../shared/components/input/input.component';

@Component({
  selector: 'app-register',
  standalone: true,
  imports: [RouterLink, ReactiveFormsModule, ButtonComponent, InputComponent],
  templateUrl: './register.component.html',
})
export class RegisterComponent {
  loading = signal(false);
  error = signal('');
  form;

  constructor(
    fb: FormBuilder,
    private auth: AuthService,
    private router: Router,
  ) {
    this.form = fb.group({
      email: ['', [Validators.required, Validators.email]],
      password: ['', [Validators.required, Validators.minLength(8)]],
    });
  }

  submit() {
    if (this.form.invalid || this.loading()) return;

    this.loading.set(true);
    this.error.set('');

    const { email, password } = this.form.value;

    this.auth.register(email!, password!).subscribe({
      next: () => this.router.navigate(['/dashboard']),
      error: (err) => {
        this.error.set(err?.error?.error?.message ?? 'Something went wrong. Please try again.');
        this.loading.set(false);
      },
    });
  }
}
