// Helpers de autenticação reutilizados pelas specs.

import { Page, expect } from "@playwright/test";
import { E2E_ADMIN } from "../globalSetup";

const API = "http://localhost:8090/api";

/** Faz login via UI. Aguarda a navegação para a home. */
export async function loginUI(page: Page, email: string, password: string) {
  await page.goto("/login");
  // Login.tsx não associa <label htmlFor> aos inputs, então mira no type
  await page.locator('input[type="email"]').fill(email);
  await page.locator('input[type="password"]').fill(password);
  // 2 botões "Entrar" na tela: toggle de modo + submit. Usa submit explícito.
  await page.locator('button[type="submit"]').click();
  // Sucesso → navega para fora de /login
  await page.waitForURL((url) => !url.pathname.startsWith("/login"), { timeout: 10_000 });
}

/** Atalho: login como o admin criado pelo globalSetup. */
export async function loginAsAdmin(page: Page) {
  await loginUI(page, E2E_ADMIN.email, E2E_ADMIN.password);
}

/** Faz logout via UI: abre o ProfileModal (se necessário) e clica em "Sair do sistema". */
export async function logoutUI(page: Page) {
  const sair = page.getByRole("button", { name: /sair do sistema/i });
  if (!(await sair.isVisible().catch(() => false))) {
    await page.locator('button[title="Ver perfil"]').click();
  }
  await sair.click();
  await page.waitForURL(/\/login/, { timeout: 10_000 });
}

/** Faz login via API e devolve o token (útil pra setup rápido sem UI). */
export async function apiLogin(email: string, password: string): Promise<string> {
  const r = await fetch(`${API}/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  if (!r.ok) throw new Error(`login api failed: ${r.status}`);
  const body = await r.json();
  return body.token;
}

export async function apiLoginAdmin(): Promise<string> {
  return apiLogin(E2E_ADMIN.email, E2E_ADMIN.password);
}

/** Faz POST autenticado. */
export async function apiPost(token: string, path: string, body: unknown): Promise<Response> {
  return fetch(`${API}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
    body: JSON.stringify(body),
  });
}

export async function apiPut(token: string, path: string, body: unknown): Promise<Response> {
  return fetch(`${API}${path}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
    body: JSON.stringify(body),
  });
}

export async function apiDelete(token: string, path: string): Promise<Response> {
  return fetch(`${API}${path}`, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
  });
}

export async function apiGet<T = any>(token: string, path: string): Promise<T> {
  const r = await fetch(`${API}${path}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!r.ok) throw new Error(`GET ${path}: ${r.status}`);
  return r.json();
}
