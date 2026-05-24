// Smoke do servidor MCP via JSON-RPC stdio.
//
// Não usa Playwright browser — spawna o binário e troca mensagens diretas
// para validar:
// - initialize responde com serverInfo.name = "homeestoque"
// - tools/list retorna 10 tools
// - find_item_location resolve via fuzzy
//
// IMPORTANTE: o binário tem que estar compilado em /bin/homeestoque-mcp.
// Em CI futuro, o pipeline roda `./tools/build-mcp.sh` antes destes testes.

import { test, expect } from "@playwright/test";
import { spawn, ChildProcessWithoutNullStreams } from "child_process";
import { resolve } from "path";
import { existsSync } from "fs";

import { apiLoginAdmin, apiPost } from "./helpers/auth";

const ROOT = resolve(__dirname, "../..");
const BIN = resolve(ROOT, "bin/homeestoque-mcp");
const TEST_DB = "/tmp/homeestoque-e2e.db";

test.beforeAll(() => {
  if (!existsSync(BIN)) {
    throw new Error(`binário não encontrado em ${BIN} — rode tools/build-mcp.sh`);
  }
});

/**
 * Pequeno cliente JSON-RPC para falar com o binário via stdio.
 * Resolve cada request por id e expõe `call(method, params)`.
 */
class MCPClient {
  private proc: ChildProcessWithoutNullStreams;
  private buf = "";
  private pending = new Map<number, (v: any) => void>();
  private nextId = 1;

  constructor(dbPath: string) {
    this.proc = spawn(BIN, [], {
      env: { ...process.env, DB_PATH: dbPath },
      stdio: ["pipe", "pipe", "pipe"],
    });

    this.proc.stdout.on("data", (chunk: Buffer) => {
      this.buf += chunk.toString("utf8");
      let nl: number;
      // O protocolo MCP usa JSON Lines (uma mensagem JSON por linha)
      while ((nl = this.buf.indexOf("\n")) >= 0) {
        const line = this.buf.slice(0, nl);
        this.buf = this.buf.slice(nl + 1);
        if (!line.trim()) continue;
        try {
          const msg = JSON.parse(line);
          if (typeof msg.id === "number" && this.pending.has(msg.id)) {
            this.pending.get(msg.id)!(msg);
            this.pending.delete(msg.id);
          }
        } catch {
          /* mensagem incompleta ou inválida — ignora */
        }
      }
    });

    this.proc.stderr.on("data", () => {
      // logs do servidor MCP — apenas para debug se precisar
    });
  }

  async call(method: string, params: any = {}, timeoutMs = 5000): Promise<any> {
    const id = this.nextId++;
    const req = { jsonrpc: "2.0", id, method, params };
    this.proc.stdin.write(JSON.stringify(req) + "\n");

    return new Promise((res, rej) => {
      this.pending.set(id, res);
      setTimeout(() => {
        if (this.pending.has(id)) {
          this.pending.delete(id);
          rej(new Error(`timeout esperando resposta de ${method}`));
        }
      }, timeoutMs);
    });
  }

  close() {
    this.proc.stdin.end();
    this.proc.kill();
  }
}

async function initialize(client: MCPClient) {
  return client.call("initialize", {
    protocolVersion: "2025-06-18",
    capabilities: {},
    clientInfo: { name: "e2e-test", version: "1.0.0" },
  });
}

test("MCP responde initialize com serverInfo.name = 'homeestoque'", async () => {
  const client = new MCPClient(TEST_DB);
  try {
    const resp = await initialize(client);
    expect(resp.result?.serverInfo?.name).toBe("homeestoque");
  } finally {
    client.close();
  }
});

test("MCP tools/list retorna 10 tools incluindo find_item_location", async () => {
  const client = new MCPClient(TEST_DB);
  try {
    await initialize(client);
    // Após initialize, envia 'initialized' notification (sem id)
    client["proc"].stdin.write(JSON.stringify({ jsonrpc: "2.0", method: "notifications/initialized" }) + "\n");

    const resp = await client.call("tools/list");
    const tools: Array<{ name: string }> = resp.result?.tools || [];
    expect(tools.length).toBe(10);

    const names = tools.map((t) => t.name);
    expect(names).toContain("find_item_location");
    expect(names).toContain("list_items");
    expect(names).toContain("create_item");
    expect(names).toContain("move_item");
    // Confirma que delete NÃO está nas tools (decisão de design)
    expect(names.some((n) => n.toLowerCase().includes("delete"))).toBe(false);
  } finally {
    client.close();
  }
});

test("MCP find_item_location encontra item criado via API", async () => {
  // Setup: cria local e item via API HTTP
  const token = await apiLoginAdmin();
  const loc = await (await apiPost(token, "/locations", { name: "Bancada MCP", type: "movel" })).json();
  await apiPost(token, "/items", {
    name: "Furadeira MCP Test",
    location_id: loc.id,
    quantity: 1, unit: "un", condition: "novo",
  });

  // Espera o write do API HTTP ser visível pelo MCP (WAL mode tem janela curta).
  await new Promise((r) => setTimeout(r, 500));

  const client = new MCPClient(TEST_DB);
  try {
    await initialize(client);
    client["proc"].stdin.write(JSON.stringify({ jsonrpc: "2.0", method: "notifications/initialized" }) + "\n");

    const resp = await client.call("tools/call", {
      name: "find_item_location",
      arguments: { query: "furadeira mcp" },
    });

    // resp.result.structuredContent é o objeto retornado pela tool
    const structured = resp.result?.structuredContent;
    expect(structured?.total).toBeGreaterThan(0);
    expect(structured.matches[0].name).toContain("Furadeira MCP");
    expect(structured.matches[0].location_path).toContain("Bancada MCP");
  } finally {
    client.close();
  }
});
