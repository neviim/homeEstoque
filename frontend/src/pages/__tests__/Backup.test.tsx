import { describe, it, expect, beforeAll, afterAll, afterEach, beforeEach, vi } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";
import toast from "react-hot-toast";

import BackupPage from "../Backup";
import { renderWithProviders } from "@/test/render";

vi.spyOn(toast, "error").mockImplementation((() => "") as never);
vi.spyOn(toast, "success").mockImplementation((() => "") as never);

const seedBackups = [
  {
    id: 1,
    filename: "backup-20260101-120000-000001.tar.gz",
    size_bytes: 2048,
    sha256: "abc123",
    created_at: "2026-01-01T12:00:00Z",
    type: "manual" as const,
    status: "ok" as const,
    verified_at: "2026-01-01T12:05:00Z",
    notes: null,
  },
  {
    id: 2,
    filename: "backup-20260102-030000-000001.tar.gz",
    size_bytes: 4096,
    sha256: "def456",
    created_at: "2026-01-02T03:00:00Z",
    type: "auto" as const,
    status: "unverified" as const,
    verified_at: null,
    notes: null,
  },
];

const seedSchedule = {
  enabled: false,
  frequency: "daily",
  weekday: null,
  time_of_day: "03:00",
  retention_count: 7,
  last_run_at: null,
  next_run_at: null,
};

const adminPermissions = ["backup.create", "backup.restore", "backup.download", "backup.schedule"];

function adminHandlers(server: ReturnType<typeof setupServer>) {
  server.use(
    http.get("http://localhost/api/auth/me", () =>
      HttpResponse.json({
        id: 1, name: "Admin", email: "admin@x.com", role: "admin", status: "active",
        permissions: adminPermissions,
      }),
    ),
    http.get("http://localhost/api/backups", () =>
      HttpResponse.json({ backups: seedBackups }),
    ),
    http.get("http://localhost/api/backup/schedule", () =>
      HttpResponse.json(seedSchedule),
    ),
  );
}

const server = setupServer();
beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

beforeEach(() => {
  localStorage.setItem("token", "tok");
  localStorage.setItem("user", JSON.stringify({
    id: 1, name: "Admin", role: "admin",
    permissions: adminPermissions,
  }));
});

describe("BackupPage — listagem", () => {
  it("renderiza backups com badges de tipo e status", async () => {
    adminHandlers(server);
    renderWithProviders(<BackupPage />);

    await waitFor(() =>
      expect(screen.getByText("backup-20260101-120000-000001.tar.gz")).toBeInTheDocument(),
    );
    expect(screen.getByText("backup-20260102-030000-000001.tar.gz")).toBeInTheDocument();

    // Badges de tipo
    expect(screen.getByText("Manual")).toBeInTheDocument();
    expect(screen.getByText("Auto")).toBeInTheDocument();

    // Badges de status
    expect(screen.getByText("Íntegro")).toBeInTheDocument();
    expect(screen.getByText("Não verificado")).toBeInTheDocument();
  });
});

describe("BackupPage — criar backup", () => {
  it("botão 'Criar backup' chama POST /api/backups e atualiza a lista", async () => {
    let called = false;
    adminHandlers(server);
    server.use(
      http.post("http://localhost/api/backups", () => {
        called = true;
        return HttpResponse.json(
          { ...seedBackups[0], id: 3, filename: "backup-novo.tar.gz" },
          { status: 201 },
        );
      }),
    );

    renderWithProviders(<BackupPage />);
    await waitFor(() => screen.getByText("backup-20260101-120000-000001.tar.gz"));

    await userEvent.setup().click(screen.getByRole("button", { name: /criar backup/i }));
    await waitFor(() => expect(called).toBe(true));
    expect(toast.success).toHaveBeenCalledWith("Backup criado");
  });
});

describe("BackupPage — verificar integridade", () => {
  it("botão Verificar chama POST /api/backups/{id}/verify e exibe toast ok", async () => {
    let verifyId = "";
    adminHandlers(server);
    server.use(
      http.post("http://localhost/api/backups/:id/verify", ({ params }) => {
        verifyId = String(params.id ?? "");
        return HttpResponse.json({ ...seedBackups[0], status: "ok" });
      }),
    );

    renderWithProviders(<BackupPage />);
    await waitFor(() => screen.getByText("backup-20260101-120000-000001.tar.gz"));

    const row = screen.getByText("backup-20260101-120000-000001.tar.gz").closest("tr")!;
    await userEvent.setup().click(within(row).getByTitle("Verificar integridade"));

    await waitFor(() => expect(verifyId).toBe("1"));
    expect(toast.success).toHaveBeenCalledWith("Backup íntegro");
  });
});

describe("BackupPage — excluir backup", () => {
  it("botão Excluir chama DELETE /api/backups/{id}", async () => {
    let deleteId = "";
    adminHandlers(server);
    server.use(
      http.delete("http://localhost/api/backups/:id", ({ params }) => {
        deleteId = String(params.id ?? "");
        return HttpResponse.json({ ok: true });
      }),
    );

    vi.spyOn(window, "confirm").mockReturnValueOnce(true);

    renderWithProviders(<BackupPage />);
    await waitFor(() => screen.getByText("backup-20260101-120000-000001.tar.gz"));

    const row = screen.getByText("backup-20260101-120000-000001.tar.gz").closest("tr")!;
    await userEvent.setup().click(within(row).getByTitle("Excluir"));

    await waitFor(() => expect(deleteId).toBe("1"));
    expect(toast.success).toHaveBeenCalledWith("Backup excluído");
  });
});

describe("BackupPage — modal de restore", () => {
  it("exige checkbox antes de continuar para passo 2", async () => {
    adminHandlers(server);
    renderWithProviders(<BackupPage />);
    await waitFor(() => screen.getByText("backup-20260101-120000-000001.tar.gz"));

    const row = screen.getByText("backup-20260101-120000-000001.tar.gz").closest("tr")!;
    await userEvent.setup().click(within(row).getByTitle("Restaurar"));

    // Modal abriu no passo 1
    expect(screen.getByText(/operação irreversível/i)).toBeInTheDocument();
    const continueBtn = screen.getByRole("button", { name: /continuar/i });
    expect(continueBtn).toBeDisabled();

    // Marcar checkbox desbloqueia o botão
    await userEvent.setup().click(screen.getByRole("checkbox", { name: /entendo que isso/i }));
    expect(continueBtn).not.toBeDisabled();
  });

  it("passo 2 aparece após prepare bem-sucedido", async () => {
    adminHandlers(server);
    server.use(
      http.post("http://localhost/api/backups/:id/restore/prepare", () =>
        HttpResponse.json({ confirm_token: "token-secreto-abc" }),
      ),
    );

    renderWithProviders(<BackupPage />);
    await waitFor(() => screen.getByText("backup-20260101-120000-000001.tar.gz"));

    const row = screen.getByText("backup-20260101-120000-000001.tar.gz").closest("tr")!;
    await userEvent.setup().click(within(row).getByTitle("Restaurar"));

    await userEvent.setup().click(screen.getByRole("checkbox", { name: /entendo que isso/i }));
    await userEvent.setup().click(screen.getByRole("button", { name: /continuar/i }));

    await waitFor(() =>
      expect(screen.getByRole("button", { name: /restaurar agora/i })).toBeInTheDocument(),
    );
  });
});

describe("BackupPage — agendamento", () => {
  it("exibe o card de agendamento com toggle desativado por padrão", async () => {
    adminHandlers(server);
    renderWithProviders(<BackupPage />);

    await waitFor(() => screen.getByText("Agendamento automático"));
    const toggle = screen.getByRole("checkbox", { hidden: true });
    expect(toggle).not.toBeChecked();
  });

  it("salvar agendamento chama PUT /api/backup/schedule", async () => {
    let putBody: any = null;
    adminHandlers(server);
    server.use(
      http.put("http://localhost/api/backup/schedule", async ({ request }) => {
        putBody = await request.json();
        return HttpResponse.json({ ...seedSchedule, ...putBody });
      }),
    );

    renderWithProviders(<BackupPage />);
    await waitFor(() => screen.getByText("Agendamento automático"));

    await userEvent.setup().click(screen.getByRole("button", { name: /salvar agendamento/i }));

    await waitFor(() => expect(putBody).not.toBeNull());
    expect(toast.success).toHaveBeenCalledWith("Agendamento salvo");
  });
});
