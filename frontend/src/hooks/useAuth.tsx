import { createContext, useContext, useEffect, useState, ReactNode } from "react";
import { api } from "@/lib/api";
import type { User } from "@/types";

interface AuthCtx {
  user: User | null;
  loading: boolean;
  isAdmin: boolean;
  isViewer: boolean;
  permissions: string[];
  hasPermission: (key: string) => boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (name: string, email: string, password: string) => Promise<"active" | "pending">;
  logout: () => void;
  updateUser: (updated: User) => void;
  refreshUser: () => Promise<void>;
}

const Ctx = createContext<AuthCtx | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const stored = localStorage.getItem("user");
    const token = localStorage.getItem("token");
    if (stored && token) {
      try {
        setUser(JSON.parse(stored));
      } catch {
        /* ignore */
      }
      // Atualiza permissions em background — se mudou no servidor enquanto o user estava deslogado/JWT antigo
      api.get<User>("/auth/me")
        .then(({ data }) => {
          localStorage.setItem("user", JSON.stringify(data));
          setUser(data);
        })
        .catch(() => {/* ignore */})
        .finally(() => setLoading(false));
      return;
    }
    setLoading(false);
  }, []);

  async function login(email: string, password: string) {
    const { data } = await api.post("/auth/login", { email, password });
    localStorage.setItem("token", data.token);
    localStorage.setItem("user", JSON.stringify(data.user));
    setUser(data.user);
  }

  async function register(name: string, email: string, password: string): Promise<"active" | "pending"> {
    const { data } = await api.post("/auth/register", { name, email, password });
    if (data.status === "pending") {
      return "pending";
    }
    localStorage.setItem("token", data.token);
    localStorage.setItem("user", JSON.stringify(data.user));
    setUser(data.user);
    return "active";
  }

  function logout() {
    localStorage.removeItem("token");
    localStorage.removeItem("user");
    setUser(null);
    window.location.href = "/login";
  }

  function updateUser(updated: User) {
    localStorage.setItem("user", JSON.stringify(updated));
    setUser(updated);
  }

  async function refreshUser() {
    try {
      const { data } = await api.get<User>("/auth/me");
      localStorage.setItem("user", JSON.stringify(data));
      setUser(data);
    } catch {
      /* ignore */
    }
  }

  const permissions = user?.permissions ?? [];
  const hasPermission = (key: string) => permissions.includes(key);
  const isAdmin = hasPermission("roles.manage") && hasPermission("users.manage");
  const isViewer = user?.role === "viewer"; // mantido apenas para o badge visual

  return (
    <Ctx.Provider value={{ user, loading, isAdmin, isViewer, permissions, hasPermission, login, register, logout, updateUser, refreshUser }}>
      {children}
    </Ctx.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(Ctx);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
