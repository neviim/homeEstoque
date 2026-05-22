import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, MapPin, Trash2, Edit3, Loader2, Home, Archive, Box, LayoutGrid } from "lucide-react";
import toast from "react-hot-toast";
import { api } from "@/lib/api";
import type { Location } from "@/types";
import PageHeader from "@/components/ui/PageHeader";
import Modal from "@/components/ui/Modal";
import EmptyState from "@/components/ui/EmptyState";
import { useAuth } from "@/hooks/useAuth";

const TYPES = [
  { value: "comodo", label: "Cômodo", icon: Home },
  { value: "movel", label: "Móvel / Bancada", icon: LayoutGrid },
  { value: "caixa", label: "Caixa / Container", icon: Box },
  { value: "armario", label: "Armário / Gaveta", icon: Archive },
  { value: "outro", label: "Outro", icon: MapPin },
];

function typeIcon(t: string) {
  const found = TYPES.find((x) => x.value === t);
  return found?.icon || MapPin;
}

export default function Locations() {
  const { hasPermission } = useAuth();
  const canManage = hasPermission("locations.manage");
  const qc = useQueryClient();
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<Location | null>(null);

  const { data: locations = [], isLoading } = useQuery({
    queryKey: ["locations"],
    queryFn: async () => (await api.get<Location[]>("/locations")).data,
  });

  const del = useMutation({
    mutationFn: async (id: number) => api.delete(`/locations/${id}`),
    onSuccess: () => {
      toast.success("Local removido");
      qc.invalidateQueries({ queryKey: ["locations"] });
    },
  });

  function onNew() {
    setEditing(null);
    setOpen(true);
  }
  function onEdit(l: Location) {
    setEditing(l);
    setOpen(true);
  }

  return (
    <>
      <PageHeader
        title="Locais"
        subtitle="Cômodos, móveis, caixas — onde guardar seus itens"
        actions={
          canManage && (
            <button onClick={onNew} className="btn-primary">
              <Plus className="w-4 h-4" /> Novo local
            </button>
          )
        }
      />

      {isLoading ? (
        <div className="text-slate-500">Carregando…</div>
      ) : locations.length === 0 ? (
        <EmptyState
          icon={MapPin}
          title="Nenhum local cadastrado"
          description="Crie locais como 'Sala', 'Bancada do escritório', 'Caixa 1'."
          action={
            canManage ? (
              <button className="btn-primary" onClick={onNew}>
                <Plus className="w-4 h-4" /> Criar local
              </button>
            ) : undefined
          }
        />
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {locations.map((l) => {
            const Icon = typeIcon(l.type);
            return (
              <div key={l.id} className="card p-4 flex items-center gap-3 group hover:shadow-md transition">
                <div className="w-11 h-11 rounded-lg bg-emerald-50 text-emerald-600 flex items-center justify-center">
                  <Icon className="w-5 h-5" />
                </div>
                <div className="flex-1 min-w-0">
                  <div className="font-semibold text-slate-900 truncate">{l.full_path || l.name}</div>
                  <div className="text-xs text-slate-500">
                    {l.type} · {l.item_count ?? 0} itens
                  </div>
                </div>
                {canManage && (
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition">
                    <button onClick={() => onEdit(l)} className="p-1.5 text-slate-400 hover:text-brand-600 hover:bg-brand-50 rounded">
                      <Edit3 className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => {
                        if (confirm(`Excluir local "${l.name}"?`)) del.mutate(l.id);
                      }}
                      className="p-1.5 text-slate-400 hover:text-red-600 hover:bg-red-50 rounded"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}

      <Modal open={open} onClose={() => setOpen(false)} title={editing ? "Editar local" : "Novo local"}>
        <LocationForm location={editing} locations={locations} onSaved={() => setOpen(false)} />
      </Modal>
    </>
  );
}

function LocationForm({
  location,
  locations,
  onSaved,
}: {
  location: Location | null;
  locations: Location[];
  onSaved: () => void;
}) {
  const qc = useQueryClient();
  const [name, setName] = useState(location?.name || "");
  const [type, setType] = useState(location?.type || "comodo");
  const [parentId, setParentId] = useState<string>(location?.parent_id ? String(location.parent_id) : "");
  const [description, setDescription] = useState(location?.description || "");

  const save = useMutation({
    mutationFn: async () => {
      const payload = {
        name,
        type,
        description,
        parent_id: parentId ? Number(parentId) : null,
      };
      if (location) return api.put(`/locations/${location.id}`, payload);
      return api.post("/locations", payload);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["locations"] });
      toast.success(location ? "Atualizado" : "Criado");
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
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="label">Tipo</label>
          <select className="input" value={type} onChange={(e) => setType(e.target.value)}>
            {TYPES.map((t) => (
              <option key={t.value} value={t.value}>
                {t.label}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className="label">Dentro de</label>
          <select className="input" value={parentId} onChange={(e) => setParentId(e.target.value)}>
            <option value="">— raiz —</option>
            {locations
              .filter((l) => l.id !== location?.id)
              .map((l) => (
                <option key={l.id} value={l.id}>
                  {l.full_path || l.name}
                </option>
              ))}
          </select>
        </div>
      </div>
      <div>
        <label className="label">Descrição</label>
        <textarea
          className="input min-h-[60px]"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
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
