import { describe, it, expect, beforeAll, afterAll, afterEach, beforeEach, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";
import { createWrapper } from "@/test/render";

import { useVersion } from "../useVersion";

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

describe("useVersion", () => {
  it("expõe updateAvailable quando versões diferem", async () => {
    server.use(
      http.get("http://localhost/api/version", () =>
        HttpResponse.json({ running: "0.1.0", available: "0.1.1", update_available: true }),
      ),
    );

    const { result } = renderHook(() => useVersion(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.updateAvailable).toBe(true));
    expect(result.current.running).toBe("0.1.0");
    expect(result.current.available).toBe("0.1.1");
  });

  it("dismissed é false antes de chamar dismiss", async () => {
    server.use(
      http.get("http://localhost/api/version", () =>
        HttpResponse.json({ running: "0.1.0", available: "0.1.1", update_available: true }),
      ),
    );

    const { result } = renderHook(() => useVersion(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.updateAvailable).toBe(true));
    expect(result.current.dismissed).toBe(false);
  });

  it("dismiss persiste no localStorage pela versão disponível", async () => {
    server.use(
      http.get("http://localhost/api/version", () =>
        HttpResponse.json({ running: "0.1.0", available: "0.1.1", update_available: true }),
      ),
    );

    const { result } = renderHook(() => useVersion(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.available).toBe("0.1.1"));

    result.current.dismiss();
    expect(localStorage.getItem("update-dismissed-0.1.1")).toBe("1");
  });
});
