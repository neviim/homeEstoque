// Configuração Playwright para HomeEstoque E2E.
//
// Estratégia:
// - Sobe backend Go e frontend Vite via webServer (Playwright gerencia o
//   ciclo de vida automaticamente).
// - Backend usa porta 8090 com DB efêmero em /tmp/homeestoque-e2e.db
//   (apagado pelo globalSetup antes de subir).
// - Frontend usa porta 5174 com VITE_API_TARGET apontando para 8090.
// - Browser cache local (não polui ~/.cache compartilhado).

import { defineConfig } from "@playwright/test";
import path from "path";

const ROOT = path.resolve(__dirname, "../..");
const BACKEND_DIR = path.join(ROOT, "backend");
const FRONTEND_DIR = path.join(ROOT, "frontend");
const TEST_DB = "/tmp/homeestoque-e2e.db";

process.env.PLAYWRIGHT_BROWSERS_PATH = path.join(__dirname, ".playwright-cache");

export default defineConfig({
  testDir: __dirname,
  testMatch: /.*\.spec\.ts/,
  timeout: 30_000,
  expect: { timeout: 5_000 },
  fullyParallel: false, // backend compartilhado tem estado, sem paralelismo
  workers: 1,
  retries: 0,
  reporter: [["list"]],
  globalSetup: require.resolve("./globalSetup"),

  use: {
    baseURL: "http://localhost:5174",
    headless: true,
    screenshot: "only-on-failure",
    trace: "retain-on-failure",
    actionTimeout: 10_000,
  },

  projects: [
    {
      name: "chromium",
      use: {
        browserName: "chromium",
        viewport: { width: 1280, height: 800 },
      },
    },
  ],

  webServer: [
    {
      command: `go run ./cmd/api`,
      cwd: BACKEND_DIR,
      env: {
        PORT: "8090",
        DB_PATH: TEST_DB,
        JWT_SECRET: "e2e-secret",
        UPLOAD_DIR: "/tmp/homeestoque-e2e-uploads",
        CORS_ORIGINS: "http://localhost:5174",
        PATH: process.env.PATH || "",
      },
      url: "http://localhost:8090/health",
      timeout: 60_000,
      reuseExistingServer: !process.env.CI,
      stdout: "pipe",
      stderr: "pipe",
    },
    {
      command: "npm run dev -- --port 5174 --strictPort",
      cwd: FRONTEND_DIR,
      env: {
        VITE_API_TARGET: "http://localhost:8090",
        PATH: process.env.PATH || "",
      },
      url: "http://localhost:5174",
      timeout: 60_000,
      reuseExistingServer: !process.env.CI,
      stdout: "pipe",
      stderr: "pipe",
    },
  ],
});
