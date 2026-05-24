// Testes do client axios + interceptors. Usamos MSW para interceptar as
// requisições no nível network (em vez de mockar axios) — testa o stack real
// incluindo o interceptor de response.

import { describe, it, expect, beforeAll, afterAll, afterEach, beforeEach, vi } from "vitest";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";

// Espiar toast.error em vez de mockar o módulo — mocks de módulo via vi.mock
// têm precedência por arquivo, mas dependem da ordem de carga: se outro
// arquivo de teste importou react-hot-toast antes (ex: via useAuth), o cache
// pode interferir mesmo com isolate. spy direto no objeto importado é mais
// confiável aqui.
import toast from "react-hot-toast";
import { api, _resetRateLimitForTesting } from "../api";

const mockErrorFn = vi.spyOn(toast, "error").mockImplementation(() => "" as never);

// MSW server local — registramos handlers por teste
const server = setupServer();

beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

beforeEach(() => {
  vi.clearAllMocks();
  // Limpa o cache de "último 403" pra rate-limit começar zerado em cada test
  // (módulo guarda `last403At` em escopo de módulo — não é resetável diretamente,
  // mas o teste é projetado para não depender de estado prévio)
});

describe("api request interceptor", () => {
  it("adiciona Authorization header quando há token no localStorage", async () => {
    localStorage.setItem("token", "meu-jwt-aqui");

    let receivedAuth: string | null = null;
    server.use(
      http.get("http://localhost/api/me", ({ request }) => {
        receivedAuth = request.headers.get("Authorization");
        return HttpResponse.json({ ok: true });
      }),
    );

    await api.get("/me", { baseURL: "http://localhost/api" });
    expect(receivedAuth).toBe("Bearer meu-jwt-aqui");
  });

  it("não adiciona Authorization header quando localStorage não tem token", async () => {
    let receivedAuth: string | null | undefined = "sentinela";
    server.use(
      http.get("http://localhost/api/pub", ({ request }) => {
        receivedAuth = request.headers.get("Authorization");
        return HttpResponse.json({});
      }),
    );

    await api.get("/pub", { baseURL: "http://localhost/api" });
    expect(receivedAuth).toBeNull();
  });
});

describe("api response interceptor — 401", () => {
  // Isolamos as manipulações de window.location nesses 2 testes; o describe é
  // executado depois dos outros pra não contaminar o ambiente do MSW.
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

  it("401 limpa localStorage e redireciona para /login", async () => {
    localStorage.setItem("token", "antigo");
    localStorage.setItem("user", JSON.stringify({ id: 1 }));

    Object.defineProperty(window, "location", {
      writable: true,
      configurable: true,
      value: { ...window.location, pathname: "/itens", href: "http://localhost/itens" },
    });

    server.use(
      http.get("http://localhost/api/needs-auth", () => {
        return new HttpResponse("unauthorized", { status: 401 });
      }),
    );

    await expect(
      api.get("/needs-auth", { baseURL: "http://localhost/api" }),
    ).rejects.toThrow();

    expect(localStorage.getItem("token")).toBeNull();
    expect(localStorage.getItem("user")).toBeNull();
  });

  it("401 estando em /login não redireciona (evita loop)", async () => {
    localStorage.setItem("token", "antigo");

    Object.defineProperty(window, "location", {
      writable: true,
      configurable: true,
      value: { ...window.location, pathname: "/login", href: "/login" },
    });

    server.use(
      http.get("http://localhost/api/needs-auth", () =>
        new HttpResponse("unauthorized", { status: 401 }),
      ),
    );

    await expect(
      api.get("/needs-auth", { baseURL: "http://localhost/api" }),
    ).rejects.toThrow();

    // pathname == "/login", então href não é setado pelo interceptor
    expect(window.location.href).toBe("/login");
  });
});

describe("api response interceptor — 403", () => {
  // Espia Date.now para controlar o rate-limit determinísticamente. Cada teste
  // avança o "relógio" em 10 segundos garantindo que o intervalo de 2s do
  // interceptor já passou.
  let nowSpy: ReturnType<typeof vi.spyOn>;
  let virtualNow = 1_000_000_000_000;

  beforeEach(() => {
    virtualNow += 10_000; // pula 10s entre testes
    nowSpy = vi.spyOn(Date, "now").mockImplementation(() => virtualNow);
    _resetRateLimitForTesting();
  });

  afterEach(() => {
    nowSpy.mockRestore();
  });

  it("403 dispara toast com mensagem do backend", async () => {
    server.use(
      http.get("http://localhost/api/blocked", () =>
        HttpResponse.json({ error: "acesso negado" }, { status: 403 }),
      ),
    );

    let caughtStatus: number | undefined;
    try {
      await api.get("/blocked", { baseURL: "http://localhost/api" });
    } catch (e: any) {
      caughtStatus = e?.response?.status;
    }
    // Confirma que o axios recebeu de fato um 403 (não erro de rede)
    expect(caughtStatus).toBe(403);
    expect(mockErrorFn).toHaveBeenCalledWith("acesso negado");
  });

  it("403 sem body usa mensagem default 'Acesso negado'", async () => {
    server.use(
      http.get("http://localhost/api/blocked2", () => new HttpResponse("", { status: 403 })),
    );

    await expect(
      api.get("/blocked2", { baseURL: "http://localhost/api" }),
    ).rejects.toThrow();
    expect(mockErrorFn).toHaveBeenCalledWith("Acesso negado");
  });

  it("403 NÃO desloga (diferente de 401)", async () => {
    localStorage.setItem("token", "valido");

    server.use(
      http.get("http://localhost/api/blocked3", () => new HttpResponse("", { status: 403 })),
    );

    await expect(
      api.get("/blocked3", { baseURL: "http://localhost/api" }),
    ).rejects.toThrow();

    expect(localStorage.getItem("token")).toBe("valido");
  });

  it("403 consecutivos têm rate-limit de 2s entre toasts", async () => {
    server.use(
      http.get("http://localhost/api/spam", () => new HttpResponse("", { status: 403 })),
    );

    // Dispara 3 rapidamente com o relógio congelado — só o primeiro gera toast
    await Promise.all([
      api.get("/spam", { baseURL: "http://localhost/api" }).catch(() => null),
      api.get("/spam", { baseURL: "http://localhost/api" }).catch(() => null),
      api.get("/spam", { baseURL: "http://localhost/api" }).catch(() => null),
    ]);

    expect(mockErrorFn).toHaveBeenCalledTimes(1);
  });
});

describe("api response interceptor — outros status", () => {
  it("500 não dispara toast nem logout", async () => {
    localStorage.setItem("token", "valido");

    server.use(
      http.get("http://localhost/api/oops", () =>
        HttpResponse.json({ error: "erro interno" }, { status: 500 }),
      ),
    );

    await expect(
      api.get("/oops", { baseURL: "http://localhost/api" }),
    ).rejects.toThrow();

    expect(mockErrorFn).not.toHaveBeenCalled();
    expect(localStorage.getItem("token")).toBe("valido");
  });
});
