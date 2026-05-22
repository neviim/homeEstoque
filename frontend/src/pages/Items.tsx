import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Plus, Search, Package, Trash2, Edit3, QrCode, MapPin, Folder } from "lucide-react";
import toast from "react-hot-toast";
import { api } from "@/lib/api";
import type { Item, ItemsPage, Category, Location } from "@/types";
import PageHeader from "@/components/ui/PageHeader";
import EmptyState from "@/components/ui/EmptyState";
import Modal from "@/components/ui/Modal";
import Pagination from "@/components/ui/Pagination";
import ItemForm from "./ItemForm";
import { useAuth } from "@/hooks/useAuth";

const PAGE_SIZE = 12;

export default function Items() {
  const { hasPermission } = useAuth();
  const canCreate = hasPermission("items.create");
  const canUpdate = hasPermission("items.update");
  const canDelete = hasPermission("items.delete");
  const qc = useQueryClient();
  const [search, setSearch] = useState("");
  const [categoryId, setCategoryId] = useState<string>("");
  const [locationId, setLocationId] = useState<string>("");
  const [page, setPage] = useState(1);
  const [showForm, setShowForm] = useState(false);
  const [editItem, setEditItem] = useState<Item | null>(null);
  const [qrItem, setQrItem] = useState<Item | null>(null);

  const params = useMemo(() => {
    const p: Record<string, string> = { page: String(page), limit: String(PAGE_SIZE) };
    if (search) p.search = search;
    if (categoryId) p.category_id = categoryId;
    if (locationId) p.location_id = locationId;
    return p;
  }, [search, categoryId, locationId, page]);

  const { data, isLoading } = useQuery({
    queryKey: ["items", params],
    queryFn: async () => (await api.get<ItemsPage>("/items", { params })).data,
    placeholderData: (prev) => prev,
  });

  const items = data?.items ?? [];
  const total = data?.total ?? 0;
  const totalPages = data?.total_pages ?? 1;

  const { data: categories = [] } = useQuery({
    queryKey: ["categories"],
    queryFn: async () => (await api.get<Category[]>("/categories")).data,
  });
  const { data: locations = [] } = useQuery({
    queryKey: ["locations"],
    queryFn: async () => (await api.get<Location[]>("/locations")).data,
  });

  const delMut = useMutation({
    mutationFn: async (id: number) => api.delete(`/items/${id}`),
    onSuccess: () => {
      toast.success("Item removido");
      qc.invalidateQueries({ queryKey: ["items"] });
      qc.invalidateQueries({ queryKey: ["dashboard"] });
    },
  });

  function onFilterChange(fn: () => void) {
    fn();
    setPage(1);
  }

  function onEdit(item: Item) {
    setEditItem(item);
    setShowForm(true);
  }

  function onNew() {
    setEditItem(null);
    setShowForm(true);
  }

  function onClose() {
    setShowForm(false);
    setEditItem(null);
  }

  return (
    <>
      <PageHeader
        title="Itens"
        subtitle={`${total} ${total === 1 ? "item" : "itens"} no estoque`}
        actions={
          canCreate && (
            <button onClick={onNew} className="btn-primary">
              <Plus className="w-4 h-4" /> Novo item
            </button>
          )
        }
      />

      <div className="card p-4 mb-5">
        <div className="grid grid-cols-1 md:grid-cols-12 gap-3">
          <div className="md:col-span-6 relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
            <input
              className="input pl-9"
              placeholder="Buscar por nome, descrição, código, marca…"
              value={search}
              onChange={(e) => onFilterChange(() => setSearch(e.target.value))}
            />
          </div>
          <select className="input md:col-span-3" value={categoryId} onChange={(e) => onFilterChange(() => setCategoryId(e.target.value))}>
            <option value="">Todas categorias</option>
            {categories.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
          <select className="input md:col-span-3" value={locationId} onChange={(e) => onFilterChange(() => setLocationId(e.target.value))}>
            <option value="">Todos locais</option>
            {locations.map((l) => (
              <option key={l.id} value={l.id}>
                {l.full_path || l.name}
              </option>
            ))}
          </select>
        </div>
      </div>

      {isLoading ? (
        <div className="text-slate-500">Carregando…</div>
      ) : items.length === 0 ? (
        <EmptyState
          icon={Package}
          title="Nenhum item encontrado"
          description="Comece cadastrando os itens que você quer controlar em casa."
          action={
            canCreate ? (
              <button className="btn-primary" onClick={onNew}>
                <Plus className="w-4 h-4" /> Cadastrar primeiro item
              </button>
            ) : undefined
          }
        />
      ) : (
        <>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {items.map((item) => (
            <div key={item.id} className="card p-4 hover:shadow-md hover:border-brand-200 transition group">
              <div className="flex items-start justify-between gap-2 mb-3">
                <Link to={`/itens/${item.id}`} className="flex items-center gap-3 min-w-0 flex-1">
                  <div className="w-10 h-10 rounded-lg bg-brand-50 text-brand-600 flex items-center justify-center shrink-0">
                    <Package className="w-5 h-5" />
                  </div>
                  <div className="min-w-0">
                    <div className="font-semibold text-slate-900 truncate group-hover:text-brand-700">
                      {item.name}
                    </div>
                    <div className="text-xs text-slate-500 truncate">{item.code}</div>
                  </div>
                </Link>
                <div className="badge-blue shrink-0">
                  {item.quantity} {item.unit}
                </div>
              </div>

              {item.description && (
                <p className="text-sm text-slate-600 line-clamp-2 mb-3">{item.description}</p>
              )}

              <div className="flex flex-wrap gap-2 mb-3">
                {item.category_name && (
                  <span className="badge-gray">
                    <Folder className="w-3 h-3" /> {item.category_name}
                  </span>
                )}
                {item.location_path && (
                  <span className="badge-green">
                    <MapPin className="w-3 h-3" /> {item.location_path}
                  </span>
                )}
              </div>

              <div className="flex items-center justify-between pt-3 border-t border-slate-100">
                <Link to={`/itens/${item.id}`} className="text-xs text-brand-600 hover:underline">
                  Ver detalhes →
                </Link>
                <div className="flex items-center gap-1">
                  <button
                    onClick={() => setQrItem(item)}
                    title="QR Code"
                    className="p-1.5 text-slate-400 hover:text-brand-600 hover:bg-brand-50 rounded"
                  >
                    <QrCode className="w-4 h-4" />
                  </button>
                  {canUpdate && (
                    <button
                      onClick={() => onEdit(item)}
                      title="Editar"
                      className="p-1.5 text-slate-400 hover:text-brand-600 hover:bg-brand-50 rounded"
                    >
                      <Edit3 className="w-4 h-4" />
                    </button>
                  )}
                  {canDelete && (
                    <button
                      onClick={() => {
                        if (confirm(`Excluir "${item.name}"?`)) delMut.mutate(item.id);
                      }}
                      title="Excluir"
                      className="p-1.5 text-slate-400 hover:text-red-600 hover:bg-red-50 rounded"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>

        {totalPages > 1 && (
          <Pagination page={page} totalPages={totalPages} total={total} pageSize={PAGE_SIZE} onPage={setPage} />
        )}
        </>
      )}

      <Modal open={showForm} onClose={onClose} title={editItem ? "Editar item" : "Novo item"} size="lg">
        <ItemForm item={editItem} onSaved={onClose} />
      </Modal>

      <Modal open={!!qrItem} onClose={() => setQrItem(null)} title="QR Code" size="sm">
        {qrItem && (
          <div className="flex flex-col items-center gap-4">
            <p className="text-sm text-slate-500 text-center">{qrItem.name}</p>
            <div className="p-3 border border-slate-200 rounded-xl bg-white">
              <img
                src={`/api/items/${qrItem.id}/qrcode`}
                alt={`QR Code — ${qrItem.name}`}
                className="w-64 h-64"
              />
            </div>
            <p className="text-xs text-slate-400 font-mono">{qrItem.code}</p>
            <a
              href={`/api/items/${qrItem.id}/qrcode`}
              download={`qrcode-${qrItem.code}.png`}
              className="btn-secondary text-sm w-full text-center"
            >
              Baixar imagem
            </a>
          </div>
        )}
      </Modal>
    </>
  );
}
