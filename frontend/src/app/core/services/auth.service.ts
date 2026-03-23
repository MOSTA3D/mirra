import { Injectable, signal, computed } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Router } from '@angular/router';
import { tap } from 'rxjs/operators';
import { environment } from '../../../environments/environment';

export interface AuthTokens {
  accessToken: string;
  expiresIn: number;
  userId: string;
}

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly TOKEN_KEY = 'mirra_access_token';

  private _token = signal<string | null>(localStorage.getItem(this.TOKEN_KEY));

  readonly isAuthenticated = computed(() => !!this._token());
  readonly token = this._token.asReadonly();

  constructor(private http: HttpClient, private router: Router) {}

  login(email: string, password: string) {
    return this.http
      .post<{ data: AuthTokens }>(`${environment.apiUrl}/auth/login`, { email, password })
      .pipe(tap(res => this.setToken(res.data.accessToken)));
  }

  register(email: string, password: string) {
    return this.http
      .post<{ data: AuthTokens }>(`${environment.apiUrl}/auth/register`, { email, password })
      .pipe(tap(res => this.setToken(res.data.accessToken)));
  }

  sendVerificationCode(email: string) {
    return this.http.post<{ data: { message: string } }>(
      `${environment.apiUrl}/auth/send-code`, { email }
    );
  }

  verifyCode(email: string, code: string) {
    return this.http.post<{ data: { message: string } }>(
      `${environment.apiUrl}/auth/verify-code`, { email, code }
    );
  }

  forgotPassword(email: string) {
    return this.http.post<{ data: { message: string } }>(
      `${environment.apiUrl}/auth/forgot-password`, { email }
    );
  }

  verifyResetCode(email: string, code: string) {
    return this.http.post<{ data: { message: string } }>(
      `${environment.apiUrl}/auth/verify-reset-code`, { email, code }
    );
  }

  resetPassword(email: string, code: string, password: string) {
    return this.http.post<{ data: { message: string } }>(
      `${environment.apiUrl}/auth/reset-password`, { email, code, password }
    );
  }

  logout() {
    this._token.set(null);
    localStorage.removeItem(this.TOKEN_KEY);
    this.router.navigate(['/']);
  }

  private setToken(token: string) {
    this._token.set(token);
    localStorage.setItem(this.TOKEN_KEY, token);
  }
}
