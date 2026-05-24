import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";

// API_TARGET é onde o proxy do dev server aponta. Em produção usa o default
// (backend na 8080); em E2E o Playwright sobe um backend isolado em outra
// porta e injeta a env var pra cá.
const apiTarget = process.env.VITE_API_TARGET || "http://localhost:8080";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "src"),
    },
  },
  server: {
    port: 5173,
    proxy: {
      "/api": apiTarget,
      "/uploads": apiTarget,
    },
  },
});
