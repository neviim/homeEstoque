import { Navigate, Route, Routes } from "react-router-dom";
import { useAuth } from "@/hooks/useAuth";
import Layout from "@/components/layout/Layout";
import Login from "@/pages/Login";
import Dashboard from "@/pages/Dashboard";
import Items from "@/pages/Items";
import ItemDetail from "@/pages/ItemDetail";
import Categories from "@/pages/Categories";
import Locations from "@/pages/Locations";
import Movements from "@/pages/Movements";
import Users from "@/pages/Users";
import Permissions from "@/pages/Permissions";
import BackupPage from "@/pages/Backup";

function Protected({ children }: { children: JSX.Element }) {
  const { user, loading } = useAuth();
  if (loading) return <div className="p-10">Carregando…</div>;
  if (!user) return <Navigate to="/login" replace />;
  return children;
}

function RequirePermission({ perm, fallback = "/", children }: { perm: string; fallback?: string; children: JSX.Element }) {
  const { hasPermission, loading } = useAuth();
  if (loading) return <div className="p-10">Carregando…</div>;
  if (!hasPermission(perm)) return <Navigate to={fallback} replace />;
  return children;
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route
        element={
          <Protected>
            <Layout />
          </Protected>
        }
      >
        <Route path="/" element={<RequirePermission perm="dashboard.view"><Dashboard /></RequirePermission>} />
        <Route path="/itens" element={<RequirePermission perm="items.view"><Items /></RequirePermission>} />
        <Route path="/itens/:id" element={<RequirePermission perm="items.view"><ItemDetail /></RequirePermission>} />
        <Route path="/categorias" element={<RequirePermission perm="categories.view"><Categories /></RequirePermission>} />
        <Route path="/locais" element={<RequirePermission perm="locations.view"><Locations /></RequirePermission>} />
        <Route path="/movimentacoes" element={<RequirePermission perm="movements.view"><Movements /></RequirePermission>} />
        <Route path="/sistema/usuarios" element={<RequirePermission perm="users.manage"><Users /></RequirePermission>} />
        <Route path="/sistema/permissoes" element={<RequirePermission perm="roles.manage"><Permissions /></RequirePermission>} />
        <Route path="/sistema/backup" element={<RequirePermission perm="backup.create"><BackupPage /></RequirePermission>} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
