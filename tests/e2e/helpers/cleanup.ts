// Cleanup utilities — chamadas pelos `beforeEach`/`afterEach` das specs para
// garantir isolamento sem reiniciar o backend.

import { apiDelete, apiGet, apiLoginAdmin } from "./auth";
import { E2E_ADMIN } from "../globalSetup";

interface UsersListResponse {
  users: Array<{ id: number; email: string; role: string }>;
}

interface ItemsListResponse {
  items: Array<{ id: number }>;
  total: number;
}

interface RolesListResponse {
  roles: Array<{ id: number; name: string; is_system: boolean; user_count: number }>;
}

/** Apaga todos os usuários exceto o admin do globalSetup. */
export async function cleanupUsers() {
  const token = await apiLoginAdmin();
  const { users } = await apiGet<UsersListResponse>(token, "/users");

  for (const u of users) {
    if (u.email === E2E_ADMIN.email) continue;
    // Best effort — ignora 4xx (último admin guard etc.)
    await apiDelete(token, `/users/${u.id}`);
  }
}

/** Apaga todos os items. */
export async function cleanupItems() {
  const token = await apiLoginAdmin();
  // Pega todos (limit alto)
  const { items } = await apiGet<ItemsListResponse>(token, "/items?limit=200");
  for (const it of items) {
    await apiDelete(token, `/items/${it.id}`);
  }
}

/** Apaga roles customizados (mantém os 3 system: admin, user, viewer). */
export async function cleanupCustomRoles() {
  const token = await apiLoginAdmin();
  const { roles } = await apiGet<RolesListResponse>(token, "/roles");
  for (const r of roles) {
    if (r.is_system) continue;
    if (["admin", "user", "viewer"].includes(r.name)) continue;
    if (r.user_count > 0) continue; // não dá pra apagar — só skip
    await apiDelete(token, `/roles/${r.id}`);
  }
}

/** Restaura permissões padrão dos roles seed (user e viewer), caso algum
 * teste tenha alterado. Admin é sempre re-sincronizado pelo backend. */
async function resetSeedRolePermissions() {
  const token = await apiLoginAdmin();
  const { roles } = await apiGet<RolesListResponse & { roles: Array<{ id: number; name: string }> }>(
    token, "/roles",
  );

  const USER_DEFAULT_PERMS = [
    "dashboard.view",
    "items.view", "items.create", "items.update", "items.delete", "items.upload_photo",
    "categories.view", "categories.manage",
    "locations.view", "locations.manage",
    "movements.view",
    "export.csv",
  ];
  const VIEWER_DEFAULT_PERMS = ["dashboard.view", "items.view"];

  const user = roles.find((r) => r.name === "user");
  const viewer = roles.find((r) => r.name === "viewer");
  if (user) {
    await fetch(`http://localhost:8090/api/roles/${user.id}/permissions`, {
      method: "PUT",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({ permissions: USER_DEFAULT_PERMS }),
    });
  }
  if (viewer) {
    await fetch(`http://localhost:8090/api/roles/${viewer.id}/permissions`, {
      method: "PUT",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({ permissions: VIEWER_DEFAULT_PERMS }),
    });
  }
}

/** Reset completo: items + users + custom roles + perms padrão dos seeds. */
export async function fullCleanup() {
  await cleanupItems();
  await cleanupUsers();
  await cleanupCustomRoles();
  await resetSeedRolePermissions();
}
