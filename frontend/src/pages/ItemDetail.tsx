import { useRef, useState } from "react";
import { Link, useParams, useNavigate } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Edit3, Trash2, QrCode, Upload, ImageIcon, X, MapPin, Folder, Calendar, DollarSign, Tag, Hash } from "lucide-react";
import toast from "react-hot-toast";
import { api } from "@/lib/api";
import type { Item, Movement } from "@/types";
import Modal from "@/components/ui/Modal";
import ItemForm from "./ItemForm";
import { formatCurrency, formatDate, formatDateTime } from "@/lib/utils";
import { useAuth } from "@/hooks/useAuth";

export default function ItemDetail() {
  const { isViewer } = useAuth();
  const { id } = useParams();
  const nav = useNavigate();
  const qc = useQueryClient();
  const [editOpen, setEditOpen] = useState(false);
  const [qrOpen, setQrOpen] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);

  const { data: item, isLoading } = useQuery({
    queryKey: ["item", id],
    queryFn: async () => (await api.get<Item>(`/items/${id}`)).data,
    enabled: !!id,
  });

  const { data: movements = [] } = useQuery({
    queryKey: ["movements", id],
    queryFn: async () => (await api.get<Movement[]>(`/items/${id}/movements`)).data,
    enabled: !!id,
  });

  const delItem = useMutation({
    mutationFn: async () => api.delete(`/items/${id}`),
    onSuccess: () => {
      toast.success("Item excluído");
      qc.invalidateQueries({ queryKey: ["items"] });
      qc.invalidateQueries({ queryKey: ["dashboard"] });
      nav("/itens");
    },
  });

  const uploadPhoto = useMutation({
    mutationFn: async (file: File) => {
      const fd = new FormData();
      fd.append("photo", file);
      return api.post(`/items/${id}/photos`, fd, {
        headers: { "Content-Type": "multipart/form-data" },
      });
    },
    onSuccess: () => {
      toast.success("Foto adicionada");
      qc.invalidateQueries({ queryKey: ["item", id] });
    },
    onError: () => toast.error("Erro no upload"),
  });

  const delPhoto = useMutation({
    mutationFn: async (photoId: number) => api.delete(`/items/${id}/photos/${photoId}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["item", id] }),
  });

  if (isLoading) return <div className="text-slate-500">Carregando…</div>;
  if (!item) return <div>Item não encontrado.</div>;

  return (
    <>
      <Link to="/itens" className="inline-flex items-center gap-1 text-sm text-slate-500 hover:text-slate-900 mb-4">
        <ArrowLeft className="w-4 h-4" /> Voltar para itens
      </Link>

      <div className="flex items-start justify-between flex-wrap gap-4 mb-6">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">{item.name}</h1>
          <div className="flex items-center gap-2 mt-2 text-sm text-slate-500">
            <span className="badge-gray">
              <Hash className="w-3 h-3" /> {item.code}
            </span>
            <span className="badge-blue">
              {item.quantity} {item.unit}
            </span>
            <span className="badge-yellow">
              <Tag className="w-3 h-3" /> {item.condition}
            </span>
          </div>
        </div>
        <div className="flex gap-2">
          <button onClick={() => setQrOpen(true)} className="btn-secondary">
            <QrCode className="w-4 h-4" /> QR Code
          </button>
          {!isViewer && (
            <>
              <button onClick={() => setEditOpen(true)} className="btn-secondary">
                <Edit3 className="w-4 h-4" /> Editar
              </button>
              <button
                onClick={() => {
                  if (confirm(`Excluir "${item.name}"?`)) delItem.mutate();
                }}
                className="btn-danger"
              >
                <Trash2 className="w-4 h-4" /> Excluir
              </button>
            </>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2 space-y-6">
          <div className="card p-5">
            <h2 className="font-semibold mb-4 text-slate-900">Informações</h2>
            <dl className="grid grid-cols-2 gap-y-3 gap-x-6 text-sm">
              {item.description && (
                <div className="col-span-2">
                  <dt className="text-slate-500">Descrição</dt>
                  <dd className="text-slate-900 mt-0.5">{item.description}</dd>
                </div>
              )}
              <Info label="Marca" value={item.brand} />
              <Info label="Modelo" value={item.model} />
              <Info label="Nº série" value={item.serial_number} />
              <Info label="Condição" value={item.condition} />
              <Info icon={Folder} label="Categoria" value={item.category_name} />
              <Info icon={MapPin} label="Local" value={item.location_path} />
              <Info icon={Calendar} label="Data da compra" value={formatDate(item.purchase_date)} />
              <Info
                icon={DollarSign}
                label="Preço pago"
                value={item.purchase_price ? formatCurrency(item.purchase_price) : null}
              />
              {item.notes && (
                <div className="col-span-2">
                  <dt className="text-slate-500">Notas</dt>
                  <dd className="text-slate-900 mt-0.5 whitespace-pre-wrap">{item.notes}</dd>
                </div>
              )}
            </dl>
          </div>

          <div className="card p-5">
            <div className="flex items-center justify-between mb-4">
              <h2 className="font-semibold text-slate-900">Fotos</h2>
              {!isViewer && (
                <>
                  <button className="btn-secondary text-xs py-1.5" onClick={() => fileRef.current?.click()}>
                    <Upload className="w-3.5 h-3.5" /> Adicionar foto
                  </button>
                  <input
                    ref={fileRef}
                    type="file"
                    accept="image/*"
                    className="hidden"
                    onChange={(e) => {
                      const f = e.target.files?.[0];
                      if (f) uploadPhoto.mutate(f);
                      e.target.value = "";
                    }}
                  />
                </>
              )}
            </div>
            {item.photos && item.photos.length > 0 ? (
              <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-3">
                {item.photos.map((p) => (
                  <div key={p.id} className="relative group aspect-square overflow-hidden rounded-lg border border-slate-200">
                    <img src={p.url} alt="" className="w-full h-full object-cover" />
                    {!isViewer && (
                      <button
                        onClick={() => delPhoto.mutate(p.id)}
                        className="absolute top-1.5 right-1.5 p-1 bg-white/90 rounded-full text-slate-700 hover:text-red-600 opacity-0 group-hover:opacity-100 transition"
                      >
                        <X className="w-3.5 h-3.5" />
                      </button>
                    )}
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-center py-8 text-slate-400 text-sm">
                <ImageIcon className="w-8 h-8 mx-auto mb-2 opacity-50" />
                Nenhuma foto ainda.
              </div>
            )}
          </div>
        </div>

        <div className="card p-5">
          <h2 className="font-semibold mb-4 text-slate-900">Histórico de movimentações</h2>
          {movements.length === 0 ? (
            <div className="text-sm text-slate-400 text-center py-6">Nenhuma movimentação registrada.</div>
          ) : (
            <ul className="space-y-3">
              {movements.map((m) => (
                <li key={m.id} className="text-sm border-l-2 border-brand-200 pl-3">
                  <div className="text-slate-900">
                    {m.from_location_name ? (
                      <>
                        <span className="text-slate-500">de</span> {m.from_location_name}{" "}
                        <span className="text-slate-500">para</span> {m.to_location_name || "—"}
                      </>
                    ) : (
                      <>
                        <span className="text-slate-500">→</span> {m.to_location_name || "Sem local"}
                      </>
                    )}
                  </div>
                  <div className="text-xs text-slate-500 mt-0.5">
                    {formatDateTime(m.created_at)}
                    {m.user_name && ` · ${m.user_name}`}
                    {m.reason && ` · ${m.reason}`}
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>

      <Modal open={editOpen} onClose={() => setEditOpen(false)} title="Editar item" size="lg">
        <ItemForm
          item={item}
          onSaved={() => {
            setEditOpen(false);
            qc.invalidateQueries({ queryKey: ["item", id] });
          }}
        />
      </Modal>

      <Modal open={qrOpen} onClose={() => setQrOpen(false)} title={`QR Code · ${item.code}`} size="sm">
        <div className="flex flex-col items-center gap-4">
          <img src={`/api/items/${item.id}/qrcode`} alt="QR" className="border border-slate-200 rounded-lg" />
          <button onClick={() => window.open(`/api/items/${item.id}/qrcode`, "_blank")} className="btn-secondary">
            <Upload className="w-4 h-4 rotate-180" /> Abrir em nova aba
          </button>
          <p className="text-xs text-slate-500 text-center">
            Cole o QR Code na caixa ou local onde o item está guardado.
          </p>
        </div>
      </Modal>
    </>
  );
}

function Info({ icon: Icon, label, value }: { icon?: any; label: string; value?: string | null }) {
  if (!value) return null;
  return (
    <div>
      <dt className="text-slate-500 flex items-center gap-1.5">
        {Icon && <Icon className="w-3.5 h-3.5" />}
        {label}
      </dt>
      <dd className="text-slate-900 mt-0.5">{value}</dd>
    </div>
  );
}
