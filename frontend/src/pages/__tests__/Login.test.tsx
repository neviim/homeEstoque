// Testes da página de Login — toggle login/register, eye toggle, fluxos
// completos de submit (success / pending / erro).

import { describe, it, expect, beforeAll, afterAll, afterEach, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";
import toast from "react-hot-toast";

import Login from "../Login";
import { renderWithProviders } from "@/test/render";

// Espia toast.error/success para não precisar montar Toaster (que tem deps de
// matchMedia/animation já polyfilladas mas dão warnings).
const toastErrorSpy = vi.spyOn(toast, "error").mockImplementation((() => "") as never);
const toastSuccessSpy = vi.spyOn(toast, "success").mockImplementation((() => "") as never);

const server = setupServer();
beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => {
  server.resetHandlers();
  toastErrorSpy.mockClear();
  toastSuccessSpy.mockClear();
});
afterAll(() => server.close());

function renderLogin() {
  return renderWithProviders(<Login />, { route: "/login" });
}

// Helper para encontrar o botão submit (há 2 botões "Entrar" + "Criar conta"
// na tela — toggle e submit; usamos type="submit").
function submitButton(): HTMLButtonElement {
  const btns = screen.getAllByRole("button");
  const submit = btns.find((b) => b.getAttribute("type") === "submit");
  if (!submit) throw new Error("submit button não encontrado");
  return submit as HTMLButtonElement;
}

describe("Login — modo toggle", () => {
  it("modo login não mostra campo Nome", () => {
    renderLogin();
    expect(screen.queryByPlaceholderText("Seu nome")).not.toBeInTheDocument();
    expect(screen.getByPlaceholderText("voce@exemplo.com")).toBeInTheDocument();
  });

  it("clicar em 'Criar conta' mostra campo Nome", async () => {
    renderLogin();
    const user = userEvent.setup();
    // O toggle tab "Criar conta" vem antes do submit (que também é "Criar conta"
    // após toggle). Pega o primeiro.
    await user.click(screen.getAllByRole("button", { name: /criar conta/i })[0]);
    expect(screen.getByPlaceholderText("Seu nome")).toBeInTheDocument();
  });
});

describe("Login — eye toggle de senha", () => {
  it("alterna o type do input de password para text", async () => {
    renderLogin();
    const user = userEvent.setup();

    const pwd = screen.getByPlaceholderText("Mínimo 6 caracteres") as HTMLInputElement;
    expect(pwd.type).toBe("password");

    const eyeBtn = pwd.parentElement!.querySelector('button[tabindex="-1"]') as HTMLElement;
    await user.click(eyeBtn);
    expect(pwd.type).toBe("text");

    await user.click(eyeBtn);
    expect(pwd.type).toBe("password");
  });
});

describe("Login — submit", () => {
  it("login bem-sucedido persiste user e dispara toast de sucesso", async () => {
    server.use(
      http.post("http://localhost/api/auth/login", () =>
        HttpResponse.json({
          token: "jwt-ok",
          user: { id: 1, name: "Alice", email: "a@x.com", role: "admin", status: "active", permissions: ["items.view"] },
        }),
      ),
    );

    renderLogin();
    const user = userEvent.setup();
    await user.type(screen.getByPlaceholderText("voce@exemplo.com"), "a@x.com");
    await user.type(screen.getByPlaceholderText("Mínimo 6 caracteres"), "senha123");
    await user.click(submitButton());

    await waitFor(() => expect(localStorage.getItem("token")).toBe("jwt-ok"));
    expect(toastSuccessSpy).toHaveBeenCalledWith("Bem-vindo!");
  });

  it("login com credenciais inválidas dispara toast de erro", async () => {
    server.use(
      http.post("http://localhost/api/auth/login", () =>
        HttpResponse.json({ error: "credenciais inválidas" }, { status: 401 }),
      ),
    );

    renderLogin();
    const user = userEvent.setup();
    await user.type(screen.getByPlaceholderText("voce@exemplo.com"), "a@x.com");
    await user.type(screen.getByPlaceholderText("Mínimo 6 caracteres"), "errada");
    await user.click(submitButton());

    await waitFor(() =>
      expect(toastErrorSpy).toHaveBeenCalledWith("credenciais inválidas"),
    );
    expect(localStorage.getItem("token")).toBeNull();
  });

  it("register em modo pending mostra toast e volta para login (sem auto-login)", async () => {
    server.use(
      http.post("http://localhost/api/auth/register", () =>
        HttpResponse.json({ status: "pending", message: "Aguarde" }, { status: 201 }),
      ),
    );

    renderLogin();
    const user = userEvent.setup();
    // Toggle para criar conta
    await user.click(screen.getAllByRole("button", { name: /criar conta/i })[0]);

    await user.type(screen.getByPlaceholderText("Seu nome"), "Maria");
    await user.type(screen.getByPlaceholderText("voce@exemplo.com"), "m@x.com");
    await user.type(screen.getByPlaceholderText("Mínimo 6 caracteres"), "senha123");
    await user.click(submitButton());

    // Toast de pending dispara
    await waitFor(() =>
      expect(toastSuccessSpy).toHaveBeenCalledWith(
        expect.stringMatching(/aguarde aprovação/i),
        expect.any(Object),
      ),
    );
    // Voltou para modo login (Nome desapareceu)
    await waitFor(() =>
      expect(screen.queryByPlaceholderText("Seu nome")).not.toBeInTheDocument(),
    );
    // Sem token salvo
    expect(localStorage.getItem("token")).toBeNull();
  });
});
