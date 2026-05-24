// Testes de Categories — render, botões condicionais por permissão.

import { describe, it, expect, beforeAll, afterAll, afterEach, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";

import Categories from "../Categories";
import { renderWithProviders } from "@/test/render";

const seedCats = [
  { id: 1, name: "Eletrônicos", color: "#3b82f6", item_count: 5 },
  { id: 2, name: "Ferramentas", color: "#f59e0b", item_count: 3 },
];

function setupServerHandlers(server: ReturnType<typeof setupServer>, perms: string[]) {
  server.use(
    http.get("http://localhost/api/auth/me", () =>
      HttpResponse.json({
        id: 1, name: "X", email: "x@x.com", role: "user", status: "active", permissions: perms,
      }),
    ),
    http.get("http://localhost/api/categories", () => HttpResponse.json(seedCats)),
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

describe("Categories", () => {
  it("renderiza categorias com item_count", async () => {
    setupServerHandlers(server, ["categories.view"]);
    renderWithProviders(<Categories />);

    await waitFor(() => expect(screen.getByText("Eletrônicos")).toBeInTheDocument());
    expect(screen.getByText("Ferramentas")).toBeInTheDocument();
    expect(screen.getByText("5 itens")).toBeInTheDocument();
    expect(screen.getByText("3 itens")).toBeInTheDocument();
  });

  it("sem categories.manage: botão 'Nova categoria' NÃO aparece", async () => {
    setupServerHandlers(server, ["categories.view"]);
    renderWithProviders(<Categories />);

    await waitFor(() => expect(screen.getByText("Eletrônicos")).toBeInTheDocument());
    expect(screen.queryByRole("button", { name: /nova categoria/i })).not.toBeInTheDocument();
  });

  it("com categories.manage: botão 'Nova categoria' aparece", async () => {
    setupServerHandlers(server, ["categories.view", "categories.manage"]);
    renderWithProviders(<Categories />);

    await waitFor(() =>
      expect(screen.getByRole("button", { name: /nova categoria/i })).toBeInTheDocument(),
    );
  });
});
