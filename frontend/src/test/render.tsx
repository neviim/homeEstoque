// Helper unificado de renderização para tests de pages. Envolve o componente
// nos providers necessários (QueryClient, Router, AuthProvider) — replica o
// stack de App.tsx.

import { ReactElement, ReactNode } from "react";
import { render, RenderOptions } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { AuthProvider } from "@/hooks/useAuth";

interface ProvidersOptions {
  /** Rota inicial do MemoryRouter (default "/"). */
  route?: string;
  /** Se true, deixa o AuthProvider fora (útil para testar logout/login). */
  noAuth?: boolean;
}

function makeWrapper({ route = "/", noAuth = false }: ProvidersOptions = {}) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  });
  return function Wrapper({ children }: { children: ReactNode }) {
    const inner = (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={[route]}>{children}</MemoryRouter>
      </QueryClientProvider>
    );
    return noAuth ? inner : <AuthProvider>{inner}</AuthProvider>;
  };
}

/** Wrapper factory para usar com renderHook (retorna o componente wrapper). */
export function createWrapper(options: ProvidersOptions = {}) {
  return makeWrapper(options);
}

export function renderWithProviders(
  ui: ReactElement,
  options: ProvidersOptions & Omit<RenderOptions, "wrapper"> = {},
) {
  const { route, noAuth, ...rtOptions } = options;
  return render(ui, { wrapper: makeWrapper({ route, noAuth }), ...rtOptions });
}
