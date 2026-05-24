import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";
import { readFileSync } from "node:fs";

// API_TARGET é onde o proxy do dev server aponta. Em produção usa o default
// (backend na 8080); em E2E o Playwright sobe um backend isolado em outra
// porta e injeta a env var pra cá.
const apiTarget = process.env.VITE_API_TARGET || "http://localhost:8080";

// Versão injetada em build-time: variável de ambiente tem prioridade (build.sh
// a define); em dev lê diretamente do arquivo VERSION na raiz do repo.
const appVersion =
  process.env.VITE_APP_VERSION ||
  (() => {
    try {
      return readFileSync(path.resolve(__dirname, "../VERSION"), "utf-8").trim();
    } catch {
      return "dev";
    }
  })();

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "src"),
    },
  },
  define: {
    __APP_VERSION__: JSON.stringify(appVersion),
  },
  server: {
    port: 5173,
    proxy: {
      "/api": apiTarget,
      "/uploads": apiTarget,
    },
  },
});
