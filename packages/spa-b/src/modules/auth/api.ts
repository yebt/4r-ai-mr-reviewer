import { request } from '@shared/api/client'

export interface AuthStatusResponse {
  authEnabled: boolean
  authenticated: boolean
}

export interface LoginResponse {
  authenticated: boolean
}

export interface LogoutResponse {
  authenticated: false
}

export function authStatus(): Promise<AuthStatusResponse> {
  return request<AuthStatusResponse>('GET', '/auth/status')
}

export function login(password: string): Promise<LoginResponse> {
  return request<LoginResponse>('POST', '/auth/login', { password })
}

export function logout(): Promise<LogoutResponse> {
  return request<LogoutResponse>('POST', '/auth/logout')
}
