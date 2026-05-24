import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import path from "path";

// Configuração dedicada para testes — não usa o vite.config (que tem proxy
// dev incompatível com jsdom). Mantém o alias "@" igual ao prod.
export default defineConfig({
  plugins: [react()],
  define: {
    // Replica o define do vite.config para que os testes enxerguem a constante.
    __APP_VERSION__: JSON.stringify("test"),
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "src"),
    },
  },
  test: {
    environment: "jsdom",
    globals: true, // expõe describe/it/expect/vi sem import
    setupFiles: ["./src/test/setup.ts"],
    css: false, // não processa CSS nos testes (Tailwind classes são strings)
    // Isolamento explícito por arquivo — evita compartilhamento de estado
    // global (window.location, axios defaults, módulos com let no escopo
    // de módulo como last403At em api.ts).
    isolate: true,
    coverage: {
      provider: "v8",
      reporter: ["text", "html"],
      include: [
        "src/hooks/**/*.{ts,tsx}",
        "src/lib/**/*.{ts,tsx}",
        "src/pages/**/*.{ts,tsx}",
        "src/components/**/*.{ts,tsx}",
        "src/App.tsx",
      ],
      exclude: ["**/*.test.{ts,tsx}", "**/*.d.ts"],
    },
  },
});
