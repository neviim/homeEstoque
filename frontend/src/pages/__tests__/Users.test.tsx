// Testes da página Users — tabs filter, aprovação pending, badges, modal edit.

import { describe, it, expect, beforeAll, afterAll, afterEach, beforeEach, vi } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";
import toast from "react-hot-toast";

import Users from "../Users";
import { renderWithProviders } from "@/test/render";

vi.spyOn(toast, "error").mockImplementation((() => "") as never);
vi.spyOn(toast, "success").mockImplementation((() => "") as never);

const seedUsers = [
  { id: 1, name: "Admin", email: "admin@x.com", role: "admin", status: "active", created_at: "2026-01-01T00:00:00Z" },
  { id: 2, name: "Bob Pen", email: "bob@x.com", role: "user", status: "pending", created_at: "2026-01-02T00:00:00Z" },
  { id: 3, name: "Carol", email: "carol@x.com", role: "viewer", status: "inactive", created_at: "2026-01-03T00:00:00Z" },
  { id: 4, name: "Dan", email: "dan@x.com", role: "user", status: "active", created_at: "2026-01-04T00:00:00Z" },
];

const seedRoles = [
  { id: 1, name: "admin", label: "Administrador", is_system: true, user_count: 1, permissions: [] },
  { id: 2, name: "user", label: "Usuário", is_system: false, user_count: 2, permissions: [] },
  { id: 3, name: "viewer", label: "Visualizador", is_system: false, user_count: 1, permissions: [] },
];

function defaultHandlers(server: ReturnType<typeof setupServer>) {
  server.use(
    http.get("http://localhost/api/auth/me", () =>
      HttpResponse.json({
        id: 1, name: "Admin", email: "admin@x.com", role: "admin", status: "active",
        permissions: ["users.manage", "roles.manage"],
      }),
    ),
    http.get("http://localhost/api/users", () => HttpResponse.json({ users: seedUsers })),
    http.get("http://localhost/api/roles", () => HttpResponse.json({ roles: seedRoles })),
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
    permissions: ["users.manage", "roles.manage"],
  }));
});

describe("Users — lista e badges", () => {
  it("renderiza todos os usuários com badge de role e status", async () => {
    defaultHandlers(server);
    renderWithProviders(<Users />);

    await waitFor(() => expect(screen.getByText("Bob Pen")).toBeInTheDocument());
    expect(screen.getByText("Carol")).toBeInTheDocument();
    expect(screen.getByText("Dan")).toBeInTheDocument();

    // Badges de status — devem aparecer ao menos 1 vez cada
    expect(screen.getAllByText(/^ativo$/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/^pendente$/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/^inativo$/i).length).toBeGreaterThan(0);
  });
});

describe("Users — filter tabs", () => {
  it("contador de pending aparece como badge na aba", async () => {
    defaultHandlers(server);
    renderWithProviders(<Users />);

    await waitFor(() => expect(screen.getByText("Bob Pen")).toBeInTheDocument());

    // Aba "Pendentes" tem um badge com o número 1
    const pendingTab = screen.getByRole("button", { name: /pendentes/i });
    expect(within(pendingTab).getByText("1")).toBeInTheDocument();
  });

  it("clicar em 'Inativos' filtra a lista", async () => {
    defaultHandlers(server);
    renderWithProviders(<Users />);

    await waitFor(() => expect(screen.getByText("Dan")).toBeInTheDocument());

    await userEvent.setup().click(screen.getByRole("button", { name: /^inativos$/i }));
    // Após filtrar, só Carol (inactive) aparece
    expect(screen.queryByText("Dan")).not.toBeInTheDocument();
    expect(screen.queryByText("Bob Pen")).not.toBeInTheDocument();
    expect(screen.getByText("Carol")).toBeInTheDocument();
  });
});

describe("Users — aprovação pending", () => {
  it("clicar em 'Aprovar' chama PUT /users/{id}/status com active", async () => {
    let putUrl = "";
    let putBody: any = null;
    defaultHandlers(server);
    server.use(
      http.put("http://localhost/api/users/:id/status", async ({ params, request }) => {
        putUrl = String(params.id);
        putBody = await request.json();
        return HttpResponse.json({ ...seedUsers[1], status: "active" });
      }),
    );

    renderWithProviders(<Users />);
    await waitFor(() => expect(screen.getByText("Bob Pen")).toBeInTheDocument());

    // Vai pra aba Pendentes
    await userEvent.setup().click(screen.getByRole("button", { name: /pendentes/i }));

    // Clica em Aprovar
    const row = screen.getByText("Bob Pen").closest("tr")!;
    await userEvent.setup().click(within(row).getByRole("button", { name: /^aprovar$/i }));

    await waitFor(() => expect(putUrl).toBe("2"));
    expect(putBody).toEqual({ status: "active" });
  });
});

describe("Users — modal de edição", () => {
  it("abre modal e o campo email aparece como read-only", async () => {
    defaultHandlers(server);
    renderWithProviders(<Users />);

    await waitFor(() => expect(screen.getByText("Dan")).toBeInTheDocument());

    const row = screen.getByText("Dan").closest("tr")!;
    await userEvent.setup().click(within(row).getByTitle("Editar"));

    // Modal abre com campo de Nome editável
    expect(screen.getByDisplayValue("Dan")).toBeInTheDocument();
    // Aviso de email imutável aparece (com hífen e sufixo "após o cadastro")
    expect(screen.getByText(/e-mail não pode ser alterado/i)).toBeInTheDocument();
    // dan@x.com aparece tanto na tabela quanto no modal — confirma plural
    expect(screen.getAllByText("dan@x.com").length).toBeGreaterThanOrEqual(2);
  });
});
