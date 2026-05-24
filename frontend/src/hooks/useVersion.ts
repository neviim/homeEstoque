import { useCallback } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import toast from "react-hot-toast";
import { api } from "@/lib/api";

interface VersionInfo {
  running: string;
  available: string;
  update_available: boolean;
}

function dismissKey(version: string) {
  return `update-dismissed-${version}`;
}

export function useVersion() {
  const navigate = useNavigate();

  const { data } = useQuery<VersionInfo>({
    queryKey: ["version"],
    queryFn: async () => (await api.get<VersionInfo>("/version")).data,
    refetchInterval: 60_000,
    refetchIntervalInBackground: false,
    staleTime: 30_000,
  });

  const running = data?.running ?? __APP_VERSION__;
  const available = data?.available ?? __APP_VERSION__;
  const updateAvailable = data?.update_available ?? false;

  const isDismissed =
    updateAvailable &&
    typeof window !== "undefined" &&
    localStorage.getItem(dismissKey(available)) === "1";

  const dismiss = useCallback(() => {
    if (available) localStorage.setItem(dismissKey(available), "1");
  }, [available]);

  const apply = useCallback(async () => {
    try {
      await api.post("/version/apply");
      toast.success("Reiniciando sistema…");
      setTimeout(() => navigate("/login"), 3000);
    } catch {
      toast.error("Falha ao aplicar atualização");
    }
  }, [navigate]);

  return { running, available, updateAvailable, dismissed: isDismissed, dismiss, apply };
}
