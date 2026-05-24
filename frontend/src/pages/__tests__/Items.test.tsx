// Testes da página Items — render, filtros, condicionais por permissão.

import { describe, it, expect, beforeAll, afterAll, afterEach, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";

import Items from "../Items";
import { renderWithProviders } from "@/test/render";

// Helpers MSW: respostas fake para /items, /categories, /locations e /auth/me

const seedItems = [
  { id: 1, code: "EST-ABC", name: "Furadeira", quantity: 1, unit: "un", condition: "novo", category_name: "Ferramentas", location_path: "Garagem" },
  { id: 2, code: "EST-DEF", name: "Cabo HDMI", quantity: 3, unit: "un", condition: "novo", category_name: "Cabos", location_path: "Escritório" },
];

function setupHandlers(server: ReturnType<typeof setupServer>, perms: string[]) {
  server.use(
    http.get("http://localhost/api/auth/me", () =>
      HttpResponse.json({
        id: 1, name: "X", email: "x@x.com", role: "user", status: "active", permissions: perms,
      }),
    ),
    http.get("http://localhost/api/items", () =>
      HttpResponse.json({
        items: seedItems, total: seedItems.length, page: 1, limit: 12, total_pages: 1,
      }),
    ),
    http.get("http://localhost/api/categories", () => HttpResponse.json([])),
    http.get("http://localhost/api/locations", () => HttpResponse.json([])),
  );
}

const server = setupServer();
beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

beforeEach(() => {
  localStorage.setItem("token", "tok");
  localStorage.setItem(
    "user",
    JSON.stringify({ id: 1, name: "X", role: "user", permissions: [] }),
  );
});

describe("Items — render básico", () => {
  it("renderiza itens da lista após carregar", async () => {
    setupHandlers(server, ["items.view"]);
    renderWithProviders(<Items />);

    await waitFor(() => expect(screen.getByText("Furadeira")).toBeInTheDocument());
    expect(screen.getByText("Cabo HDMI")).toBeInTheDocument();
  });

  it("mostra contador no subtitle (total)", async () => {
    setupHandlers(server, ["items.view"]);
    renderWithProviders(<Items />);
    // PageHeader subtitle: "2 itens no estoque"
    await waitFor(() => expect(screen.getByText(/2 itens no estoque/i)).toBeInTheDocument());
  });
});

describe("Items — condicionais por permissão", () => {
  it("sem items.create o botão 'Novo item' não aparece", async () => {
    setupHandlers(server, ["items.view"]);
    renderWithProviders(<Items />);

    await waitFor(() => expect(screen.getByText("Furadeira")).toBeInTheDocument());
    expect(screen.queryByRole("button", { name: /novo item/i })).not.toBeInTheDocument();
  });

  it("com items.create o botão 'Novo item' aparece", async () => {
    setupHandlers(server, ["items.view", "items.create"]);
    renderWithProviders(<Items />);

    await waitFor(() =>
      expect(screen.getByRole("button", { name: /novo item/i })).toBeInTheDocument(),
    );
  });

  it("sem items.update/delete os botões Editar/Excluir do card NÃO aparecem", async () => {
    setupHandlers(server, ["items.view"]);
    renderWithProviders(<Items />);

    await waitFor(() => expect(screen.getByText("Furadeira")).toBeInTheDocument());
    expect(screen.queryByTitle("Editar")).not.toBeInTheDocument();
    expect(screen.queryByTitle("Excluir")).not.toBeInTheDocument();
  });

  it("com items.update e items.delete os botões aparecem no card", async () => {
    setupHandlers(server, ["items.view", "items.update", "items.delete"]);
    renderWithProviders(<Items />);

    await waitFor(() => expect(screen.getByText("Furadeira")).toBeInTheDocument());
    // 2 items × 2 botões = 4
    expect(screen.getAllByTitle("Editar")).toHaveLength(2);
    expect(screen.getAllByTitle("Excluir")).toHaveLength(2);
  });
});

describe("Items — search filter", () => {
  it("digitar no search resetapaginação para 1", async () => {
    let calledWith: URL | null = null;
    server.use(
      http.get("http://localhost/api/auth/me", () =>
        HttpResponse.json({
          id: 1, name: "X", email: "x@x.com", role: "user", status: "active",
          permissions: ["items.view"],
        }),
      ),
      http.get("http://localhost/api/items", ({ request }) => {
        calledWith = new URL(request.url);
        return HttpResponse.json({ items: [], total: 0, page: 1, limit: 12, total_pages: 0 });
      }),
      http.get("http://localhost/api/categories", () => HttpResponse.json([])),
      http.get("http://localhost/api/locations", () => HttpResponse.json([])),
    );

    renderWithProviders(<Items />);
    await waitFor(() => expect(calledWith).not.toBeNull());

    const input = screen.getByPlaceholderText(/buscar por nome/i);
    await userEvent.setup().type(input, "Furadeira");

    // Aguarda o request com search query
    await waitFor(() => {
      expect(calledWith!.searchParams.get("search")).toBe("Furadeira");
      expect(calledWith!.searchParams.get("page")).toBe("1");
    });
  });
});
