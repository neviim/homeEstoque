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

function Protected({ children }: { children: JSX.Element }) {
  const { user, loading } = useAuth();
  if (loading) return <div className="p-10">Carregando…</div>;
  if (!user) return <Navigate to="/login" replace />;
  return children;
}

function AdminOnly({ children }: { children: JSX.Element }) {
  const { isAdmin, loading } = useAuth();
  if (loading) return <div className="p-10">Carregando…</div>;
  if (!isAdmin) return <Navigate to="/" replace />;
  return children;
}

function NotViewer({ children }: { children: JSX.Element }) {
  const { isViewer, loading } = useAuth();
  if (loading) return <div className="p-10">Carregando…</div>;
  if (isViewer) return <Navigate to="/itens" replace />;
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
        <Route path="/" element={<Dashboard />} />
        <Route path="/itens" element={<Items />} />
        <Route path="/itens/:id" element={<ItemDetail />} />
        <Route path="/categorias" element={<NotViewer><Categories /></NotViewer>} />
        <Route path="/locais" element={<NotViewer><Locations /></NotViewer>} />
        <Route path="/movimentacoes" element={<NotViewer><Movements /></NotViewer>} />
        <Route
          path="/sistema/usuarios"
          element={
            <AdminOnly>
              <Users />
            </AdminOnly>
          }
        />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
