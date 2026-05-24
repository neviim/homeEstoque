// Testes da Dashboard — cards básicos e card de valor patrimonial condicional.

import { describe, it, expect, beforeAll, afterAll, afterEach, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";

import Dashboard from "../Dashboard";
import { renderWithProviders } from "@/test/render";

const seedStats = {
  total_items: 5,
  total_quantity: 12,
  total_categories: 3,
  total_locations: 2,
  total_value: 1500.50,
  recent_items: [],
  updated_items: [],
  top_categories: [],
};

function handlers(server: ReturnType<typeof setupServer>, perms: string[]) {
  server.use(
    http.get("http://localhost/api/auth/me", () =>
      HttpResponse.json({
        id: 1, name: "X", email: "x@x.com", role: "user", status: "active", permissions: perms,
      }),
    ),
    http.get("http://localhost/api/dashboard", () => HttpResponse.json(seedStats)),
  );
}

const server = setupServer();
beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

beforeEach(() => {
  localStorage.setItem("token", "tok");
  localStorage.setItem("user", JSON.stringify({ id: 1, name: "X", role: "user", permissions: [] }));
});

describe("Dashboard", () => {
  it("renderiza os 4 cards de contadores", async () => {
    handlers(server, ["dashboard.view"]);
    renderWithProviders(<Dashboard />);

    await waitFor(() => expect(screen.getByText("Itens cadastrados")).toBeInTheDocument());
    expect(screen.getByText("5")).toBeInTheDocument(); // total_items
    expect(screen.getByText("12")).toBeInTheDocument(); // total_quantity
    expect(screen.getByText("Categorias")).toBeInTheDocument();
    expect(screen.getByText("Locais")).toBeInTheDocument();
  });

  it("sem dashboard.view_value: card de valor patrimonial NÃO aparece", async () => {
    handlers(server, ["dashboard.view"]);
    renderWithProviders(<Dashboard />);

    await waitFor(() => expect(screen.getByText("Itens cadastrados")).toBeInTheDocument());
    expect(screen.queryByText(/valor patrimonial/i)).not.toBeInTheDocument();
  });

  it("com dashboard.view_value: card de valor aparece formatado em R$", async () => {
    handlers(server, ["dashboard.view", "dashboard.view_value"]);
    renderWithProviders(<Dashboard />);

    await waitFor(() => expect(screen.getByText(/valor patrimonial/i)).toBeInTheDocument());
    // R$ 1.500,50 — formato pt-BR (com espaço não-quebrável)
    expect(screen.getByText(/R\$\s*1[.,]500/)).toBeInTheDocument();
  });
});
