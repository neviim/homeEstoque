import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { ArrowRight, ArrowLeftRight, User } from "lucide-react";
import { api } from "@/lib/api";
import type { MovementsPage } from "@/types";
import PageHeader from "@/components/ui/PageHeader";
import EmptyState from "@/components/ui/EmptyState";
import Pagination from "@/components/ui/Pagination";
import { formatDateTime } from "@/lib/utils";

const PAGE_SIZE = 15;

interface MovementUser { id: number; name: string }

export default function Movements() {
  const [page, setPage] = useState(1);
  const [userId, setUserId] = useState("");

  function onFilterChange(fn: () => void) {
    fn();
    setPage(1);
  }

  const params = useMemo(() => {
    const p: Record<string, string> = { page: String(page), limit: String(PAGE_SIZE) };
    if (userId) p.user_id = userId;
    return p;
  }, [page, userId]);

  const { data, isLoading } = useQuery({
    queryKey: ["all-movements", params],
    queryFn: async () =>
      (await api.get<MovementsPage>("/movements", { params })).data,
    placeholderData: (prev) => prev,
  });

  const { data: users = [] } = useQuery({
    queryKey: ["movement-users"],
    queryFn: async () => (await api.get<MovementUser[]>("/movements/users")).data,
  });

  const movements = data?.movements ?? [];
  const total = data?.total ?? 0;
  const totalPages = data?.total_pages ?? 1;

  return (
    <>
      <PageHeader
        title="Movimentações"
        subtitle={
          total > 0
            ? `${total} ${total === 1 ? "registro" : "registros"} no histórico`
            : "Histórico de quando seus itens mudaram de local"
        }
      />

      {users.length > 0 && (
        <div className="card p-4 mb-5">
          <div className="flex items-center gap-3">
            <User className="w-4 h-4 text-slate-400 shrink-0" />
            <select
              className="input flex-1 max-w-xs"
              value={userId}
              onChange={(e) => onFilterChange(() => setUserId(e.target.value))}
            >
              <option value="">Todos os usuários</option>
              {users.map((u) => (
                <option key={u.id} value={u.id}>
                  {u.name}
                </option>
              ))}
            </select>
            {userId && (
              <button
                onClick={() => onFilterChange(() => setUserId(""))}
                className="text-xs text-slate-400 hover:text-slate-600"
              >
                Limpar filtro
              </button>
            )}
          </div>
        </div>
      )}

      {isLoading ? (
        <div className="text-slate-500">Carregando…</div>
      ) : movements.length === 0 ? (
        <EmptyState
          icon={ArrowLeftRight}
          title="Sem movimentações"
          description="Ao criar ou mover itens entre locais, o histórico aparece aqui."
        />
      ) : (
        <>
          <div className="card divide-y divide-slate-100">
            {movements.map((m) => (
              <div key={m.id} className="px-5 py-3 flex items-center gap-4">
                <div className="w-9 h-9 rounded-lg bg-brand-50 text-brand-600 flex items-center justify-center shrink-0">
                  <ArrowLeftRight className="w-4 h-4" />
                </div>
                <div className="flex-1 min-w-0">
                  <Link to={`/itens/${m.item_id}`} className="font-medium text-slate-900 hover:text-brand-600">
                    {m.item_name || `Item #${m.item_id}`}
                  </Link>
                  <div className="text-xs text-slate-500 flex items-center gap-1.5 mt-0.5 flex-wrap">
                    <span>{m.from_location_name || "Origem desconhecida"}</span>
                    <ArrowRight className="w-3 h-3" />
                    <span className="font-medium text-slate-700">{m.to_location_name || "—"}</span>
                    {m.reason && <span className="text-slate-400">· {m.reason}</span>}
                  </div>
                </div>
                <div className="text-xs text-slate-400 shrink-0 text-right">
                  <div>{formatDateTime(m.created_at)}</div>
                  {m.user_name && (
                    <div className={m.user_name === "MCP Assistant" ? "text-brand-400 font-medium" : "text-slate-300"}>
                      por {m.user_name}
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>

          {totalPages > 1 && (
            <Pagination
              page={page}
              totalPages={totalPages}
              total={total}
              pageSize={PAGE_SIZE}
              onPage={setPage}
            />
          )}
        </>
      )}
    </>
  );
}
