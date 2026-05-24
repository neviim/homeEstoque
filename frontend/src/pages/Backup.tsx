import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  HardDrive,
  Plus,
  Download,
  ShieldCheck,
  RotateCcw,
  Trash2,
  Loader2,
  CheckCircle2,
  AlertTriangle,
  Clock,
  CalendarClock,
  X,
} from "lucide-react";
import { api } from "@/lib/api";
import { useAuth } from "@/hooks/useAuth";
import toast from "react-hot-toast";

interface Backup {
  id: number;
  filename: string;
  size_bytes: number;
  sha256: string;
  created_at: string;
  type: "manual" | "auto";
  status: "ok" | "corrupted" | "missing" | "orphan" | "unverified";
  verified_at?: string;
  notes?: string;
}

interface Schedule {
  enabled: boolean;
  frequency: "daily" | "weekly";
  weekday: number | null;
  time_of_day: string;
  retention_count: number;
  last_run_at?: string;
  next_run_at?: string;
}

const WEEKDAYS = ["Domingo", "Segunda", "Terça", "Quarta", "Quinta", "Sexta", "Sábado"];

function formatBytes(n: number) {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 * 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1)} MB`;
  return `${(n / 1024 / 1024 / 1024).toFixed(2)} GB`;
}

function formatDateTime(s?: string) {
  if (!s) return "—";
  const d = new Date(s);
  return d.toLocaleString("pt-BR", { dateStyle: "short", timeStyle: "short" });
}

function TypeBadge({ type }: { type: Backup["type"] }) {
  if (type === "auto") {
    return (
      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-blue-50 text-blue-700">
        <Clock className="w-3 h-3" /> Auto
      </span>
    );
  }
  return (
    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-slate-100 text-slate-700">
      Manual
    </span>
  );
}

function StatusBadge({ status }: { status: Backup["status"] }) {
  if (status === "ok") {
    return (
      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-emerald-50 text-emerald-700">
        <CheckCircle2 className="w-3 h-3" /> Íntegro
      </span>
    );
  }
  if (status === "corrupted") {
    return (
      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-red-50 text-red-700">
        <AlertTriangle className="w-3 h-3" /> Corrompido
      </span>
    );
  }
  if (status === "missing") {
    return (
      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-amber-50 text-amber-700">
        <AlertTriangle className="w-3 h-3" /> Sumiu do disco
      </span>
    );
  }
  if (status === "orphan") {
    return (
      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-amber-50 text-amber-700">
        Órfão
      </span>
    );
  }
  return (
    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-slate-100 text-slate-500">
      Não verificado
    </span>
  );
}

function ScheduleCard() {
  const qc = useQueryClient();
  const { data: sched, isLoading } = useQuery<Schedule>({
    queryKey: ["backup", "schedule"],
    queryFn: async () => (await api.get("/backup/schedule")).data,
  });
  const [local, setLocal] = useState<Schedule | null>(null);

  useEffect(() => {
    if (sched && !local) setLocal(sched);
  }, [sched, local]);

  const save = useMutation({
    mutationFn: (s: Schedule) => api.put("/backup/schedule", s).then((r) => r.data as Schedule),
    onSuccess: (data) => {
      qc.setQueryData(["backup", "schedule"], data);
      setLocal(data);
      toast.success("Agendamento salvo");
    },
    onError: (err: any) => toast.error(err?.response?.data?.error || "Erro ao salvar"),
  });

  if (isLoading || !local) {
    return (
      <div className="card mb-6 p-6 flex items-center justify-center">
        <Loader2 className="w-5 h-5 animate-spin text-brand-500" />
      </div>
    );
  }

  return (
    <div className="card mb-6">
      <div className="flex items-center gap-3 px-6 py-4 border-b border-slate-100">
        <div className="w-9 h-9 rounded-lg bg-brand-50 flex items-center justify-center">
          <CalendarClock className="w-5 h-5 text-brand-600" />
        </div>
        <div className="flex-1">
          <h2 className="text-sm font-semibold text-slate-900">Agendamento automático</h2>
          <p className="text-xs text-slate-500">
            {local.enabled
              ? `Próxima execução: ${formatDateTime(local.next_run_at)}`
              : "Desativado — os backups precisam ser feitos manualmente"}
          </p>
        </div>
        <label className="inline-flex items-center cursor-pointer">
          <input
            type="checkbox"
            className="sr-only peer"
            checked={local.enabled}
            onChange={(e) => setLocal({ ...local, enabled: e.target.checked })}
          />
          <div className="w-11 h-6 bg-slate-200 rounded-full peer-checked:bg-brand-500 transition relative">
            <div
              className={`absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full shadow transition ${
                local.enabled ? "translate-x-5" : ""
              }`}
            />
          </div>
        </label>
      </div>

      <div className="p-6 grid grid-cols-1 md:grid-cols-4 gap-4">
        <div>
          <label className="label">Frequência</label>
          <select
            className="input"
            value={local.frequency}
            onChange={(e) => setLocal({ ...local, frequency: e.target.value as "daily" | "weekly" })}
            disabled={!local.enabled}
          >
            <option value="daily">Diário</option>
            <option value="weekly">Semanal</option>
          </select>
        </div>

        {local.frequency === "weekly" && (
          <div>
            <label className="label">Dia da semana</label>
            <select
              className="input"
              value={local.weekday ?? 0}
              onChange={(e) => setLocal({ ...local, weekday: Number(e.target.value) })}
              disabled={!local.enabled}
            >
              {WEEKDAYS.map((w, i) => (
                <option key={i} value={i}>
                  {w}
                </option>
              ))}
            </select>
          </div>
        )}

        <div>
          <label className="label">Horário (24h)</label>
          <div className="flex gap-1.5 items-center">
            <select
              className="input"
              value={local.time_of_day.split(":")[0]}
              onChange={(e) =>
                setLocal({ ...local, time_of_day: `${e.target.value}:${local.time_of_day.split(":")[1]}` })
              }
              disabled={!local.enabled}
            >
              {Array.from({ length: 24 }, (_, h) => String(h).padStart(2, "0")).map((h) => (
                <option key={h} value={h}>{h}h</option>
              ))}
            </select>
            <span className="text-slate-400 text-sm font-medium">:</span>
            <select
              className="input"
              value={local.time_of_day.split(":")[1]}
              onChange={(e) =>
                setLocal({ ...local, time_of_day: `${local.time_of_day.split(":")[0]}:${e.target.value}` })
              }
              disabled={!local.enabled}
            >
              {["00", "15", "30", "45"].map((m) => (
                <option key={m} value={m}>{m}min</option>
              ))}
            </select>
          </div>
        </div>

        <div>
          <label className="label">Manter últimos</label>
          <input
            type="number"
            min={1}
            max={100}
            className="input"
            value={local.retention_count}
            onChange={(e) => setLocal({ ...local, retention_count: Number(e.target.value) })}
          />
        </div>

        <div className="md:col-span-4 flex items-center justify-between gap-3 pt-1">
          <p className="text-xs text-slate-400">
            {local.last_run_at && <>Última execução: {formatDateTime(local.last_run_at)}</>}
          </p>
          <button
            onClick={() => {
              const payload: Schedule = {
                ...local,
                weekday: local.frequency === "weekly" ? local.weekday ?? 0 : null,
              };
              save.mutate(payload);
            }}
            disabled={save.isPending}
            className="btn-primary gap-2"
          >
            {save.isPending && <Loader2 className="w-4 h-4 animate-spin" />}
            Salvar agendamento
          </button>
        </div>
      </div>
    </div>
  );
}

interface RestoreModalProps {
  backup: Backup;
  onClose: () => void;
  onConfirmed: () => void;
}

function RestoreModal({ backup, onClose, onConfirmed }: RestoreModalProps) {
  const [step, setStep] = useState<"warn" | "confirm" | "running">("warn");
  const [ack, setAck] = useState(false);
  const [token, setToken] = useState<string | null>(null);

  async function prepare() {
    try {
      const r = await api.post(`/backups/${backup.id}/restore/prepare`);
      setToken(r.data.confirm_token);
      setStep("confirm");
    } catch (err: any) {
      toast.error(err?.response?.data?.error || "Erro ao preparar restore");
    }
  }

  async function execute() {
    if (!token) return;
    setStep("running");
    try {
      await api.post(`/backups/${backup.id}/restore`, { confirm_token: token });
      onConfirmed();
      // O servidor vai reiniciar — o usuário vê uma tela de "aguardando"
      toast.success("Restore iniciado, servidor reiniciando...");
      setTimeout(() => {
        window.location.href = "/login";
      }, 3000);
    } catch (err: any) {
      toast.error(err?.response?.data?.error || "Erro ao restaurar");
      setStep("confirm");
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-xl shadow-2xl w-full max-w-lg">
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-100">
          <h2 className="text-lg font-semibold text-slate-900 flex items-center gap-2">
            <RotateCcw className="w-5 h-5 text-amber-600" />
            Restaurar backup
          </h2>
          {step !== "running" && (
            <button onClick={onClose} className="p-1.5 text-slate-400 hover:text-slate-600 rounded-lg">
              <X className="w-5 h-5" />
            </button>
          )}
        </div>

        <div className="p-6 space-y-4">
          {step === "warn" && (
            <>
              <div className="flex gap-3 p-4 rounded-lg bg-amber-50 border border-amber-200">
                <AlertTriangle className="w-5 h-5 text-amber-600 shrink-0 mt-0.5" />
                <div className="text-sm text-amber-900">
                  <p className="font-semibold mb-1">Operação irreversível</p>
                  <p>
                    Restaurar este backup vai <b>substituir</b> todos os dados atuais
                    (itens, categorias, locais, usuários, fotos). Um snapshot automático
                    é criado antes como rede de segurança.
                  </p>
                </div>
              </div>
              <div className="text-sm text-slate-700">
                <div>
                  <b>Backup:</b> <span className="font-mono text-xs">{backup.filename}</span>
                </div>
                <div>
                  <b>Criado em:</b> {formatDateTime(backup.created_at)}
                </div>
                <div>
                  <b>Tamanho:</b> {formatBytes(backup.size_bytes)}
                </div>
              </div>
              <label className="flex items-center gap-2 text-sm text-slate-700">
                <input type="checkbox" checked={ack} onChange={(e) => setAck(e.target.checked)} />
                Entendo que isso vai substituir todos os dados atuais
              </label>
              <div className="flex gap-3 pt-2">
                <button className="flex-1 btn-secondary" onClick={onClose}>
                  Cancelar
                </button>
                <button className="flex-1 btn-primary" disabled={!ack} onClick={prepare}>
                  Continuar
                </button>
              </div>
            </>
          )}

          {step === "confirm" && (
            <>
              <p className="text-sm text-slate-700">
                Confirmação preparada. O servidor vai reiniciar após o restore — a página
                redireciona para o login automaticamente.
              </p>
              <div className="flex gap-3 pt-2">
                <button className="flex-1 btn-secondary" onClick={onClose}>
                  Voltar
                </button>
                <button className="flex-1 btn-primary bg-amber-600 hover:bg-amber-700" onClick={execute}>
                  Restaurar agora
                </button>
              </div>
            </>
          )}

          {step === "running" && (
            <div className="flex flex-col items-center py-8 gap-3">
              <Loader2 className="w-8 h-8 animate-spin text-brand-500" />
              <p className="text-sm text-slate-600">Restaurando e reiniciando o servidor…</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default function BackupPage() {
  const qc = useQueryClient();
  const { hasPermission } = useAuth();
  const [restoreTarget, setRestoreTarget] = useState<Backup | null>(null);

  const { data: backups = [], isLoading } = useQuery<Backup[]>({
    queryKey: ["backups"],
    queryFn: async () => (await api.get<{ backups: Backup[] }>("/backups")).data.backups ?? [],
  });

  const invalidate = () => qc.invalidateQueries({ queryKey: ["backups"] });

  const createMut = useMutation({
    mutationFn: () => api.post("/backups"),
    onSuccess: () => {
      toast.success("Backup criado");
      invalidate();
    },
    onError: (err: any) => toast.error(err?.response?.data?.error || "Erro ao criar"),
  });

  const verifyMut = useMutation({
    mutationFn: (id: number) => api.post(`/backups/${id}/verify`),
    onSuccess: (r) => {
      const status = r.data?.status;
      if (status === "ok") toast.success("Backup íntegro");
      else toast.error(`Backup com problema: ${status}`);
      invalidate();
    },
    onError: (err: any) => toast.error(err?.response?.data?.error || "Erro ao verificar"),
  });

  const deleteMut = useMutation({
    mutationFn: (id: number) => api.delete(`/backups/${id}`),
    onSuccess: () => {
      toast.success("Backup excluído");
      invalidate();
    },
    onError: (err: any) => toast.error(err?.response?.data?.error || "Erro ao excluir"),
  });

  function confirmDelete(b: Backup) {
    if (!confirm(`Excluir o backup "${b.filename}"?`)) return;
    deleteMut.mutate(b.id);
  }

  async function downloadBackup(b: Backup) {
    try {
      const token = localStorage.getItem("token");
      const res = await fetch(`/api/backups/${b.id}/download`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = b.filename;
      a.click();
      URL.revokeObjectURL(url);
    } catch (err: any) {
      toast.error("Erro ao baixar: " + err.message);
    }
  }

  const canSchedule = hasPermission("backup.schedule");
  const canRestore = hasPermission("backup.restore");
  const canDownload = hasPermission("backup.download");

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-brand-500 to-indigo-600 flex items-center justify-center">
            <HardDrive className="w-5 h-5 text-white" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-slate-900">Backup</h1>
            <p className="text-sm text-slate-500 mt-0.5">
              Faça backup, restaure e agende a proteção dos seus dados
            </p>
          </div>
        </div>
        <button
          onClick={() => createMut.mutate()}
          disabled={createMut.isPending}
          className="btn-primary gap-2"
        >
          {createMut.isPending ? (
            <Loader2 className="w-4 h-4 animate-spin" />
          ) : (
            <Plus className="w-4 h-4" />
          )}
          Criar backup
        </button>
      </div>

      {canSchedule && <ScheduleCard />}

      <div className="card">
        {isLoading ? (
          <div className="flex justify-center p-12">
            <Loader2 className="w-6 h-6 animate-spin text-brand-500" />
          </div>
        ) : backups.length === 0 ? (
          <div className="text-center py-16 text-slate-400">
            <HardDrive className="w-10 h-10 mx-auto mb-3 opacity-30" />
            <p className="mb-1">Nenhum backup ainda</p>
            <p className="text-xs">Clique em "Criar backup" para fazer o primeiro.</p>
          </div>
        ) : (
          <table className="w-full">
            <thead>
              <tr className="border-b border-slate-100">
                <th className="text-left text-xs font-semibold text-slate-500 uppercase tracking-wider px-5 py-3">
                  Arquivo
                </th>
                <th className="text-left text-xs font-semibold text-slate-500 uppercase tracking-wider px-5 py-3">
                  Tamanho
                </th>
                <th className="text-left text-xs font-semibold text-slate-500 uppercase tracking-wider px-5 py-3">
                  Tipo
                </th>
                <th className="text-left text-xs font-semibold text-slate-500 uppercase tracking-wider px-5 py-3">
                  Status
                </th>
                <th className="text-left text-xs font-semibold text-slate-500 uppercase tracking-wider px-5 py-3">
                  Criado em
                </th>
                <th className="px-5 py-3" />
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-50">
              {backups.map((b) => (
                <tr key={b.id} className="hover:bg-slate-50/50 transition">
                  <td className="px-5 py-3.5">
                    <div className="text-sm font-medium text-slate-900 font-mono">{b.filename}</div>
                    <div className="text-xs text-slate-400 font-mono">
                      sha256: {b.sha256 ? b.sha256.slice(0, 16) + "…" : "—"}
                    </div>
                  </td>
                  <td className="px-5 py-3.5 text-sm text-slate-600">{formatBytes(b.size_bytes)}</td>
                  <td className="px-5 py-3.5">
                    <TypeBadge type={b.type} />
                  </td>
                  <td className="px-5 py-3.5">
                    <StatusBadge status={b.status} />
                  </td>
                  <td className="px-5 py-3.5 text-sm text-slate-500">
                    {formatDateTime(b.created_at)}
                  </td>
                  <td className="px-5 py-3.5">
                    <div className="flex items-center justify-end gap-1">
                      <button
                        onClick={() => verifyMut.mutate(b.id)}
                        disabled={verifyMut.isPending}
                        className="p-1.5 text-slate-400 hover:text-brand-600 hover:bg-brand-50 rounded-lg transition"
                        title="Verificar integridade"
                      >
                        <ShieldCheck className="w-4 h-4" />
                      </button>
                      {canDownload && (
                        <button
                          onClick={() => downloadBackup(b)}
                          className="p-1.5 text-slate-400 hover:text-emerald-600 hover:bg-emerald-50 rounded-lg transition"
                          title="Baixar"
                        >
                          <Download className="w-4 h-4" />
                        </button>
                      )}
                      {canRestore && (
                        <button
                          onClick={() => setRestoreTarget(b)}
                          disabled={b.status !== "ok"}
                          className="p-1.5 text-slate-400 hover:text-amber-600 hover:bg-amber-50 rounded-lg transition disabled:opacity-30 disabled:cursor-not-allowed"
                          title="Restaurar"
                        >
                          <RotateCcw className="w-4 h-4" />
                        </button>
                      )}
                      <button
                        onClick={() => confirmDelete(b)}
                        className="p-1.5 text-slate-400 hover:text-red-600 hover:bg-red-50 rounded-lg transition"
                        title="Excluir"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {restoreTarget && (
        <RestoreModal
          backup={restoreTarget}
          onClose={() => setRestoreTarget(null)}
          onConfirmed={() => setRestoreTarget(null)}
        />
      )}
    </div>
  );
}
