import { Injectable, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { environment } from '../../../environments/environment';

export interface Persona {
  id: string;
  ownerId: string;
  name: string;
  slug: string;
  visibility: 'private' | 'public';
  status: 'draft' | 'processing' | 'ready';
  disclaimer: string;
  confidence: Record<string, number>;
  createdAt: string;
  updatedAt: string;
}

export interface Source {
  id: string;
  personaId: string;
  type: 'url' | 'pdf' | 'text';
  content: string;
  status: 'pending' | 'processed' | 'failed';
  createdAt: string;
}

export interface CreatePersonaDto {
  name: string;
  visibility?: 'private' | 'public';
}

export interface AddSourceDto {
  type: 'url' | 'pdf' | 'text';
  content: string;
}

@Injectable({ providedIn: 'root' })
export class PersonaService {
  readonly personas = signal<Persona[]>([]);
  readonly loading = signal(false);

  constructor(private http: HttpClient) {}

  loadAll() {
    this.loading.set(true);
    return this.http.get<{ data: Persona[] }>(`${environment.apiUrl}/personas`);
  }

  create(dto: CreatePersonaDto) {
    return this.http.post<{ data: Persona }>(`${environment.apiUrl}/personas`, dto);
  }

  get(id: string) {
    return this.http.get<{ data: Persona }>(`${environment.apiUrl}/personas/${id}`);
  }

  addSource(personaId: string, dto: AddSourceDto) {
    return this.http.post<{ data: Source }>(`${environment.apiUrl}/personas/${personaId}/sources`, dto);
  }

  process(personaId: string) {
    return this.http.post<{ data: { message: string; status: string } }>(`${environment.apiUrl}/personas/${personaId}/process`, {});
  }

  export(personaId: string) {
    return this.http.post<{ data: { content: string } }>(`${environment.apiUrl}/personas/${personaId}/export`, {});
  }
}
