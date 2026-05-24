// E2E — sistema de perfis e permissões (coração do diferencial do projeto).
//
// Cobertura P0:
// 1. Admin acessa /sistema/permissoes e vê os 3 roles seed + 15 perms agrupadas
// 2. Mudança de permissão pelo admin vale imediatamente no próximo /auth/me
//    do user afetado (sem precisar relogar)

import { test, expect } from "@playwright/test";
import { loginAsAdmin, apiLoginAdmin, apiPost, apiPut, apiGet } from "./helpers/auth";
import { fullCleanup } from "./helpers/cleanup";

test.beforeEach(async () => {
  // Reset antes de cada teste para garantir que permissions do role "user"
  // estão no default (caso teste anterior tenha modificado).
  await fullCleanup();
});

test.afterEach(async () => {
  await fullCleanup();
});

test("admin vê página de permissões com roles e catálogo carregados", async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto("/sistema/permissoes");

  // 3 roles seed visíveis na coluna esquerda (.first() porque o nome do role
  // aparece tanto na lista quanto no header de detalhe)
  await expect(page.getByText("Administrador").first()).toBeVisible();
  await expect(page.getByText("Usuário").first()).toBeVisible();
  await expect(page.getByText("Visualizador").first()).toBeVisible();

  // Por default seleciona o admin; banner imutável aparece
  await expect(page.getByText(/administrador tem todas as permissões/i)).toBeVisible();

  // Catálogo: pelo menos uma label do agrupamento aparece
  await expect(page.getByText(/^visualização$/i).first()).toBeVisible();
  await expect(page.getByText(/^itens$/i).first()).toBeVisible();
});

test("admin cria perfil customizado, define permissões, atribui a usuário", async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto("/sistema/permissoes");

  // Clica em "+ Novo perfil"
  await page.getByRole("button", { name: /novo perfil/i }).click();
  await page.locator('input[placeholder*="editor_de_itens"]').fill("auditor_e2e");
  await page.locator('input[placeholder*="Editor de Itens"]').fill("Auditor E2E");
  await page.locator('textarea').fill("Lê tudo, não escreve");
  await page.getByRole("button", { name: /^criar$/i }).click();

  // Perfil aparece na lista (clicado automaticamente)
  await expect(page.getByText("Auditor E2E").first()).toBeVisible({ timeout: 5_000 });

  // Configura via API (mais rápido que clicar 15 toggles)
  const token = await apiLoginAdmin();
  const { roles } = await apiGet<{ roles: Array<{ id: number; name: string }> }>(token, "/roles");
  const auditor = roles.find((r) => r.name === "auditor_e2e");
  expect(auditor).toBeTruthy();

  await apiPut(token, `/roles/${auditor!.id}/permissions`, {
    permissions: ["dashboard.view", "items.view", "categories.view", "locations.view", "movements.view"],
  });

  // Cria um usuário com esse role
  await apiPost(token, "/users", {
    name: "Audit",
    email: "audit@e2e.test",
    password: "audit-senha-123",
    role: "auditor_e2e",
  });

  // Loga como ele e verifica que só Dashboard e Itens aparecem no menu
  await page.getByRole("button", { name: "Sair" }).click();
  await page.locator('input[type="email"]').fill("audit@e2e.test");
  await page.locator('input[type="password"]').fill("audit-senha-123");
  await page.locator('button[type="submit"]').click();
  await expect(page).toHaveURL(/\/$|\/itens/);

  // Botão "Novo item" NÃO aparece (sem items.create)
  await page.goto("/itens");
  await expect(page.getByRole("button", { name: /novo item/i })).toHaveCount(0);

  // Menu Sistema NÃO aparece (sem users.manage / roles.manage)
  await expect(page.getByText(/^sistema$/i)).toHaveCount(0);
});

test("mudança de permissão vale imediatamente sem relogar", async ({ page }) => {
  // Cria user com role "user" (que tem items.create)
  const token = await apiLoginAdmin();
  await apiPost(token, "/users", {
    name: "Joao", email: "joao@e2e.test", password: "joao-senha-123", role: "user",
  });

  // Joao loga e vê "Novo item"
  await page.goto("/login");
  await page.locator('input[type="email"]').fill("joao@e2e.test");
  await page.locator('input[type="password"]').fill("joao-senha-123");
  await page.locator('button[type="submit"]').click();
  await expect(page).toHaveURL(/\/$|\/itens/);

  await page.goto("/itens");
  await expect(page.getByRole("button", { name: /novo item/i })).toBeVisible();

  // Admin (em "background") remove items.create do perfil user
  const { roles } = await apiGet<{ roles: Array<{ id: number; name: string; permissions: string[] }> }>(
    token, "/roles",
  );
  const userRole = roles.find((r) => r.name === "user")!;
  const newPerms = userRole.permissions.filter((p) => p !== "items.create");
  await apiPut(token, `/roles/${userRole.id}/permissions`, { permissions: newPerms });

  // Joao tenta criar item — backend deve recusar (toast Acesso negado)
  // Clica no botão antes de o frontend saber da mudança
  await page.getByRole("button", { name: /novo item/i }).click();
  await page.locator('input[placeholder^="Ex: Notebook"]').fill("Item bloqueado");
  await page.locator('button[type="submit"]').click();

  await expect(page.getByText(/acesso negado/i).first()).toBeVisible({ timeout: 5_000 });

  // Refresh: o botão some agora que /auth/me devolve permissions atualizadas
  await page.reload();
  await expect(page.getByRole("button", { name: /novo item/i })).toHaveCount(0, { timeout: 10_000 });
});
