import { useState } from "react";
import { Link, NavLink, Outlet, useLocation } from "react-router-dom";
import {
  LayoutDashboard,
  Package,
  Folder,
  MapPin,
  ArrowLeftRight,
  Boxes,
  Download,
  Settings,
  Users,
  Shield,
  ChevronDown,
  HardDrive,
} from "lucide-react";
import { useAuth } from "@/hooks/useAuth";
import { useVersion } from "@/hooks/useVersion";
import { cn } from "@/lib/utils";
import ProfileModal from "@/components/ui/ProfileModal";

interface NavItem {
  to: string;
  label: string;
  icon: typeof LayoutDashboard;
  end?: boolean;
  requires: string; // permission key
}

const navItems: NavItem[] = [
  { to: "/", label: "Dashboard", icon: LayoutDashboard, end: true, requires: "dashboard.view" },
  { to: "/itens", label: "Itens", icon: Package, requires: "items.view" },
  { to: "/categorias", label: "Categorias", icon: Folder, requires: "categories.view" },
  { to: "/locais", label: "Locais", icon: MapPin, requires: "locations.view" },
  { to: "/movimentacoes", label: "Movimentações", icon: ArrowLeftRight, requires: "movements.view" },
];

export default function Layout() {
  const { user, isViewer, hasPermission } = useAuth();
  const { running, updateAvailable, dismissed, available } = useVersion();
  const [showProfile, setShowProfile] = useState(false);
  const location = useLocation();
  const sistemaOpen = location.pathname.startsWith("/sistema");
  const [sistemaExpanded, setSistemaExpanded] = useState(sistemaOpen);

  const canSeeSistema =
    hasPermission("users.manage") || hasPermission("roles.manage") || hasPermission("export.csv") || hasPermission("backup.create");

  return (
    <div className="min-h-screen flex">
      <aside className="w-64 bg-white border-r border-slate-200 flex flex-col">
        <div className="px-5 py-6 flex items-center gap-2.5 border-b border-slate-100">
          <div className="w-9 h-9 rounded-lg bg-gradient-to-br from-brand-500 to-brand-700 flex items-center justify-center shadow-sm">
            <Boxes className="w-5 h-5 text-white" />
          </div>
          <div>
            <div className="font-bold text-slate-900 text-base leading-tight">HomeEstoque</div>
            <div className="text-xs text-slate-500">Controle inteligente</div>
          </div>
        </div>

        <nav className="flex-1 px-3 py-4 space-y-1">
          {navItems.filter((item) => hasPermission(item.requires)).map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              className={({ isActive }) =>
                cn(
                  "flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition",
                  isActive
                    ? "bg-brand-50 text-brand-700"
                    : "text-slate-600 hover:bg-slate-50 hover:text-slate-900"
                )
              }
            >
              <item.icon className="w-[18px] h-[18px]" />
              {item.label}
            </NavLink>
          ))}

          {canSeeSistema && (
            <div className="pt-2 mt-2 border-t border-slate-100">
              <button
                onClick={() => setSistemaExpanded((v) => !v)}
                className={cn(
                  "w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition",
                  sistemaOpen
                    ? "text-brand-700 bg-brand-50/60"
                    : "text-slate-600 hover:bg-slate-50 hover:text-slate-900"
                )}
              >
                <Settings className="w-[18px] h-[18px] shrink-0" />
                <span className="flex-1 text-left">Sistema</span>
                <ChevronDown
                  className={cn("w-4 h-4 shrink-0 transition-transform", sistemaExpanded && "rotate-180")}
                />
              </button>

              {sistemaExpanded && (
                <div className="ml-3 mt-0.5 pl-4 border-l border-slate-200 space-y-0.5">
                  {hasPermission("users.manage") && (
                    <NavLink
                      to="/sistema/usuarios"
                      className={({ isActive }) =>
                        cn(
                          "flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition",
                          isActive
                            ? "bg-brand-50 text-brand-700"
                            : "text-slate-600 hover:bg-slate-50 hover:text-slate-900"
                        )
                      }
                    >
                      <Users className="w-[16px] h-[16px]" />
                      Usuários
                    </NavLink>
                  )}
                  {hasPermission("roles.manage") && (
                    <NavLink
                      to="/sistema/permissoes"
                      className={({ isActive }) =>
                        cn(
                          "flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition",
                          isActive
                            ? "bg-brand-50 text-brand-700"
                            : "text-slate-600 hover:bg-slate-50 hover:text-slate-900"
                        )
                      }
                    >
                      <Shield className="w-[16px] h-[16px]" />
                      Permissões
                    </NavLink>
                  )}
                  {hasPermission("backup.create") && (
                    <NavLink
                      to="/sistema/backup"
                      className={({ isActive }) =>
                        cn(
                          "flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition",
                          isActive
                            ? "bg-brand-50 text-brand-700"
                            : "text-slate-600 hover:bg-slate-50 hover:text-slate-900"
                        )
                      }
                    >
                      <HardDrive className="w-[16px] h-[16px]" />
                      Backup
                    </NavLink>
                  )}
                  {hasPermission("export.csv") && (
                    <a
                      href="/api/export/csv"
                      className="flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium text-slate-600 hover:bg-slate-50 hover:text-slate-900 transition"
                      onClick={(e) => {
                        e.preventDefault();
                        const token = localStorage.getItem("token");
                        fetch("/api/export/csv", { headers: { Authorization: `Bearer ${token}` } })
                          .then((r) => r.blob())
                          .then((blob) => {
                            const url = URL.createObjectURL(blob);
                            const a = document.createElement("a");
                            a.href = url;
                            a.download = "estoque.csv";
                            a.click();
                            URL.revokeObjectURL(url);
                          });
                      }}
                    >
                      <Download className="w-[16px] h-[16px]" />
                      Exportar CSV
                    </a>
                  )}
                </div>
              )}
            </div>
          )}
        </nav>

        <div className="border-t border-slate-100 p-3">
          <div className="flex justify-end items-center gap-1.5 px-2 mb-1 h-3">
            <span className="text-[10px] text-slate-400 tabular-nums">v{running}</span>
            {updateAvailable && !dismissed && (
              <Link
                to="/"
                className="inline-flex items-center gap-1 text-[10px] text-amber-600 hover:text-amber-700"
                title={`Nova versão v${available} disponível — aplicar no Dashboard`}
              >
                <span className="w-1.5 h-1.5 rounded-full bg-amber-400 animate-pulse" />
                atualização
              </Link>
            )}
          </div>
          <div className="flex items-center gap-3 px-2 py-2">
            <button
              onClick={() => setShowProfile(true)}
              className="w-9 h-9 rounded-full bg-gradient-to-br from-brand-500 to-indigo-600 text-white flex items-center justify-center font-semibold text-sm hover:ring-2 hover:ring-brand-300 transition shrink-0"
              title="Ver perfil"
            >
              {user?.name?.[0]?.toUpperCase() || "?"}
            </button>
            <button
              onClick={() => setShowProfile(true)}
              className="flex-1 min-w-0 text-left hover:opacity-80 transition"
            >
              <div className="text-sm font-medium text-slate-900 truncate flex items-center gap-1.5">
                {user?.name}
                {isViewer && (
                  <span className="text-[10px] font-semibold px-1.5 py-0.5 rounded bg-purple-100 text-purple-700 leading-none">
                    Visualizador
                  </span>
                )}
              </div>
              <div className="text-xs text-slate-500 truncate">{user?.email}</div>
            </button>
          </div>
        </div>

        <ProfileModal open={showProfile} onClose={() => setShowProfile(false)} />
      </aside>

      <main className="flex-1 overflow-auto">
        <div className="max-w-7xl mx-auto px-8 py-8">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
