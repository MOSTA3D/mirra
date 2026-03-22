import { Component, OnInit, signal } from '@angular/core';
import { RouterLink, Router } from '@angular/router';
import { DatePipe } from '@angular/common';
import { AuthService } from '../../core/services/auth.service';
import { PersonaService, Persona } from '../../core/services/persona.service';
import { ButtonComponent } from '../../shared/components/button/button.component';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [RouterLink, ButtonComponent, DatePipe],
  templateUrl: './dashboard.component.html',
})
export class DashboardComponent implements OnInit {
  personas = signal<Persona[]>([]);
  loading = signal(true);

  constructor(
    private auth: AuthService,
    private personaService: PersonaService,
  ) {}

  ngOnInit() {
    this.personaService.loadAll().subscribe({
      next: (res) => {
        this.personas.set(res.data ?? []);
        this.loading.set(false);
      },
      error: () => this.loading.set(false),
    });
  }

  logout() {
    this.auth.logout();
  }

  statusColor(status: string): string {
    return { draft: '#71717A', processing: '#F59E0B', ready: '#22C55E' }[status] ?? '#71717A';
  }

  statusLabel(status: string): string {
    return { draft: 'Draft', processing: 'Processing...', ready: 'Ready' }[status] ?? status;
  }
}
