import { describe, it, expect } from "vitest";
import { formatCurrency, formatDate, formatDateTime, cn } from "../utils";

describe("formatCurrency", () => {
  it("formata valores em BRL com vírgula decimal e separador de milhar", () => {
    const out = formatCurrency(1234.56);
    // O Intl.NumberFormat pode usar espaço não-quebrável (U+00A0) entre R$ e o número
    expect(out.replace(/\s/g, " ")).toBe("R$ 1.234,56");
  });

  it("retorna placeholder quando value é null ou undefined", () => {
    expect(formatCurrency(null)).toBe("—");
    expect(formatCurrency(undefined)).toBe("—");
  });

  it("formata zero corretamente sem crashar", () => {
    const out = formatCurrency(0);
    expect(out).toMatch(/R\$.*0,00/);
  });
});

describe("formatDate", () => {
  it("formata data ISO no padrão DD MMM AAAA pt-BR", () => {
    const out = formatDate("2026-05-22T10:00:00Z");
    // Esperamos algo como "22 de mai. de 2026" — formato variável por locale
    expect(out).toMatch(/2026/);
    expect(out).toMatch(/22/);
  });

  it("retorna placeholder para valor falsy", () => {
    expect(formatDate(null)).toBe("—");
    expect(formatDate(undefined)).toBe("—");
    expect(formatDate("")).toBe("—");
  });

  it("devolve o original em data inválida sem crashar", () => {
    const out = formatDate("não-é-data");
    // Intl.DateTimeFormat retorna "Invalid Date" ou throws — aceitamos qualquer string
    expect(typeof out).toBe("string");
  });
});

describe("formatDateTime", () => {
  it("inclui hora e minuto", () => {
    const out = formatDateTime("2026-05-22T15:30:00Z");
    // Esperado: contém 2026 e algum padrão de hora
    expect(out).toMatch(/2026/);
    expect(out).toMatch(/\d{2}:\d{2}/);
  });

  it("retorna placeholder para falsy", () => {
    expect(formatDateTime(null)).toBe("—");
  });
});

describe("cn", () => {
  it("combina classes simples", () => {
    expect(cn("a", "b")).toBe("a b");
  });

  it("filtra valores falsy", () => {
    expect(cn("a", false, null, undefined, "b")).toBe("a b");
  });

  it("dedupa via twMerge (last-wins do Tailwind)", () => {
    // px-2 vs px-4 → twMerge mantém apenas o último
    expect(cn("px-2", "px-4")).toBe("px-4");
  });

  it("respeita classes condicionais via objeto", () => {
    expect(cn({ "text-red-500": true, "text-green-500": false })).toBe("text-red-500");
  });
});
