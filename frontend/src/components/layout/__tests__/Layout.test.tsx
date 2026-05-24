// Testes do Layout (sidebar) — verifica filtros por permissão.

import { describe, it, expect, vi, beforeAll, afterAll, afterEach, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";

import { AuthProvider } from "@/hooks/useAuth";
import Layout from "../Layout";

const server = setupServer();
beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

beforeEach(() => localStorage.clear());

// renderLayout monta Layout com user pré-hidratado em localStorage e responde
// /auth/me com as permissions dadas.
async function renderLayout(perms: string[], role = "user") {
  const user = { id: 1, name: "Alice", email: "a@x.com", role, status: "active", permissions: perms };
  server.use(
    http.get("http://localhost/api/auth/me", () => HttpResponse.json(user)),
    http.get("http://localhost/api/version", () =>
      HttpResponse.json({ running: "0.1.0", available: "0.1.0", update_available: false }),
    ),
  );
  localStorage.setItem("token", "tok");
  localStorage.setItem("user", JSON.stringify(user));

  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  });

  render(
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <MemoryRouter initialEntries={["/"]}>
          <Routes>
            <Route path="/" element={<Layout />}>
              <Route index element={<div>conteudo</div>} />
              <Route path="*" element={<div>outra rota</div>} />
            </Route>
          </Routes>
        </MemoryRouter>
      </AuthProvider>
    </QueryClientProvider>,
  );

  // Espera pelo nome do usuário aparecer (garante que loading terminou)
  await waitFor(() => expect(screen.getByText("Alice")).toBeInTheDocument());
}

describe("Layout — menu filtrado por permissão", () => {
  it("user com items.view vê 'Itens' no menu", async () => {
    await renderLayout(["items.view"]);
    expect(screen.getByText("Itens")).toBeInTheDocument();
  });

  it("sem items.view o item 'Itens' não aparece", async () => {
    await renderLayout(["dashboard.view"]);
    expect(screen.queryByText("Itens")).not.toBeInTheDocument();
  });

  it("sem categories.view o item 'Categorias' não aparece", async () => {
    await renderLayout(["dashboard.view", "items.view"]);
    expect(screen.queryByText("Categorias")).not.toBeInTheDocument();
  });

  it("botão 'Exportar CSV' só aparece com export.csv", async () => {
    await renderLayout(["items.view"]);
    expect(screen.queryByText("Exportar CSV")).not.toBeInTheDocument();
  });

  it("Exportar CSV visível dentro de Sistema com a permissão", async () => {
    await renderLayout(["items.view", "export.csv"]);
    // Está dentro do menu Sistema (colapsado por default) — expandir.
    await userEvent.click(screen.getByText("Sistema"));
    expect(screen.getByText("Exportar CSV")).toBeInTheDocument();
  });
});

describe("Layout — menu Sistema", () => {
  it("seção 'Sistema' não aparece sem users.manage e sem roles.manage", async () => {
    await renderLayout(["items.view"]);
    expect(screen.queryByText("Sistema")).not.toBeInTheDocument();
  });

  it("seção 'Sistema' aparece se houver users.manage", async () => {
    await renderLayout(["items.view", "users.manage"]);
    expect(screen.getByText("Sistema")).toBeInTheDocument();
  });

  it("submenu 'Usuários' só com users.manage, 'Permissões' só com roles.manage", async () => {
    await renderLayout(["items.view", "users.manage"]);
    // Sistema fechado por default — expandir
    const sistemaButton = screen.getByText("Sistema");
    await userEvent.click(sistemaButton);

    expect(screen.getByText("Usuários")).toBeInTheDocument();
    expect(screen.queryByText("Permissões")).not.toBeInTheDocument();
  });
});

describe("Layout — badge Visualizador", () => {
  it("mostra badge 'Visualizador' quando user.role === 'viewer'", async () => {
    await renderLayout(["dashboard.view", "items.view"], "viewer");
    expect(screen.getByText("Visualizador")).toBeInTheDocument();
  });

  it("não mostra badge para role !== viewer", async () => {
    await renderLayout(["dashboard.view"], "user");
    expect(screen.queryByText("Visualizador")).not.toBeInTheDocument();
  });
});
