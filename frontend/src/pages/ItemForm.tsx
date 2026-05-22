import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import toast from "react-hot-toast";
import { Loader2 } from "lucide-react";
import { api } from "@/lib/api";
import type { Category, Item, Location } from "@/types";

interface Props {
  item: Item | null;
  onSaved: () => void;
}

export default function ItemForm({ item, onSaved }: Props) {
  const qc = useQueryClient();
  const [form, setForm] = useState<Partial<Item>>(
    item || {
      name: "",
      description: "",
      brand: "",
      model: "",
      serial_number: "",
      quantity: 1,
      unit: "un",
      condition: "novo",
      category_id: null,
      location_id: null,
      notes: "",
    }
  );

  const { data: categories = [] } = useQuery({
    queryKey: ["categories"],
    queryFn: async () => (await api.get<Category[]>("/categories")).data,
  });
  const { data: locations = [] } = useQuery({
    queryKey: ["locations"],
    queryFn: async () => (await api.get<Location[]>("/locations")).data,
  });

  const save = useMutation({
    mutationFn: async () => {
      const payload = {
        ...form,
        category_id: form.category_id ? Number(form.category_id) : null,
        location_id: form.location_id ? Number(form.location_id) : null,
        quantity: Number(form.quantity || 1),
        purchase_price: form.purchase_price ? Number(form.purchase_price) : null,
      };
      if (item) return api.put(`/items/${item.id}`, payload);
      return api.post("/items", payload);
    },
    onSuccess: () => {
      toast.success(item ? "Item atualizado" : "Item criado");
      qc.invalidateQueries({ queryKey: ["items"] });
      qc.invalidateQueries({ queryKey: ["dashboard"] });
      onSaved();
    },
    onError: (e: any) => {
      toast.error(e?.response?.data?.error || "Erro ao salvar");
    },
  });

  function set<K extends keyof Item>(key: K, value: any) {
    setForm((f) => ({ ...f, [key]: value }));
  }

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        save.mutate();
      }}
      className="space-y-4"
    >
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div className="sm:col-span-2">
          <label className="label">Nome *</label>
          <input
            className="input"
            value={form.name || ""}
            onChange={(e) => set("name", e.target.value)}
            required
            placeholder="Ex: Notebook Dell Inspiron"
          />
        </div>

        <div>
          <label className="label">Marca</label>
          <input className="input" value={form.brand || ""} onChange={(e) => set("brand", e.target.value)} />
        </div>
        <div>
          <label className="label">Modelo</label>
          <input className="input" value={form.model || ""} onChange={(e) => set("model", e.target.value)} />
        </div>
        <div>
          <label className="label">Nº de série</label>
          <input
            className="input"
            value={form.serial_number || ""}
            onChange={(e) => set("serial_number", e.target.value)}
          />
        </div>
        <div>
          <label className="label">Condição</label>
          <select
            className="input"
            value={form.condition || "novo"}
            onChange={(e) => set("condition", e.target.value)}
          >
            <option value="novo">Novo</option>
            <option value="usado">Usado</option>
            <option value="bom">Bom estado</option>
            <option value="reparo">Necessita reparo</option>
            <option value="descarte">Para descarte</option>
          </select>
        </div>

        <div>
          <label className="label">Quantidade</label>
          <input
            type="number"
            min={1}
            className="input"
            value={form.quantity ?? 1}
            onChange={(e) => set("quantity", e.target.value)}
          />
        </div>
        <div>
          <label className="label">Unidade</label>
          <input
            className="input"
            value={form.unit || "un"}
            onChange={(e) => set("unit", e.target.value)}
            placeholder="un, kg, m, l"
          />
        </div>

        <div>
          <label className="label">Categoria</label>
          <select
            className="input"
            value={form.category_id || ""}
            onChange={(e) => set("category_id", e.target.value || null)}
          >
            <option value="">— sem categoria —</option>
            {categories.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className="label">Local</label>
          <select
            className="input"
            value={form.location_id || ""}
            onChange={(e) => set("location_id", e.target.value || null)}
          >
            <option value="">— sem local —</option>
            {locations.map((l) => (
              <option key={l.id} value={l.id}>
                {l.full_path || l.name}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label className="label">Data da compra</label>
          <input
            type="date"
            className="input"
            value={form.purchase_date || ""}
            onChange={(e) => set("purchase_date", e.target.value || null)}
          />
        </div>
        <div>
          <label className="label">Preço pago (R$)</label>
          <input
            type="number"
            step="0.01"
            className="input"
            value={form.purchase_price ?? ""}
            onChange={(e) => set("purchase_price", e.target.value)}
          />
        </div>

        <div className="sm:col-span-2">
          <label className="label">Descrição</label>
          <textarea
            className="input min-h-[70px]"
            value={form.description || ""}
            onChange={(e) => set("description", e.target.value)}
          />
        </div>
        <div className="sm:col-span-2">
          <label className="label">Notas</label>
          <textarea
            className="input min-h-[60px]"
            value={form.notes || ""}
            onChange={(e) => set("notes", e.target.value)}
            placeholder="Garantia, observações…"
          />
        </div>
      </div>

      <div className="flex justify-end gap-2 pt-4 border-t border-slate-100">
        <button type="button" className="btn-secondary" onClick={onSaved}>
          Cancelar
        </button>
        <button type="submit" className="btn-primary" disabled={save.isPending}>
          {save.isPending && <Loader2 className="w-4 h-4 animate-spin" />}
          {item ? "Salvar alterações" : "Criar item"}
        </button>
      </div>
    </form>
  );
}
