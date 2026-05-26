// E2E — fluxos de autenticação.

import { test, expect } from "@playwright/test";
import { loginAsAdmin, loginUI, logoutUI, apiLoginAdmin, apiPost } from "./helpers/auth";
import { fullCleanup } from "./helpers/cleanup";

test.afterEach(async () => {
  await fullCleanup();
});

test("registro subsequente entra como pending e não loga", async ({ page }) => {
  await page.goto("/login");
  await page.getByRole("button", { name: /^criar conta$/i }).click();

  await page.locator('input[placeholder="Seu nome"]').fill("Maria Pendente");
  await page.locator('input[type="email"]').fill("maria@e2e.test");
  await page.locator('input[type="password"]').fill("senha-maria-123");
  await page.locator('button[type="submit"]').click();

  // Voltou para o modo "login" (sem auto-login)
  await expect(page.locator('input[placeholder="Seu nome"]')).toHaveCount(0, { timeout: 5_000 });
  await expect(page).toHaveURL(/\/login/);

  // Tentar logar agora deve falhar com 403 (conta pending)
  await page.locator('input[type="email"]').fill("maria@e2e.test");
  await page.locator('input[type="password"]').fill("senha-maria-123");
  await page.locator('button[type="submit"]').click();
  await expect(page.getByText(/aguardando aprovação/i).first()).toBeVisible({ timeout: 5_000 });
});

test("admin aprova pending e user consegue logar", async ({ page }) => {
  // Cria a pending via API
  await fetch("http://localhost:8090/api/auth/register", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name: "Pen", email: "pen@e2e.test", password: "pen-senha-123" }),
  });

  await loginAsAdmin(page);
  await page.goto("/sistema/usuarios");
  await page.getByRole("button", { name: /pendentes/i }).click();

  const row = page.locator("tr", { hasText: "Pen" });
  await expect(row).toBeVisible();
  await row.getByRole("button", { name: /^aprovar$/i }).click();

  // Após aprovar, a aba "Pendentes" fica vazia (Pen foi para "Ativos")
  await expect(page.getByRole("button", { name: /pendentes/i })).toBeVisible();

  // Logout
  await logoutUI(page);

  // Pen agora consegue logar
  await loginUI(page, "pen@e2e.test", "pen-senha-123");
});

test("troca de senha pelo ProfileModal e relogin", async ({ page }) => {
  // Cria user via API
  const token = await apiLoginAdmin();
  await apiPost(token, "/users", {
    name: "Beto", email: "beto@e2e.test", password: "senha-antiga", role: "user",
  });

  await loginUI(page, "beto@e2e.test", "senha-antiga");

  // Abre ProfileModal clicando no avatar/nome
  await page.locator('button[title="Ver perfil"]').click();
  await expect(page.getByRole("button", { name: /segurança/i })).toBeVisible();

  await page.getByRole("button", { name: /segurança/i }).click();

  await page.locator('input[autocomplete="current-password"]').fill("senha-antiga");
  await page.locator('input[autocomplete="new-password"]').first().fill("senha-nova-456");
  await page.locator('input[autocomplete="new-password"]').nth(1).fill("senha-nova-456");
  await page.getByRole("button", { name: /alterar senha/i }).click();

  // Toast de sucesso (usa .first() porque toasts duplicam visualmente)
  await expect(page.getByText(/senha alterada/i).first()).toBeVisible({ timeout: 5_000 });

  // Logout (a opção "Sair" agora vive dentro do ProfileModal, já aberto)
  await logoutUI(page);

  // Login com a nova senha
  await loginUI(page, "beto@e2e.test", "senha-nova-456");
});
