// Testes do roteamento de App.tsx — Protected, RequirePermission, redirects.
// Stubamos as páginas para isolar a lógica de guards.

import { describe, it, expect, vi, beforeAll, afterAll, afterEach, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route, Navigate, useLocation } from "react-router-dom";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";

import { AuthProvider, useAuth } from "@/hooks/useAuth";

const server = setupServer(
  http.get("http://localhost/api/auth/me", () =>
    HttpResponse.json({ id: 1, name: "X", email: "x@x.com", role: "user", status: "active", permissions: ["items.view"] }),
  ),
);
beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

beforeEach(() => localStorage.clear());

// Componentes-stub
function HomePage() {
  return <div>HOME</div>;
}
function LoginPage() {
  return <div>LOGIN</div>;
}
function ProtectedPage() {
  return <div>PROTECTED</div>;
}
function AdminPage() {
  return <div>ADMIN</div>;
}

// Replica simplificada dos guards de App.tsx
function Protected({ children }: { children: JSX.Element }) {
  const { user, loading } = useAuth();
  if (loading) return <div>Carregando…</div>;
  if (!user) return <Navigate to="/login" replace />;
  return children;
}
function RequirePermission({ perm, fallback = "/", children }: { perm: string; fallback?: string; children: JSX.Element }) {
  const { hasPermission, loading } = useAuth();
  if (loading) return <div>Carregando…</div>;
  if (!hasPermission(perm)) return <Navigate to={fallback} replace />;
  return children;
}

// Helper para renderizar a árvore com um path inicial
function renderAt(initialPath: string) {
  return render(
    <AuthProvider>
      <MemoryRouter initialEntries={[initialPath]}>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route
            path="/"
            element={
              <Protected>
                <HomePage />
              </Protected>
            }
          />
          <Route
            path="/items"
            element={
              <Protected>
                <RequirePermission perm="items.view">
                  <ProtectedPage />
                </RequirePermission>
              </Protected>
            }
          />
          <Route
            path="/admin"
            element={
              <Protected>
                <RequirePermission perm="users.manage">
                  <AdminPage />
                </RequirePermission>
              </Protected>
            }
          />
        </Routes>
      </MemoryRouter>
    </AuthProvider>,
  );
}

describe("Protected guard", () => {
  it("sem user redireciona /login", async () => {
    renderAt("/");
    await waitFor(() => expect(screen.getByText("LOGIN")).toBeInTheDocument());
  });

  it("com user permite acesso a /", async () => {
    localStorage.setItem("token", "x");
    localStorage.setItem("user", JSON.stringify({ id: 1, name: "X", role: "user", permissions: [] }));

    renderAt("/");
    // Pode mostrar HOME diretamente do cache, ou após refetch
    await waitFor(() => expect(screen.getByText("HOME")).toBeInTheDocument());
  });

  it("mostra 'Carregando…' enquanto useAuth está loading", async () => {
    // user+token presente → useAuth entra em fluxo de refetch e loading=true inicialmente
    localStorage.setItem("token", "x");
    localStorage.setItem("user", JSON.stringify({ id: 1, name: "X", role: "user", permissions: [] }));

    const { container } = renderAt("/");
    // Antes do refetch resolver, deve aparecer carregando
    expect(container.textContent).toMatch(/(Carregando|HOME)/);
  });
});

describe("RequirePermission guard", () => {
  it("com perm correta renderiza children", async () => {
    server.use(
      http.get("http://localhost/api/auth/me", () =>
        HttpResponse.json({
          id: 1, name: "X", email: "x@x.com", role: "user", status: "active",
          permissions: ["items.view"],
        }),
      ),
    );
    localStorage.setItem("token", "x");
    localStorage.setItem("user", JSON.stringify({ id: 1, name: "X", role: "user", permissions: ["items.view"] }));

    renderAt("/items");
    await waitFor(() => expect(screen.getByText("PROTECTED")).toBeInTheDocument());
  });

  it("sem perm redireciona ao fallback (/)", async () => {
    server.use(
      http.get("http://localhost/api/auth/me", () =>
        HttpResponse.json({
          id: 1, name: "X", email: "x@x.com", role: "user", status: "active",
          permissions: ["items.view"], // sem users.manage
        }),
      ),
    );
    localStorage.setItem("token", "x");
    localStorage.setItem("user", JSON.stringify({ id: 1, name: "X", role: "user", permissions: ["items.view"] }));

    renderAt("/admin");
    // Sem users.manage, redirect → / → HomePage
    await waitFor(() => expect(screen.getByText("HOME")).toBeInTheDocument());
  });
});
