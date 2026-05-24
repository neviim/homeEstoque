// Roda UMA vez antes de qualquer test. Não apaga o DB (causaria estado
// inconsistente quando o backend reusa entre runs). Apenas garante que o
// admin de teste exista e está com a senha esperada. Cleanup de dados de
// teste é feito por beforeEach de cada spec via API.

const API_BASE = "http://localhost:8090/api";

export const E2E_ADMIN = {
  name: "Admin E2E",
  email: "admin@e2e.test",
  password: "admin123",
};

export default async function globalSetup() {
  await waitForHealth();

  // Verifica se o admin já existe e funciona
  const probe = await fetch(`${API_BASE}/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email: E2E_ADMIN.email, password: E2E_ADMIN.password }),
  });
  if (probe.ok) return;

  // Não existe ainda — tenta registrar (vira admin se for o primeiro humano)
  const reg = await fetch(`${API_BASE}/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(E2E_ADMIN),
  });
  if (!reg.ok && reg.status !== 409) {
    throw new Error(`globalSetup register falhou: ${reg.status} ${await reg.text()}`);
  }

  // Se virou pending (DB tinha admin de outro setup), promove via PUT /users/{id}/status
  // — mas precisamos de outro admin pra isso. Como fallback, esperamos que o usuário
  // execute manualmente uma limpeza do DB ou que esta seja a primeira run.
  const body = reg.ok ? await reg.json() : null;
  if (body?.status === "pending") {
    throw new Error(
      "DB tem outro admin pré-existente e o admin E2E ficou pending. " +
      "Apague /tmp/homeestoque-e2e.db e rode novamente (com o backend parado).",
    );
  }
  if (body && body.user?.role !== "admin") {
    throw new Error(`globalSetup: novo user deveria ser admin, got role=${body.user?.role}`);
  }
}

async function waitForHealth() {
  const deadline = Date.now() + 60_000;
  while (Date.now() < deadline) {
    try {
      const r = await fetch("http://localhost:8090/health");
      if (r.ok) return;
    } catch {
      /* still booting */
    }
    await new Promise((res) => setTimeout(res, 500));
  }
  throw new Error("backend não respondeu em /health dentro de 60s");
}
