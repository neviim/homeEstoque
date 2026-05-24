// Setup global para Vitest:
// - matchers do jest-dom (toBeInTheDocument, toHaveClass, etc.)
// - cleanup automático entre testes (useAuth tem efeitos de localStorage)
// - força baseURL absoluta no client axios para o MSW conseguir interceptar
//   (em jsdom, baseURL relativa "/api" pode resultar em URLs que o MSW não casa)

import "@testing-library/jest-dom/vitest";
import { afterEach, beforeEach, vi } from "vitest";
import { cleanup } from "@testing-library/react";
import { api } from "@/lib/api";

api.defaults.baseURL = "http://localhost/api";

// jsdom não tem matchMedia — polyfill mínimo para evitar crash do react-hot-toast
// e qualquer outra dep que dependa dele.
if (typeof window !== "undefined" && !window.matchMedia) {
  window.matchMedia = (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  });
}

afterEach(() => {
  cleanup();
  localStorage.clear();
  // Limpa todos os timers/mocks para evitar vazamento entre testes
  vi.clearAllMocks();
});

beforeEach(() => {
  // Reset do localStorage no início de cada teste pra garantir isolamento
  localStorage.clear();
});
