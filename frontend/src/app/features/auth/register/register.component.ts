import { Component } from '@angular/core';
import { RouterLink } from '@angular/router';

@Component({
  selector: 'app-register',
  standalone: true,
  imports: [RouterLink],
  template: `<div class="min-h-screen flex items-center justify-center" style="background-color: var(--color-bg-base)"><p style="color: var(--color-text-primary)">Register — coming soon</p></div>`,
})
export class RegisterComponent {}
