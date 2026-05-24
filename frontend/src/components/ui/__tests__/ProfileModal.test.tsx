// Testes do ProfileModal — abertura, tabs, password strength bar.

import { describe, it, expect, beforeAll, afterAll, afterEach, beforeEach, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";
import toast from "react-hot-toast";

import ProfileModal from "../ProfileModal";
import { renderWithProviders } from "@/test/render";

vi.spyOn(toast, "error").mockImplementation((() => "") as never);
vi.spyOn(toast, "success").mockImplementation((() => "") as never);

const server = setupServer(
  http.get("http://localhost/api/auth/me", () =>
    HttpResponse.json({
      id: 1, name: "Maria", email: "maria@x.com", role: "user", status: "active",
      permissions: ["items.view"],
    }),
  ),
);
beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

beforeEach(() => {
  localStorage.setItem("token", "tok");
  localStorage.setItem("user", JSON.stringify({
    id: 1, name: "Maria", email: "maria@x.com", role: "user", permissions: ["items.view"],
  }));
});

describe("ProfileModal", () => {
  it("modal aberto mostra dados do user e tabs", async () => {
    renderWithProviders(<ProfileModal open={true} onClose={() => {}} />);
    // Header tem o nome
    await waitFor(() => expect(screen.getByText("Maria")).toBeInTheDocument());
    // Email aparece no header (e também no campo read-only — plural)
    expect(screen.getAllByText("maria@x.com").length).toBeGreaterThanOrEqual(1);
    // Tabs Perfil e Segurança
    expect(screen.getByRole("button", { name: /^perfil$/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /segurança/i })).toBeInTheDocument();
  });

  it("aba Perfil: email aparece como read-only com aviso", async () => {
    renderWithProviders(<ProfileModal open={true} onClose={() => {}} />);
    await waitFor(() => expect(screen.getByText("Maria")).toBeInTheDocument());

    // Tab Perfil já é o default. Verifica que NÃO há input editável de email
    const emailInputs = screen.queryAllByDisplayValue("maria@x.com");
    expect(emailInputs).toHaveLength(0);
    // Aviso de imutabilidade aparece
    expect(screen.getByText(/e-mail não pode ser alterado/i)).toBeInTheDocument();
  });

  it("aba Segurança: strength bar reage à senha digitada", async () => {
    renderWithProviders(<ProfileModal open={true} onClose={() => {}} />);
    await waitFor(() => expect(screen.getByText("Maria")).toBeInTheDocument());

    await userEvent.setup().click(screen.getByRole("button", { name: /segurança/i }));

    const newPwd = screen.getByPlaceholderText("Mínimo 6 caracteres");
    // Senha curta → fraca
    await userEvent.setup().type(newPwd, "123");
    expect(screen.getByText(/fraca/i)).toBeInTheDocument();

    // Senha média
    await userEvent.setup().clear(newPwd);
    await userEvent.setup().type(newPwd, "1234567");
    expect(screen.getByText(/média/i)).toBeInTheDocument();
  });

  it("modal fechado não renderiza nada", () => {
    const { container } = renderWithProviders(<ProfileModal open={false} onClose={() => {}} />);
    expect(container.textContent).toBe("");
  });
});
