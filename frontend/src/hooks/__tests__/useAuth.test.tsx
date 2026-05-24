// Testes do hook useAuth — login, register (active vs pending), logout,
// hasPermission, updateUser, refreshUser. Usa MSW para mockar /auth/* e
// React Testing Library + renderHook.

import { describe, it, expect, beforeAll, afterAll, afterEach, beforeEach, vi } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";
import { AuthProvider, useAuth } from "../useAuth";

// MSW intercepta o axios — o api.ts tem baseURL "/api", e jsdom resolve isso
// relativamente a window.location.origin (http://localhost por padrão).
const server = setupServer(
  http.get("http://localhost/api/auth/me", () =>
    HttpResponse.json({ id: 1, name: "Cached", email: "c@x.com", role: "admin", status: "active", permissions: ["items.view"] }),
  ),
);

beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
afterEach(() => {
  server.resetHandlers();
});
afterAll(() => server.close());

beforeEach(() => {
  localStorage.clear();
});

// Wrapper Provider para renderHook
function wrapper({ children }: { children: React.ReactNode }) {
  return <AuthProvider>{children}</AuthProvider>;
}

describe("useAuth — montagem inicial", () => {
  it("sem localStorage: user=null, loading vira false", async () => {
    const { result } = renderHook(() => useAuth(), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.user).toBeNull();
  });

  it("com user+token em localStorage hidrata user e faz refetch", async () => {
    localStorage.setItem("token", "old-jwt");
    localStorage.setItem(
      "user",
      JSON.stringify({ id: 1, name: "Cached", email: "c@x.com", role: "user", permissions: [] }),
    );

    const { result } = renderHook(() => useAuth(), { wrapper });
    // Hidratado já com cache antes de loading=false
    expect(result.current.user).not.toBeNull();
    expect(result.current.user?.name).toBe("Cached");

    // Após o refetch em background, role deve virar "admin" (resposta mockada)
    await waitFor(() => expect(result.current.user?.role).toBe("admin"));
    expect(result.current.user?.permissions).toContain("items.view");
  });
});

describe("useAuth — login", () => {
  it("login OK persiste token+user e atualiza estado", async () => {
    server.use(
      http.post("http://localhost/api/auth/login", () =>
        HttpResponse.json({
          token: "new-jwt",
          user: { id: 2, name: "Alice", email: "a@x.com", role: "user", status: "active", permissions: ["items.view"] },
        }),
      ),
    );

    const { result } = renderHook(() => useAuth(), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await result.current.login("a@x.com", "senha");
    });

    expect(localStorage.getItem("token")).toBe("new-jwt");
    expect(result.current.user?.name).toBe("Alice");
  });

  it("login falha: estado não muda, erro propaga", async () => {
    server.use(
      http.post("http://localhost/api/auth/login", () =>
        HttpResponse.json({ error: "credenciais inválidas" }, { status: 401 }),
      ),
    );

    const { result } = renderHook(() => useAuth(), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));

    let thrown: any;
    await act(async () => {
      try {
        await result.current.login("x@x.com", "errada");
      } catch (e) {
        thrown = e;
      }
    });

    expect(thrown).toBeTruthy();
    expect(result.current.user).toBeNull();
    // 401 limpa o token via interceptor — confirma que pelo menos isso é coerente
  });
});

describe("useAuth — register", () => {
  it("register active (primeiro user) faz auto-login", async () => {
    server.use(
      http.post("http://localhost/api/auth/register", () =>
        HttpResponse.json({
          token: "first-jwt",
          user: { id: 1, name: "Primeiro", email: "p@x.com", role: "admin", status: "active", permissions: ["users.manage"] },
        }),
      ),
    );

    const { result } = renderHook(() => useAuth(), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));

    let outcome: string | undefined;
    await act(async () => {
      outcome = await result.current.register("P", "p@x.com", "senha123");
    });

    expect(outcome).toBe("active");
    expect(localStorage.getItem("token")).toBe("first-jwt");
    expect(result.current.user?.role).toBe("admin");
  });

  it("register pending: não salva token, retorna 'pending'", async () => {
    server.use(
      http.post("http://localhost/api/auth/register", () =>
        HttpResponse.json({ status: "pending", message: "Aguarde" }, { status: 201 }),
      ),
    );

    const { result } = renderHook(() => useAuth(), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));

    let outcome: string | undefined;
    await act(async () => {
      outcome = await result.current.register("X", "x@x.com", "senha123");
    });

    expect(outcome).toBe("pending");
    expect(localStorage.getItem("token")).toBeNull();
    expect(result.current.user).toBeNull();
  });
});

describe("useAuth — logout & updateUser", () => {
  // logout assigna window.location.href, então cada teste do bloco precisa
  // restaurar o location original ao final pra não corromper o axios baseURL
  // dos próximos describes.
  let originalLocation: Location;
  beforeEach(() => {
    originalLocation = window.location;
  });
  afterEach(() => {
    Object.defineProperty(window, "location", {
      writable: true,
      configurable: true,
      value: originalLocation,
    });
  });

  it("logout limpa localStorage", async () => {
    localStorage.setItem("token", "x");
    localStorage.setItem("user", JSON.stringify({ id: 1 }));

    const { result } = renderHook(() => useAuth(), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));

    Object.defineProperty(window, "location", {
      writable: true,
      configurable: true,
      value: { ...window.location, href: "http://localhost/" },
    });

    act(() => {
      result.current.logout();
    });

    expect(localStorage.getItem("token")).toBeNull();
    expect(localStorage.getItem("user")).toBeNull();
  });

  it("updateUser persiste no localStorage e no estado", async () => {
    localStorage.setItem("token", "x");
    localStorage.setItem("user", JSON.stringify({ id: 1, name: "Old", email: "o@x.com", role: "user", permissions: [] }));

    const { result } = renderHook(() => useAuth(), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));

    act(() => {
      result.current.updateUser({ id: 1, name: "Renamed", email: "o@x.com", role: "user", permissions: [] });
    });

    expect(result.current.user?.name).toBe("Renamed");
    const stored = JSON.parse(localStorage.getItem("user")!);
    expect(stored.name).toBe("Renamed");
  });

  it("refreshUser chama /auth/me e atualiza permissions", async () => {
    let meCallCount = 0;

    // Sem cache no localStorage (sem token) — assim o useEffect do mount NÃO
    // dispara /auth/me em background e o test controla totalmente a chamada.
    const { result } = renderHook(() => useAuth(), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));

    // Agora registra o handler mockado para a chamada explícita de refreshUser
    server.use(
      http.get("http://localhost/api/auth/me", () => {
        meCallCount++;
        return HttpResponse.json({
          id: 1, name: "X", email: "x@x.com", role: "user", status: "active",
          permissions: ["items.view", "items.create"],
        });
      }),
    );
    localStorage.setItem("token", "x");

    await act(async () => {
      await result.current.refreshUser();
    });

    expect(meCallCount).toBeGreaterThan(0);
    expect(result.current.user?.permissions).toContain("items.create");
  });
});

describe("useAuth — hasPermission e isAdmin", () => {
  // Helper: registra handler /auth/me e renderiza o hook esperando o refetch
  // do mount completar com as permissions dadas.
  async function setupWithPerms(perms: string[], role = "user") {
    server.use(
      http.get("http://localhost/api/auth/me", () =>
        HttpResponse.json({
          id: 1, name: "X", email: "x@x.com", role, status: "active", permissions: perms,
        }),
      ),
    );
    localStorage.setItem("token", "x");
    localStorage.setItem("user", JSON.stringify({ id: 1, name: "X", email: "x@x.com", role, permissions: perms }));

    const { result } = renderHook(() => useAuth(), { wrapper });
    await waitFor(() => {
      expect(result.current.loading).toBe(false);
      expect(result.current.user?.permissions).toEqual(perms);
    });
    return result;
  }

  it("hasPermission true para keys nas permissions", async () => {
    const result = await setupWithPerms(["items.view", "categories.manage"]);
    expect(result.current.hasPermission("items.view")).toBe(true);
    expect(result.current.hasPermission("categories.manage")).toBe(true);
  });

  it("hasPermission false para keys ausentes", async () => {
    const result = await setupWithPerms(["items.view"]);
    expect(result.current.hasPermission("users.manage")).toBe(false);
    expect(result.current.hasPermission("inexistente")).toBe(false);
  });

  it("hasPermission false quando user é null", async () => {
    const { result } = renderHook(() => useAuth(), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.hasPermission("items.view")).toBe(false);
  });

  it("isAdmin true quando user tem roles.manage E users.manage", async () => {
    const result = await setupWithPerms(["roles.manage", "users.manage", "items.view"], "admin");
    expect(result.current.isAdmin).toBe(true);
  });

  it("isAdmin false se faltar uma das duas perms críticas", async () => {
    const result = await setupWithPerms(["users.manage"]); // sem roles.manage
    expect(result.current.isAdmin).toBe(false);
  });
});
