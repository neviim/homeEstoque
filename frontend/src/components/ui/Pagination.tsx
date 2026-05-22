import { ChevronLeft, ChevronRight } from "lucide-react";

function getPageNumbers(current: number, total: number): (number | "…")[] {
  if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1);
  const pages: (number | "…")[] = [1];
  if (current > 3) pages.push("…");
  for (let p = Math.max(2, current - 1); p <= Math.min(total - 1, current + 1); p++) pages.push(p);
  if (current < total - 2) pages.push("…");
  pages.push(total);
  return pages;
}

interface PaginationProps {
  page: number;
  totalPages: number;
  total: number;
  pageSize: number;
  onPage: (p: number) => void;
}

export default function Pagination({ page, totalPages, total, pageSize, onPage }: PaginationProps) {
  const from = (page - 1) * pageSize + 1;
  const to = Math.min(page * pageSize, total);

  const btnBase =
    "flex items-center justify-center w-9 h-9 rounded-lg text-sm font-medium transition-all duration-150 select-none";
  const btnActive = "bg-brand-600 text-white shadow-sm";
  const btnIdle = "text-slate-600 hover:bg-slate-100";
  const btnDisabled = "text-slate-300 cursor-not-allowed";

  return (
    <div className="flex flex-col sm:flex-row items-center justify-between gap-3 mt-6 pt-5 border-t border-slate-100">
      <p className="text-sm text-slate-500 order-2 sm:order-1">
        Exibindo{" "}
        <span className="font-medium text-slate-700">{from}–{to}</span> de{" "}
        <span className="font-medium text-slate-700">{total}</span>{" "}
        {total === 1 ? "registro" : "registros"}
      </p>

      <div className="flex items-center gap-1 order-1 sm:order-2">
        <button
          onClick={() => onPage(page - 1)}
          disabled={page === 1}
          className={`${btnBase} ${page === 1 ? btnDisabled : btnIdle} gap-1 px-3 w-auto`}
        >
          <ChevronLeft className="w-4 h-4" />
          <span className="hidden sm:inline">Anterior</span>
        </button>

        <div className="flex items-center gap-1 mx-1">
          {getPageNumbers(page, totalPages).map((p, i) =>
            p === "…" ? (
              <span key={`e${i}`} className="w-9 h-9 flex items-center justify-center text-slate-400 text-sm">
                ···
              </span>
            ) : (
              <button
                key={p}
                onClick={() => onPage(p as number)}
                className={`${btnBase} ${page === p ? btnActive : btnIdle}`}
              >
                {p}
              </button>
            )
          )}
        </div>

        <button
          onClick={() => onPage(page + 1)}
          disabled={page === totalPages}
          className={`${btnBase} ${page === totalPages ? btnDisabled : btnIdle} gap-1 px-3 w-auto`}
        >
          <span className="hidden sm:inline">Próxima</span>
          <ChevronRight className="w-4 h-4" />
        </button>
      </div>
    </div>
  );
}
