import { useEffect, useMemo, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Plus,
  Pencil,
  Trash2,
  ShieldCheck,
  Lock,
  X,
  Loader2,
  Save,
  AlertCircle,
} from "lucide-react";
import toast from "react-hot-toast";
import { api } from "@/lib/api";
import type { Role, Permission } from "@/types";
import { useAuth } from "@/hooks/useAuth";

function fetchRoles(): Promise<Role[]> {
  return api.get<{ roles: Role[] }>("/roles").then((r) => r.data.roles);
}
function fetchPermissions(): Promise<Permission[]> {
  return api.get<{ permissions: Permission[] }>("/permissions").then((r) => r.data.permissions);
}

function groupBy<T>(arr: T[], key: (t: T) => string): Map<string, T[]> {
  const out = new Map<string, T[]>();
  for (const item of arr) {
    const k = key(item);
    const list = out.get(k) || [];
    list.push(item);
    out.set(k, list);
  }
  return out;
}

interface RoleFormData {
  name: string;
  label: string;
  description: string;
}

interface RoleModalProps {
  role?: Role; // se passado: edição; senão: novo
  onClose: () => void;
  onSave: (data: RoleFormData) => Promise<void>;
}

function RoleModal({ role, onClose, onSave }: RoleModalProps) {
  const [name, setName] = useState(role?.name || "");
  const [label, setLabel] = useState(role?.label || "");
  const [description, setDescription] = useState(role?.description || "");
  const [loading, setLoading] = useState(false);

  const nameLocked = !!role?.is_system;

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    try {
      await onSave({ name: name.trim(), label: label.trim(), description: description.trim() });
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
            {role ? "Editar perfil" : "Novo perfil"}
          </h2>
          <button onClick={onClose} className="p-1.5 text-slate-400 hover:text-slate-600 rounded-lg transition">
            <X className="w-5 h-5" />
          </button>
        </div>
        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          <div>
            <label className="label">Identificador (slug)</label>
            <input
              className="input font-mono text-sm"
              value={name}
              onChange={(e) => setName(e.target.value.toLowerCase().replace(/[^a-z0-9_]/g, "_"))}
              required
              minLength={2}
              maxLength={50}
              placeholder="ex: editor_de_itens"
              disabled={nameLocked}
            />
            {nameLocked && (
              <p className="text-xs text-slate-400 mt-1">
                <Lock className="w-3 h-3 inline mr-1" />
                Perfil de sistema — identificador não pode ser alterado.
              </p>
            )}
          </div>
          <div>
            <label className="label">Nome de exibição</label>
            <input
              className="input"
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              required
              placeholder="ex: Editor de Itens"
            />
          </div>
          <div>
            <label className="label">Descrição (opcional)</label>
            <textarea
              className="input min-h-[70px]"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="O que esse perfil pode fazer?"
            />
          </div>
          <div className="flex gap-3 pt-2">
            <button type="button" onClick={onClose} className="flex-1 btn-secondary">
              Cancelar
            </button>
            <button type="submit" className="flex-1 btn-primary" disabled={loading}>
              {loading && <Loader2 className="w-4 h-4 animate-spin" />}
              {role ? "Salvar" : "Criar"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

export default function Permissions() {
  const qc = useQueryClient();
  const { refreshUser, user } = useAuth();

  const { data: roles = [], isLoading: rolesLoading } = useQuery({ queryKey: ["roles"], queryFn: fetchRoles });
  const { data: catalog = [], isLoading: catalogLoading } = useQuery({
    queryKey: ["permissions"],
    queryFn: fetchPermissions,
  });

  const [selectedRoleId, setSelectedRoleId] = useState<number | null>(null);
  const [draftPerms, setDraftPerms] = useState<Set<string>>(new Set());
  const [showCreate, setShowCreate] = useState(false);
  const [editingRole, setEditingRole] = useState<Role | null>(null);

  // Seleciona o primeiro role assim que carregar
  useEffect(() => {
    if (selectedRoleId === null && roles.length > 0) {
      setSelectedRoleId(roles[0].id);
    }
  }, [roles, selectedRoleId]);

  const selectedRole = useMemo(
    () => roles.find((r) => r.id === selectedRoleId) || null,
    [roles, selectedRoleId]
  );

  // Sempre que o role selecionado mudar, reseta o draft para o estado atual do role
  useEffect(() => {
    if (selectedRole) setDraftPerms(new Set(selectedRole.permissions));
  }, [selectedRole]);

  const grouped = useMemo(() => groupBy(catalog, (p) => p.category), [catalog]);
  const categories = useMemo(() => Array.from(grouped.keys()), [grouped]);

  const isAdminRole = selectedRole?.name === "admin";
  const dirty = useMemo(() => {
    if (!selectedRole) return false;
    const current = new Set(selectedRole.permissions);
    if (current.size !== draftPerms.size) return true;
    for (const p of draftPerms) if (!current.has(p)) return true;
    return false;
  }, [draftPerms, selectedRole]);

  function togglePerm(key: string) {
    if (isAdminRole) return; // admin é imutável
    setDraftPerms((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  }

  const saveMut = useMutation({
    mutationFn: async () => {
      if (!selectedRole) return;
      await api.put(`/roles/${selectedRole.id}/permissions`, {
        permissions: Array.from(draftPerms),
      });
    },
    onSuccess: () => {
      toast.success("Permissões atualizadas");
      qc.invalidateQueries({ queryKey: ["roles"] });
      // Se o role afetado é o do próprio user, recarrega permissions imediatamente
      if (selectedRole?.name === user?.role) refreshUser();
    },
    onError: (err: any) => toast.error(err?.response?.data?.error || "Erro ao salvar"),
  });

  const createMut = useMutation({
    mutationFn: async (data: RoleFormData) => {
      const res = await api.post<Role>("/roles", data);
      return res.data;
    },
    onSuccess: (created) => {
      toast.success("Perfil criado");
      qc.invalidateQueries({ queryKey: ["roles"] });
      setSelectedRoleId(created.id);
    },
  });

  const updateMut = useMutation({
    mutationFn: async (data: RoleFormData) => {
      if (!editingRole) return;
      await api.put(`/roles/${editingRole.id}`, data);
    },
    onSuccess: () => {
      toast.success("Perfil atualizado");
      qc.invalidateQueries({ queryKey: ["roles"] });
    },
  });

  const deleteMut = useMutation({
    mutationFn: (id: number) => api.delete(`/roles/${id}`),
    onSuccess: () => {
      toast.success("Perfil excluído");
      qc.invalidateQueries({ queryKey: ["roles"] });
      setSelectedRoleId(null);
    },
    onError: (err: any) => toast.error(err?.response?.data?.error || "Erro ao excluir"),
  });

  function confirmDelete(role: Role) {
    if (role.is_system) return;
    if (role.user_count > 0) {
      toast.error("Existem usuários atribuídos a este perfil. Reatribua-os antes de excluir.");
      return;
    }
    if (!confirm(`Excluir o perfil "${role.label}"? Esta ação não pode ser desfeita.`)) return;
    deleteMut.mutate(role.id);
  }

  if (rolesLoading || catalogLoading) {
    return (
      <div className="flex justify-center p-12">
        <Loader2 className="w-6 h-6 animate-spin text-brand-500" />
      </div>
    );
  }

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">Permissões</h1>
        <p className="text-sm text-slate-500 mt-0.5">
          Configure o que cada perfil pode fazer. As alterações são aplicadas imediatamente.
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* COLUNA ESQUERDA — lista de perfis */}
        <div className="lg:col-span-4 xl:col-span-3 space-y-3">
          <button
            onClick={() => setShowCreate(true)}
            className="w-full btn-primary justify-center"
          >
            <Plus className="w-4 h-4" /> Novo perfil
          </button>
          <div className="card overflow-hidden">
            <ul className="divide-y divide-slate-100">
              {roles.map((r) => {
                const active = r.id === selectedRoleId;
                return (
                  <li key={r.id}>
                    <button
                      onClick={() => setSelectedRoleId(r.id)}
                      className={`w-full text-left px-4 py-3 transition flex items-start gap-3 ${
                        active ? "bg-brand-50/60" : "hover:bg-slate-50"
                      }`}
                    >
                      <div
                        className={`w-9 h-9 rounded-lg shrink-0 flex items-center justify-center ${
                          r.is_system ? "bg-brand-500/10 text-brand-600" : "bg-slate-100 text-slate-500"
                        }`}
                      >
                        <ShieldCheck className="w-4 h-4" />
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-1.5">
                          <span className={`font-medium truncate ${active ? "text-brand-700" : "text-slate-900"}`}>
                            {r.label}
                          </span>
                          {r.is_system && <Lock className="w-3 h-3 text-slate-400 shrink-0" />}
                        </div>
                        <div className="text-xs text-slate-500 truncate">
                          {r.permissions.length} permissões · {r.user_count} {r.user_count === 1 ? "usuário" : "usuários"}
                        </div>
                      </div>
                      <div className="flex items-center gap-0.5 opacity-60 hover:opacity-100">
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            setEditingRole(r);
                          }}
                          className="p-1 rounded hover:bg-white text-slate-400 hover:text-brand-600"
                          title="Editar perfil"
                        >
                          <Pencil className="w-3.5 h-3.5" />
                        </button>
                        {!r.is_system && (
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              confirmDelete(r);
                            }}
                            className="p-1 rounded hover:bg-white text-slate-400 hover:text-red-600"
                            title="Excluir perfil"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        )}
                      </div>
                    </button>
                  </li>
                );
              })}
            </ul>
          </div>
        </div>

        {/* COLUNA DIREITA — permissões do perfil selecionado */}
        <div className="lg:col-span-8 xl:col-span-9">
          {!selectedRole ? (
            <div className="card p-12 text-center text-slate-400">
              <ShieldCheck className="w-10 h-10 mx-auto mb-3 opacity-30" />
              Selecione um perfil para configurar suas permissões.
            </div>
          ) : (
            <div className="card">
              <div className="px-6 py-5 border-b border-slate-100 flex items-start gap-4">
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <h2 className="text-lg font-semibold text-slate-900">{selectedRole.label}</h2>
                    {selectedRole.is_system && (
                      <span className="text-[10px] font-semibold px-1.5 py-0.5 rounded bg-brand-100 text-brand-700 leading-none">
                        SISTEMA
                      </span>
                    )}
                  </div>
                  <div className="text-xs text-slate-500 font-mono mt-0.5">{selectedRole.name}</div>
                  {selectedRole.description && (
                    <p className="text-sm text-slate-600 mt-2">{selectedRole.description}</p>
                  )}
                </div>
                <button
                  onClick={() => saveMut.mutate()}
                  disabled={!dirty || isAdminRole || saveMut.isPending}
                  className="btn-primary disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  {saveMut.isPending ? (
                    <Loader2 className="w-4 h-4 animate-spin" />
                  ) : (
                    <Save className="w-4 h-4" />
                  )}
                  Salvar
                </button>
              </div>

              {isAdminRole && (
                <div className="px-6 py-3 bg-amber-50 border-b border-amber-100 flex items-center gap-2">
                  <AlertCircle className="w-4 h-4 text-amber-600 shrink-0" />
                  <p className="text-sm text-amber-800">
                    Administrador tem todas as permissões e não pode ser editado.
                  </p>
                </div>
              )}

              <div className="p-6 space-y-6">
                {categories.map((cat) => (
                  <div key={cat}>
                    <h3 className="text-xs font-bold uppercase tracking-wider text-slate-400 mb-3">{cat}</h3>
                    <ul className="space-y-1.5">
                      {grouped.get(cat)!.map((perm) => {
                        const checked = isAdminRole || draftPerms.has(perm.key);
                        return (
                          <li
                            key={perm.key}
                            className={`flex items-start gap-3 p-3 rounded-lg transition ${
                              isAdminRole ? "bg-slate-50/50" : "hover:bg-slate-50"
                            }`}
                          >
                            <div className="flex-1 min-w-0">
                              <div className="text-sm font-medium text-slate-900">{perm.label}</div>
                              <div className="text-xs text-slate-500 mt-0.5">{perm.description}</div>
                              <div className="text-[10px] text-slate-400 font-mono mt-1">{perm.key}</div>
                            </div>
                            <button
                              type="button"
                              onClick={() => togglePerm(perm.key)}
                              disabled={isAdminRole}
                              className={`relative shrink-0 w-11 h-6 rounded-full transition cursor-pointer disabled:cursor-not-allowed ${
                                checked ? "bg-brand-600" : "bg-slate-300"
                              } ${isAdminRole ? "opacity-60" : ""}`}
                              role="switch"
                              aria-checked={checked}
                            >
                              <span
                                className={`absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full shadow transform transition ${
                                  checked ? "translate-x-5" : "translate-x-0"
                                }`}
                              />
                            </button>
                          </li>
                        );
                      })}
                    </ul>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>

      {showCreate && (
        <RoleModal
          onClose={() => setShowCreate(false)}
          onSave={async (data) => {
            await createMut.mutateAsync(data);
          }}
        />
      )}
      {editingRole && (
        <RoleModal
          role={editingRole}
          onClose={() => setEditingRole(null)}
          onSave={async (data) => {
            await updateMut.mutateAsync(data);
          }}
        />
      )}
    </div>
  );
}
