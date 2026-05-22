import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, Folder, Trash2, Edit3, Loader2 } from "lucide-react";
import toast from "react-hot-toast";
import { api } from "@/lib/api";
import type { Category } from "@/types";
import PageHeader from "@/components/ui/PageHeader";
import Modal from "@/components/ui/Modal";
import EmptyState from "@/components/ui/EmptyState";
import { useAuth } from "@/hooks/useAuth";

const COLORS = [
  "#3b82f6", "#6366f1", "#8b5cf6", "#ec4899",
  "#f59e0b", "#eab308", "#10b981", "#22c55e",
  "#06b6d4", "#f97316", "#64748b", "#94a3b8",
];

export default function Categories() {
  const { hasPermission } = useAuth();
  const canManage = hasPermission("categories.manage");
  const qc = useQueryClient();
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<Category | null>(null);

  const { data: categories = [], isLoading } = useQuery({
    queryKey: ["categories"],
    queryFn: async () => (await api.get<Category[]>("/categories")).data,
  });

  const del = useMutation({
    mutationFn: async (id: number) => api.delete(`/categories/${id}`),
    onSuccess: () => {
      toast.success("Categoria removida");
      qc.invalidateQueries({ queryKey: ["categories"] });
    },
  });

  function onNew() {
    setEditing(null);
    setOpen(true);
  }
  function onEdit(c: Category) {
    setEditing(c);
    setOpen(true);
  }

  return (
    <>
      <PageHeader
        title="Categorias"
        subtitle="Organize seus itens por tipo"
        actions={
          canManage && (
            <button onClick={onNew} className="btn-primary">
              <Plus className="w-4 h-4" /> Nova categoria
            </button>
          )
        }
      />

      {isLoading ? (
        <div className="text-slate-500">Carregando…</div>
      ) : categories.length === 0 ? (
        <EmptyState
          icon={Folder}
          title="Nenhuma categoria"
          description="Crie categorias para organizar seus itens (ex: Eletrônicos, Cabos, Cozinha)."
          action={
            canManage ? (
              <button className="btn-primary" onClick={onNew}>
                <Plus className="w-4 h-4" /> Criar categoria
              </button>
            ) : undefined
          }
        />
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {categories.map((c) => (
            <div key={c.id} className="card p-4 flex items-center gap-3 group hover:shadow-md transition">
              <div
                className="w-11 h-11 rounded-lg flex items-center justify-center"
                style={{ background: (c.color || "#94a3b8") + "20", color: c.color || "#64748b" }}
              >
                <Folder className="w-5 h-5" />
              </div>
              <div className="flex-1 min-w-0">
                <div className="font-semibold text-slate-900 truncate">{c.name}</div>
                <div className="text-xs text-slate-500">{c.item_count ?? 0} itens</div>
              </div>
              {canManage && (
                <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition">
                  <button onClick={() => onEdit(c)} className="p-1.5 text-slate-400 hover:text-brand-600 hover:bg-brand-50 rounded">
                    <Edit3 className="w-4 h-4" />
                  </button>
                  <button
                    onClick={() => {
                      if (confirm(`Excluir categoria "${c.name}"? Itens existentes ficarão sem categoria.`))
                        del.mutate(c.id);
                    }}
                    className="p-1.5 text-slate-400 hover:text-red-600 hover:bg-red-50 rounded"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      <Modal open={open} onClose={() => setOpen(false)} title={editing ? "Editar categoria" : "Nova categoria"}>
        <CategoryForm category={editing} onSaved={() => setOpen(false)} />
      </Modal>
    </>
  );
}

function CategoryForm({ category, onSaved }: { category: Category | null; onSaved: () => void }) {
  const qc = useQueryClient();
  const [name, setName] = useState(category?.name || "");
  const [color, setColor] = useState(category?.color || COLORS[0]);

  const save = useMutation({
    mutationFn: async () => {
      const payload = { name, color, icon: "" };
      if (category) return api.put(`/categories/${category.id}`, payload);
      return api.post("/categories", payload);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["categories"] });
      toast.success(category ? "Atualizada" : "Criada");
      onSaved();
    },
  });

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        save.mutate();
      }}
      className="space-y-4"
    >
      <div>
        <label className="label">Nome *</label>
        <input className="input" value={name} onChange={(e) => setName(e.target.value)} required />
      </div>
      <div>
        <label className="label">Cor</label>
        <div className="flex flex-wrap gap-2">
          {COLORS.map((c) => (
            <button
              key={c}
              type="button"
              onClick={() => setColor(c)}
              className={`w-8 h-8 rounded-lg border-2 transition ${
                color === c ? "border-slate-900 scale-110" : "border-transparent"
              }`}
              style={{ background: c }}
            />
          ))}
        </div>
      </div>
      <div className="flex justify-end gap-2 pt-2">
        <button type="button" className="btn-secondary" onClick={onSaved}>
          Cancelar
        </button>
        <button type="submit" className="btn-primary" disabled={save.isPending}>
          {save.isPending && <Loader2 className="w-4 h-4 animate-spin" />}
          Salvar
        </button>
      </div>
    </form>
  );
}
