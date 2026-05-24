// Testes da página Permissions — lista roles, grid de perms, toggle, admin lock.

import { describe, it, expect, beforeAll, afterAll, afterEach, beforeEach, vi } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";
import toast from "react-hot-toast";

import Permissions from "../Permissions";
import { renderWithProviders } from "@/test/render";

vi.spyOn(toast, "error").mockImplementation((() => "") as never);
vi.spyOn(toast, "success").mockImplementation((() => "") as never);

const seedRoles = [
  {
    id: 1, name: "admin", label: "Administrador", description: "tudo",
    is_system: true, user_count: 1,
    permissions: ["dashboard.view", "items.view", "items.create", "users.manage", "roles.manage"],
    created_at: "2026-01-01T00:00:00Z",
  },
  {
    id: 2, name: "user", label: "Usuário", description: "padrão",
    is_system: false, user_count: 3,
    permissions: ["dashboard.view", "items.view", "items.create"],
    created_at: "2026-01-01T00:00:00Z",
  },
  {
    id: 3, name: "viewer", label: "Visualizador", description: "só lê",
    is_system: false, user_count: 0,
    permissions: ["dashboard.view", "items.view"],
    created_at: "2026-01-01T00:00:00Z",
  },
];

const seedCatalog = [
  { key: "dashboard.view", label: "Ver Dashboard", description: "...", category: "Visualização" },
  { key: "items.view", label: "Ver Itens", description: "...", category: "Itens" },
  { key: "items.create", label: "Criar Itens", description: "...", category: "Itens" },
  { key: "users.manage", label: "Gerenciar Usuários", description: "...", category: "Sistema" },
  { key: "roles.manage", label: "Gerenciar Perfis", description: "...", category: "Sistema" },
];

function defaultHandlers(server: ReturnType<typeof setupServer>) {
  server.use(
    http.get("http://localhost/api/auth/me", () =>
      HttpResponse.json({
        id: 1, name: "Admin", email: "a@x.com", role: "admin", status: "active",
        permissions: ["roles.manage", "users.manage"],
      }),
    ),
    http.get("http://localhost/api/roles", () => HttpResponse.json({ roles: seedRoles })),
    http.get("http://localhost/api/permissions", () => HttpResponse.json({ permissions: seedCatalog })),
  );
}

const server = setupServer();
beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

beforeEach(() => {
  localStorage.setItem("token", "tok");
  localStorage.setItem("user", JSON.stringify({
    id: 1, name: "Admin", role: "admin",
    permissions: ["roles.manage", "users.manage"],
  }));
});

describe("Permissions — render", () => {
  it("lista os 3 roles seed na coluna esquerda", async () => {
    defaultHandlers(server);
    renderWithProviders(<Permissions />);

    await waitFor(() => expect(screen.getByText("Administrador")).toBeInTheDocument());
    expect(screen.getByText("Usuário")).toBeInTheDocument();
    expect(screen.getByText("Visualizador")).toBeInTheDocument();
  });

  it("mostra categorias do catálogo agrupadas", async () => {
    defaultHandlers(server);
    renderWithProviders(<Permissions />);

    await waitFor(() => expect(screen.getByText("Administrador")).toBeInTheDocument());
    // .first() porque "Sistema" também aparece no nome do menu raiz
    expect(screen.getAllByText(/^visualização$/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/^itens$/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/^sistema$/i).length).toBeGreaterThan(0);
  });
});

describe("Permissions — admin lock", () => {
  it("admin selecionado mostra banner 'tem todas as permissões'", async () => {
    defaultHandlers(server);
    renderWithProviders(<Permissions />);

    // Admin é o primeiro role e é selecionado automaticamente
    await waitFor(() =>
      expect(screen.getByText(/administrador tem todas as permissões/i)).toBeInTheDocument(),
    );
  });

  it("botão Salvar fica desabilitado para admin", async () => {
    defaultHandlers(server);
    renderWithProviders(<Permissions />);

    await waitFor(() =>
      expect(screen.getByText(/administrador tem todas/i)).toBeInTheDocument(),
    );
    const saveBtn = screen.getByRole("button", { name: /salvar/i });
    expect(saveBtn).toBeDisabled();
  });
});

describe("Permissions — toggle e Save", () => {
  it("toggle marca/desmarca permissão e habilita Save (role user)", async () => {
    defaultHandlers(server);
    renderWithProviders(<Permissions />);

    await waitFor(() => expect(screen.getByText("Usuário")).toBeInTheDocument());

    // Clica no role "Usuário" para selecioná-lo
    const userRoleBtn = screen.getByText("Usuário").closest("button")!;
    await userEvent.setup().click(userRoleBtn);

    // Botão Salvar inicial está desabilitado (sem dirty)
    const save = screen.getByRole("button", { name: /salvar/i });
    expect(save).toBeDisabled();

    // Clica em algum switch — pega o primeiro que não está marcado
    // O perfil "user" tem 3 perms; total catalog = 5. Vamos clicar no toggle
    // de "Gerenciar Usuários" (não está nas suas perms).
    const usersManageRow = screen.getByText("Gerenciar Usuários").closest("li")!;
    const toggle = within(usersManageRow).getByRole("switch");
    await userEvent.setup().click(toggle);

    // Agora dirty = true → Salvar habilitado
    await waitFor(() => expect(save).toBeEnabled());
  });
});

describe("Permissions — criar role", () => {
  it("abre modal e cria perfil novo via POST /roles", async () => {
    let postBody: any = null;
    defaultHandlers(server);
    server.use(
      http.post("http://localhost/api/roles", async ({ request }) => {
        postBody = await request.json();
        return HttpResponse.json(
          {
            id: 4, name: postBody.name, label: postBody.label, description: postBody.description,
            is_system: false, user_count: 0, permissions: [], created_at: "2026-01-01T00:00:00Z",
          },
          { status: 201 },
        );
      }),
    );

    renderWithProviders(<Permissions />);
    await waitFor(() => expect(screen.getByText("Administrador")).toBeInTheDocument());

    await userEvent.setup().click(screen.getByRole("button", { name: /novo perfil/i }));
    // Modal abre
    expect(screen.getByPlaceholderText(/editor_de_itens/i)).toBeInTheDocument();

    await userEvent.setup().type(screen.getByPlaceholderText(/editor_de_itens/i), "auditor");
    await userEvent.setup().type(screen.getByPlaceholderText(/Editor de Itens/i), "Auditor");
    await userEvent.setup().click(screen.getByRole("button", { name: /^criar$/i }));

    await waitFor(() => expect(postBody).not.toBeNull());
    expect(postBody.name).toBe("auditor");
    expect(postBody.label).toBe("Auditor");
  });
});
