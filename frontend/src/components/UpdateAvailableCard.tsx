import { ArrowRight, Download, Sparkles } from "lucide-react";
import { useVersion } from "@/hooks/useVersion";

export default function UpdateAvailableCard() {
  const { running, available, updateAvailable, dismissed, dismiss, apply } = useVersion();

  if (!updateAvailable || dismissed) return null;

  return (
    <div className="card mb-8 p-4 bg-gradient-to-br from-indigo-50 via-white to-violet-50 border border-indigo-100 relative overflow-hidden animate-fade-in">
      <div className="flex items-center gap-4">
        <div className="relative w-10 h-10 rounded-xl bg-white shadow-sm flex items-center justify-center text-indigo-600 shrink-0">
          <Sparkles className="w-5 h-5" />
          <span className="absolute inset-0 rounded-xl ring-2 ring-indigo-300 animate-update-glow" />
        </div>

        <div className="flex-1 min-w-0">
          <div className="text-sm font-semibold text-slate-900">Nova versão disponível</div>
          <div className="text-xs text-slate-600 mt-0.5 flex items-center gap-1">
            <span className="font-mono">{running}</span>
            <ArrowRight className="w-3 h-3 text-slate-400" />
            <span className="font-mono font-semibold text-indigo-600">{available}</span>
          </div>
        </div>

        <div className="flex items-center gap-2 shrink-0">
          <button
            onClick={dismiss}
            className="text-xs text-slate-500 hover:text-slate-700 px-2 py-1.5 rounded transition"
          >
            Lembrar depois
          </button>
          <button
            onClick={apply}
            className="btn-primary text-xs gap-1.5"
          >
            <Download className="w-3.5 h-3.5" />
            Aplicar agora
          </button>
        </div>
      </div>
    </div>
  );
}
