import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  UserPlus,
  Pencil,
  Trash2,
  KeyRound,
  CheckCircle2,
  XCircle,
  Clock,
  ShieldCheck,
  User as UserIcon,
  Eye,
  Loader2,
  X,
} from "lucide-react";
import { api } from "@/lib/api";
import toast from "react-hot-toast";
import type { User } from "@/types";

type StatusFilter = "all" | "active" | "pending" | "inactive";

interface UserListResponse {
  users: User[];
}

function fetchUsers(): Promise<User[]> {
  return api.get<UserListResponse>("/users").then((r) => r.data.users);
}

const STATUS_LABELS: Record<string, string> = {
  active: "Ativo",
  pending: "Pendente",
  inactive: "Inativo",
};

const ROLE_LABELS: Record<string, string> = {
  admin: "Admin",
  user: "Usuário",
  viewer: "Visualizador",
};

function StatusBadge({ status }: { status?: string }) {
  if (status === "active")
    return (
      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-green-50 text-green-700">
        <CheckCircle2 className="w-3 h-3" /> Ativo
      </span>
    );
  if (status === "pending")
    return (
      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-amber-50 text-amber-700">
        <Clock className="w-3 h-3" /> Pendente
      </span>
    );
  return (
    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-slate-100 text-slate-500">
      <XCircle className="w-3 h-3" /> Inativo
    </span>
  );
}

function RoleBadge({ role }: { role?: string }) {
  if (role === "admin")
    return (
      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-brand-50 text-brand-700">
        <ShieldCheck className="w-3 h-3" /> Admin
      </span>
    );
  if (role === "viewer")
    return (
      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-purple-50 text-purple-700">
        <Eye className="w-3 h-3" /> Visualizador
      </span>
    );
  return (
    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-slate-100 text-slate-600">
      <UserIcon className="w-3 h-3" /> Usuário
    </span>
  );
}

interface UserFormData {
  name: string;
  email: string;
  password: string;
  role: string;
}


interface UserModalProps {
  user?: User;
  onClose: () => void;
  onSave: (data: UserFormData) => Promise<void>;
}

function UserModal({ user, onClose, onSave }: UserModalProps) {
  const [name, setName] = useState(user?.name || "");
  const [email, setEmail] = useState(user?.email || "");
  const [password, setPassword] = useState("");
  const [role, setRole] = useState(user?.role || "user");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    try {
      await onSave({ name, email, password, role });
      onClose();
    } catch (err: any) {
      toast.error(err?.response?.data?.error || "Erro ao salvar");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-2xl shadow-2xl w-full max-w-md">
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-100">
          <h2 className="text-lg font-semibold text-slate-900">
            {user ? "Editar usuário" : "Novo usuário"}
          </h2>
          <button onClick={onClose} className="p-1.5 text-slate-400 hover:text-slate-600 rounded-lg transition">
            <X className="w-5 h-5" />
          </button>
        </div>
        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          <div>
            <label className="label">Nome</label>
            <input className="input" value={name} onChange={(e) => setName(e.target.value)} required placeholder="Nome completo" />
          </div>
          {!user ? (
            <div>
              <label className="label">Email</label>
              <input type="email" className="input" value={email} onChange={(e) => setEmail(e.target.value)} required placeholder="email@exemplo.com" />
            </div>
          ) : (
            <div>
              <label className="label">Email</label>
              <div className="px-3 py-2.5 rounded-lg bg-slate-100 text-slate-500 text-sm select-all cursor-default">
                {user.email}
              </div>
              <p className="text-xs text-slate-400 mt-1">O e-mail não pode ser alterado após o cadastro.</p>
            </div>
          )}
          {!user && (
            <div>
              <label className="label">Senha</label>
              <input
                type="password"
                className="input"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                minLength={6}
                placeholder="Mínimo 6 caracteres"
              />
            </div>
          )}
          <div>
            <label className="label">Perfil</label>
            <select className="input" value={role} onChange={(e) => setRole(e.target.value)}>
              <option value="user">Usuário</option>
              <option value="viewer">Visualizador (somente leitura)</option>
              <option value="admin">Administrador</option>
            </select>
          </div>
          <div className="flex gap-3 pt-2">
            <button type="button" onClick={onClose} className="flex-1 btn-secondary">
              Cancelar
            </button>
            <button type="submit" className="flex-1 btn-primary" disabled={loading}>
              {loading && <Loader2 className="w-4 h-4 animate-spin" />}
              {user ? "Salvar" : "Criar"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

interface ResetPwdModalProps {
  user: User;
  onClose: () => void;
}

function ResetPwdModal({ user, onClose }: ResetPwdModalProps) {
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    try {
      await api.put(`/users/${user.id}/password`, { password });
      toast.success("Senha redefinida com sucesso");
      onClose();
    } catch (err: any) {
      toast.error(err?.response?.data?.error || "Erro ao redefinir senha");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-2xl shadow-2xl w-full max-w-sm">
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-100">
          <h2 className="text-lg font-semibold text-slate-900">Redefinir senha</h2>
          <button onClick={onClose} className="p-1.5 text-slate-400 hover:text-slate-600 rounded-lg transition">
            <X className="w-5 h-5" />
          </button>
        </div>
        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          <p className="text-sm text-slate-500">
            Nova senha para <span className="font-medium text-slate-800">{user.name}</span>
          </p>
          <div>
            <label className="label">Nova senha</label>
            <input
              type="password"
              className="input"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              minLength={6}
              placeholder="Mínimo 6 caracteres"
              autoFocus
            />
          </div>
          <div className="flex gap-3 pt-2">
            <button type="button" onClick={onClose} className="flex-1 btn-secondary">
              Cancelar
            </button>
            <button type="submit" className="flex-1 btn-primary" disabled={loading}>
              {loading && <Loader2 className="w-4 h-4 animate-spin" />}
              Redefinir
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

export default function Users() {
  const qc = useQueryClient();
  const [filter, setFilter] = useState<StatusFilter>("all");
  const [showAdd, setShowAdd] = useState(false);
  const [editUser, setEditUser] = useState<User | null>(null);
  const [resetUser, setResetUser] = useState<User | null>(null);

  const { data: users = [], isLoading } = useQuery({ queryKey: ["users"], queryFn: fetchUsers });

  const invalidate = () => qc.invalidateQueries({ queryKey: ["users"] });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/users/${id}`),
    onSuccess: () => { toast.success("Usuário excluído"); invalidate(); },
    onError: (err: any) => toast.error(err?.response?.data?.error || "Erro ao excluir"),
  });

  const statusMutation = useMutation({
    mutationFn: ({ id, status }: { id: number; status: string }) =>
      api.put(`/users/${id}/status`, { status }),
    onSuccess: () => { toast.success("Status atualizado"); invalidate(); },
    onError: (err: any) => toast.error(err?.response?.data?.error || "Erro ao atualizar status"),
  });

  async function handleCreate(data: UserFormData) {
    await api.post("/users", data);
    toast.success("Usuário criado");
    invalidate();
  }

  async function handleEdit(data: UserFormData) {
    if (!editUser) return;
    await api.put(`/users/${editUser.id}`, { name: data.name, role: data.role });
    toast.success("Usuário atualizado");
    invalidate();
  }

  function confirmDelete(user: User) {
    if (!confirm(`Excluir o usuário "${user.name}"? Esta ação não pode ser desfeita.`)) return;
    deleteMutation.mutate(user.id);
  }

  const filtered = filter === "all" ? users : users.filter((u) => u.status === filter);

  const tabs: { key: StatusFilter; label: string }[] = [
    { key: "all", label: "Todos" },
    { key: "active", label: "Ativos" },
    { key: "pending", label: "Pendentes" },
    { key: "inactive", label: "Inativos" },
  ];

  const pendingCount = users.filter((u) => u.status === "pending").length;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Usuários</h1>
          <p className="text-sm text-slate-500 mt-0.5">Gerencie os membros com acesso ao sistema</p>
        </div>
        <button onClick={() => setShowAdd(true)} className="btn-primary gap-2">
          <UserPlus className="w-4 h-4" />
          Novo usuário
        </button>
      </div>

      <div className="card">
        <div className="flex items-center gap-1 p-4 border-b border-slate-100">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              onClick={() => setFilter(tab.key)}
              className={`relative px-4 py-1.5 rounded-lg text-sm font-medium transition ${
                filter === tab.key
                  ? "bg-brand-50 text-brand-700"
                  : "text-slate-500 hover:text-slate-800 hover:bg-slate-50"
              }`}
            >
              {tab.label}
              {tab.key === "pending" && pendingCount > 0 && (
                <span className="ml-1.5 inline-flex items-center justify-center w-4 h-4 text-xs rounded-full bg-amber-500 text-white font-bold">
                  {pendingCount}
                </span>
              )}
            </button>
          ))}
        </div>

        {isLoading ? (
          <div className="flex justify-center p-12">
            <Loader2 className="w-6 h-6 animate-spin text-brand-500" />
          </div>
        ) : filtered.length === 0 ? (
          <div className="text-center py-12 text-slate-400">
            <UserIcon className="w-10 h-10 mx-auto mb-3 opacity-30" />
            <p>Nenhum usuário encontrado</p>
          </div>
        ) : (
          <table className="w-full">
            <thead>
              <tr className="border-b border-slate-100">
                <th className="text-left text-xs font-semibold text-slate-500 uppercase tracking-wider px-5 py-3">Usuário</th>
                <th className="text-left text-xs font-semibold text-slate-500 uppercase tracking-wider px-5 py-3">Perfil</th>
                <th className="text-left text-xs font-semibold text-slate-500 uppercase tracking-wider px-5 py-3">Status</th>
                <th className="text-left text-xs font-semibold text-slate-500 uppercase tracking-wider px-5 py-3">Cadastrado em</th>
                <th className="px-5 py-3" />
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-50">
              {filtered.map((user) => (
                <tr key={user.id} className={`transition ${user.status === "inactive" ? "opacity-40 hover:opacity-60" : "hover:bg-slate-50/50"}`}>
                  <td className="px-5 py-3.5">
                    <div className="flex items-center gap-3">
                      <div className="w-8 h-8 rounded-full bg-gradient-to-br from-brand-500 to-indigo-600 text-white flex items-center justify-center text-sm font-semibold shrink-0">
                        {user.name?.[0]?.toUpperCase()}
                      </div>
                      <div>
                        <div className="text-sm font-medium text-slate-900">{user.name}</div>
                        <div className="text-xs text-slate-500">{user.email}</div>
                      </div>
                    </div>
                  </td>
                  <td className="px-5 py-3.5">
                    <RoleBadge role={user.role} />
                  </td>
                  <td className="px-5 py-3.5">
                    <StatusBadge status={user.status} />
                  </td>
                  <td className="px-5 py-3.5 text-sm text-slate-500">
                    {user.created_at ? new Date(user.created_at).toLocaleDateString("pt-BR") : "—"}
                  </td>
                  <td className="px-5 py-3.5">
                    <div className="flex items-center justify-end gap-1">
                      {user.status === "pending" && (
                        <>
                          <button
                            onClick={() => statusMutation.mutate({ id: user.id, status: "active" })}
                            className="flex items-center gap-1 px-2.5 py-1 text-xs font-medium rounded-lg bg-green-50 text-green-700 hover:bg-green-100 transition"
                            title="Aprovar"
                          >
                            <CheckCircle2 className="w-3.5 h-3.5" /> Aprovar
                          </button>
                          <button
                            onClick={() => statusMutation.mutate({ id: user.id, status: "inactive" })}
                            className="flex items-center gap-1 px-2.5 py-1 text-xs font-medium rounded-lg bg-red-50 text-red-600 hover:bg-red-100 transition"
                            title="Rejeitar"
                          >
                            <XCircle className="w-3.5 h-3.5" /> Rejeitar
                          </button>
                        </>
                      )}
                      {user.status === "active" && (
                        <button
                          onClick={() => statusMutation.mutate({ id: user.id, status: "inactive" })}
                          className="px-2.5 py-1 text-xs font-medium rounded-lg bg-slate-100 text-slate-600 hover:bg-slate-200 transition"
                          title="Inativar"
                        >
                          Inativar
                        </button>
                      )}
                      {user.status === "inactive" && (
                        <button
                          onClick={() => statusMutation.mutate({ id: user.id, status: "active" })}
                          className="px-2.5 py-1 text-xs font-medium rounded-lg bg-green-50 text-green-700 hover:bg-green-100 transition"
                          title="Ativar"
                        >
                          Ativar
                        </button>
                      )}
                      <button
                        onClick={() => setEditUser(user)}
                        className="p-1.5 text-slate-400 hover:text-brand-600 hover:bg-brand-50 rounded-lg transition"
                        title="Editar"
                      >
                        <Pencil className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => setResetUser(user)}
                        className="p-1.5 text-slate-400 hover:text-amber-600 hover:bg-amber-50 rounded-lg transition"
                        title="Redefinir senha"
                      >
                        <KeyRound className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => confirmDelete(user)}
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

      {showAdd && (
        <UserModal onClose={() => setShowAdd(false)} onSave={handleCreate} />
      )}
      {editUser && (
        <UserModal user={editUser} onClose={() => setEditUser(null)} onSave={handleEdit} />
      )}
      {resetUser && (
        <ResetPwdModal user={resetUser} onClose={() => setResetUser(null)} />
      )}
    </div>
  );
}
