import axios from "axios";
import toast from "react-hot-toast";

export const api = axios.create({
  baseURL: "/api",
  headers: { "Content-Type": "application/json" },
});

api.interceptors.request.use((config) => {
  const token = localStorage.getItem("token");
  if (token) {
    config.headers = config.headers || {};
    (config.headers as any).Authorization = `Bearer ${token}`;
  }
  return config;
});

// Controle leve para evitar spam de toasts em 403 — um por janela de 2s.
let last403At = 0;

// Exposto apenas para testes resetarem o estado do rate-limit entre casos.
// Não usado pela aplicação em produção.
export function _resetRateLimitForTesting() {
  last403At = 0;
}

api.interceptors.response.use(
  (res) => res,
  (err) => {
    const status = err.response?.status;
    if (status === 401) {
      localStorage.removeItem("token");
      localStorage.removeItem("user");
      if (window.location.pathname !== "/login") {
        window.location.href = "/login";
      }
    } else if (status === 403) {
      const now = Date.now();
      if (now - last403At > 2000) {
        last403At = now;
        const msg = err.response?.data?.error || "Acesso negado";
        toast.error(msg);
      }
    }
    return Promise.reject(err);
  }
);
