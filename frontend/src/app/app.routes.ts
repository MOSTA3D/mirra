import { Routes } from '@angular/router';
import { authGuard } from './core/guards/auth.guard';

export const routes: Routes = [
  {
    path: '',
    loadComponent: () => import('./features/auth/landing/landing.component').then(m => m.LandingComponent),
  },
  {
    path: 'auth',
    children: [
      {
        path: 'login',
        loadComponent: () => import('./features/auth/login/login.component').then(m => m.LoginComponent),
      },
      {
        path: 'register',
        loadComponent: () => import('./features/auth/register/register.component').then(m => m.RegisterComponent),
      },
    ]
  },
  {
    path: 'dashboard',
    canActivate: [authGuard],
    loadComponent: () => import('./features/dashboard/dashboard.component').then(m => m.DashboardComponent),
  },
  {
    path: 'persona',
    canActivate: [authGuard],
    children: [
      {
        path: 'new',
        loadComponent: () => import('./features/persona/create/create.component').then(m => m.CreatePersonaComponent),
      },
      {
        path: ':id',
        loadComponent: () => import('./features/persona/detail/detail.component').then(m => m.PersonaDetailComponent),
      },
    ]
  },
  {
    path: '**',
    redirectTo: '',
  }
];
