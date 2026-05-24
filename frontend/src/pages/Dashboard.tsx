import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { api } from "@/lib/api";
import type { DashboardStats } from "@/types";
import { Package, Boxes, Folder, MapPin, TrendingUp, ArrowRight, History } from "lucide-react";
import PageHeader from "@/components/ui/PageHeader";
import { formatCurrency, formatDateTime } from "@/lib/utils";
import { useAuth } from "@/hooks/useAuth";

export default function Dashboard() {
  const { hasPermission } = useAuth();
  const { data, isLoading } = useQuery({
    queryKey: ["dashboard"],
    queryFn: async () => (await api.get<DashboardStats>("/dashboard")).data,
  });

  if (isLoading) return <div className="text-slate-500">Carregando dashboard…</div>;
  if (!data) return null;

  const cards = [
    { label: "Itens cadastrados", value: data.total_items, icon: Package, color: "bg-blue-50 text-blue-600" },
    { label: "Quantidade total", value: data.total_quantity, icon: Boxes, color: "bg-violet-50 text-violet-600" },
    { label: "Categorias", value: data.total_categories, icon: Folder, color: "bg-amber-50 text-amber-600" },
    { label: "Locais", value: data.total_locations, icon: MapPin, color: "bg-emerald-50 text-emerald-600" },
  ];

  return (
    <>
      <PageHeader
        title="Dashboard"
        subtitle="Visão geral do seu estoque doméstico"
        actions={
          <Link to="/itens" className="btn-primary">
            <Package className="w-4 h-4" /> Ver todos os itens
          </Link>
        }
      />

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        {cards.map((c) => (
          <div key={c.label} className="card p-5 hover:shadow-md transition">
            <div className="flex items-start justify-between">
              <div>
                <div className="text-sm text-slate-500">{c.label}</div>
                <div className="text-3xl font-bold mt-2 text-slate-900">{c.value}</div>
              </div>
              <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${c.color}`}>
                <c.icon className="w-5 h-5" />
              </div>
            </div>
          </div>
        ))}
      </div>

      {hasPermission("dashboard.view_value") && data.total_value > 0 && (
        <div className="card p-6 mb-8 bg-gradient-to-br from-brand-50 to-indigo-50 border-brand-100">
          <div className="flex items-center gap-3">
            <div className="w-12 h-12 rounded-xl bg-white shadow-sm flex items-center justify-center text-brand-600">
              <TrendingUp className="w-6 h-6" />
            </div>
            <div>
              <div className="text-sm text-slate-600">Valor patrimonial estimado</div>
              <div className="text-2xl font-bold text-slate-900">{formatCurrency(data.total_value)}</div>
            </div>
          </div>
        </div>
      )}

      <div className="card mb-8">
        <div className="px-5 py-4 border-b border-slate-100 flex items-center gap-2">
          <History className="w-4 h-4 text-slate-500" />
          <h2 className="font-semibold text-slate-900">Últimos itens modificados</h2>
        </div>
        {data.updated_items.length === 0 ? (
          <div className="px-5 py-8 text-sm text-slate-500 text-center">Nenhum item modificado ainda.</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-slate-50 text-slate-500 text-xs uppercase tracking-wide">
                <tr>
                  <th className="text-left font-medium px-5 py-3">Item</th>
                  <th className="text-left font-medium px-5 py-3">Categoria</th>
                  <th className="text-left font-medium px-5 py-3">Local</th>
                  <th className="text-right font-medium px-5 py-3">Quantidade</th>
                  <th className="text-right font-medium px-5 py-3">Modificado em</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {data.updated_items.map((item) => (
                  <tr key={item.id} className="hover:bg-slate-50 transition">
                    <td className="px-5 py-3">
                      <Link to={`/itens/${item.id}`} className="font-medium text-slate-900 hover:text-brand-600">
                        {item.name}
                      </Link>
                    </td>
                    <td className="px-5 py-3 text-slate-600">{item.category_name || "—"}</td>
                    <td className="px-5 py-3 text-slate-600">{item.location_path || "—"}</td>
                    <td className="px-5 py-3 text-right text-slate-900">
                      {item.quantity} {item.unit}
                    </td>
                    <td className="px-5 py-3 text-right text-xs text-slate-500">{formatDateTime(item.updated_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="card">
          <div className="px-5 py-4 border-b border-slate-100 flex items-center justify-between">
            <h2 className="font-semibold text-slate-900">Itens recentes</h2>
            <Link to="/itens" className="text-sm text-brand-600 hover:underline inline-flex items-center gap-1">
              ver todos <ArrowRight className="w-3.5 h-3.5" />
            </Link>
          </div>
          <ul className="divide-y divide-slate-100">
            {data.recent_items.length === 0 && (
              <li className="px-5 py-8 text-sm text-slate-500 text-center">Nenhum item cadastrado ainda.</li>
            )}
            {data.recent_items.map((item) => (
              <li key={item.id}>
                <Link
                  to={`/itens/${item.id}`}
                  className="flex items-center gap-3 px-5 py-3 hover:bg-slate-50 transition"
                >
                  <div className="w-9 h-9 rounded-lg bg-slate-100 flex items-center justify-center text-slate-500">
                    <Package className="w-4 h-4" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="text-sm font-medium text-slate-900 truncate">{item.name}</div>
                    <div className="text-xs text-slate-500 truncate">
                      {item.category_name || "Sem categoria"} · {item.location_path || "Sem local"}
                    </div>
                  </div>
                  <div className="text-xs text-slate-400">{formatDateTime(item.created_at)}</div>
                </Link>
              </li>
            ))}
          </ul>
        </div>

        <div className="card">
          <div className="px-5 py-4 border-b border-slate-100">
            <h2 className="font-semibold text-slate-900">Top categorias</h2>
          </div>
          <ul className="divide-y divide-slate-100">
            {data.top_categories.length === 0 && (
              <li className="px-5 py-8 text-sm text-slate-500 text-center">Cadastre categorias para começar.</li>
            )}
            {data.top_categories.map((c) => (
              <li key={c.id} className="px-5 py-3 flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div
                    className="w-8 h-8 rounded-lg flex items-center justify-center"
                    style={{ background: (c.color || "#94a3b8") + "22", color: c.color || "#64748b" }}
                  >
                    <Folder className="w-4 h-4" />
                  </div>
                  <div className="text-sm font-medium text-slate-900">{c.name}</div>
                </div>
                <div className="badge-blue">{c.item_count} itens</div>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </>
  );
}
