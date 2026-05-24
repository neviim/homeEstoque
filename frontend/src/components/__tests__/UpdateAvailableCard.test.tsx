import { describe, it, expect, beforeAll, afterAll, afterEach, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";

import UpdateAvailableCard from "../UpdateAvailableCard";
import { renderWithProviders } from "@/test/render";

const server = setupServer();
beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => {
  server.resetHandlers();
  localStorage.clear();
});
afterAll(() => server.close());

beforeEach(() => {
  localStorage.setItem("token", "tok");
  localStorage.setItem("user", JSON.stringify({ id: 1, name: "Admin", role: "admin", permissions: ["system.update"] }));
});

function setupVersionHandler(running = "0.1.0", available = "0.1.1", updateAvailable = true) {
  server.use(
    http.get("http://localhost/api/version", () =>
      HttpResponse.json({ running, available, update_available: updateAvailable }),
    ),
  );
}

describe("UpdateAvailableCard", () => {
  it("renderiza quando updateAvailable é true e não dispensado", async () => {
    setupVersionHandler();
    renderWithProviders(<UpdateAvailableCard />);

    await waitFor(() =>
      expect(screen.getByText("Nova versão disponível")).toBeInTheDocument(),
    );
    expect(screen.getByText("0.1.0")).toBeInTheDocument();
    expect(screen.getByText("0.1.1")).toBeInTheDocument();
  });

  it("não renderiza quando updateAvailable é false", async () => {
    setupVersionHandler("0.1.0", "0.1.0", false);
    renderWithProviders(<UpdateAvailableCard />);

    // aguarda a query resolver
    await new Promise((r) => setTimeout(r, 100));
    expect(screen.queryByText("Nova versão disponível")).not.toBeInTheDocument();
  });

  it("botão 'Lembrar depois' persiste flag e oculta card", async () => {
    setupVersionHandler();
    const { rerender } = renderWithProviders(<UpdateAvailableCard />);

    await waitFor(() =>
      expect(screen.getByText("Nova versão disponível")).toBeInTheDocument(),
    );

    await userEvent.setup().click(screen.getByRole("button", { name: /lembrar depois/i }));
    expect(localStorage.getItem("update-dismissed-0.1.1")).toBe("1");

    // Re-render deve ocultar o card
    rerender(<UpdateAvailableCard />);
    expect(screen.queryByText("Nova versão disponível")).not.toBeInTheDocument();
  });
});
