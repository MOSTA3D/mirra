import { Component } from '@angular/core';
import { RouterLink } from '@angular/router';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [RouterLink],
  template: `<div class="min-h-screen p-8" style="background-color: var(--color-bg-base)">
    <h1 class="text-2xl font-bold" style="color: var(--color-text-primary)">Dashboard</h1>
    <p style="color: var(--color-text-muted)" class="mt-2">Your personas will appear here.</p>
    <a routerLink="/persona/new" class="inline-block mt-6 px-6 py-3 rounded-xl font-semibold text-white gradient-accent">
      + New Persona
    </a>
  </div>`,
})
export class DashboardComponent {}
