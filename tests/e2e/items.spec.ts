// E2E — CRUD de itens (caminho crítico do produto).

import { test, expect } from "@playwright/test";
import { loginAsAdmin, apiLoginAdmin, apiPost, apiGet, apiPut } from "./helpers/auth";
import { cleanupItems } from "./helpers/cleanup";

test.beforeEach(async () => {
  await cleanupItems();
});

test("admin cria item pela UI e ele aparece na lista", async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto("/itens");
  await page.getByRole("button", { name: /novo item/i }).click();

  await page.locator('input[placeholder^="Ex: Notebook"]').fill("Furadeira E2E");
  // Submit explícito via type=submit (várias UIs têm múltiplos botões)
  await page.locator('button[type="submit"]').click();

  // Toast de sucesso
  await expect(page.getByText(/item criado/i).first()).toBeVisible({ timeout: 5_000 });
  // Item aparece na lista
  await expect(page.getByText("Furadeira E2E")).toBeVisible({ timeout: 5_000 });
  // Code gerado no formato EST-XXXXXXXX
  const codeText = await page.getByText(/EST-[A-Z0-9]{8}/).first().textContent();
  expect(codeText).toMatch(/EST-[A-Z0-9]{8}/);
});

test("editar item mudando local registra movement", async ({ page }) => {
  // Setup via API (mais rápido e robusto que UI nesse passo)
  const token = await apiLoginAdmin();
  const loc1 = await (await apiPost(token, "/locations", { name: "Local1 E2E", type: "comodo" })).json();
  const loc2 = await (await apiPost(token, "/locations", { name: "Local2 E2E", type: "comodo" })).json();
  const item = await (await apiPost(token, "/items", {
    name: "Item movível",
    location_id: loc1.id,
    quantity: 1, unit: "un", condition: "novo",
  })).json();

  // PUT direto via API muda a location (testa a regra do movement)
  const updateResp = await apiPut(token, `/items/${item.id}`, {
    ...item,
    location_id: loc2.id,
  });
  expect(updateResp.status).toBe(200);

  // Verifica via API que o movement com from→to correto foi registrado
  const movs = await apiGet<Array<{ from_location_id: number | null; to_location_id: number | null }>>(
    token, `/items/${item.id}/movements`,
  );
  const hasMove = movs.some((m) => m.from_location_id === loc1.id && m.to_location_id === loc2.id);
  expect(hasMove).toBe(true);

  // E confirma na UI que o item aparece na página de Movimentações
  await loginAsAdmin(page);
  await page.goto("/movimentacoes");
  await expect(page.getByText("Item movível").first()).toBeVisible();
});

test("excluir item remove da lista e zera o total", async ({ page }) => {
  const token = await apiLoginAdmin();
  await apiPost(token, "/items", {
    name: "Item descartável", quantity: 1, unit: "un", condition: "novo",
  });

  await loginAsAdmin(page);
  await page.goto("/itens");
  await expect(page.getByText("Item descartável")).toBeVisible();

  // O confirm() é nativo do navegador — aceita antes de clicar
  page.once("dialog", (d) => d.accept());

  const card = page.locator(".card", { hasText: "Item descartável" });
  await card.getByTitle("Excluir").click();

  // Toast de remoção
  await expect(page.getByText(/item removido/i).first()).toBeVisible({ timeout: 5_000 });

  // Total via API
  const { total } = await apiGet<{ total: number }>(token, "/items");
  expect(total).toBe(0);
});
