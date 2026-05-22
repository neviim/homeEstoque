import { useEffect, useState } from "react";
import { X, User, Lock, Eye, EyeOff, Calendar, ShieldCheck, Check } from "lucide-react";
import toast from "react-hot-toast";
import { api } from "@/lib/api";
import { useAuth } from "@/hooks/useAuth";
import { formatDate } from "@/lib/utils";

type Tab = "perfil" | "seguranca";

// Gradientes baseados no primeiro char do nome — cor consistente por usuário
const PALETTES = [
  { banner: "from-violet-600 via-indigo-600 to-indigo-700", avatar: "from-violet-500 to-indigo-600" },
  { banner: "from-blue-600 via-blue-600 to-cyan-700",        avatar: "from-blue-500 to-cyan-600"     },
  { banner: "from-emerald-600 via-teal-600 to-teal-700",     avatar: "from-emerald-500 to-teal-600"  },
  { banner: "from-amber-500 via-orange-500 to-orange-600",   avatar: "from-amber-400 to-orange-500"  },
  { banner: "from-rose-600 via-pink-600 to-pink-700",        avatar: "from-rose-500 to-pink-600"     },
  { banner: "from-purple-600 via-violet-600 to-violet-700",  avatar: "from-purple-500 to-violet-600" },
  { banner: "from-sky-500 via-sky-600 to-blue-700",          avatar: "from-sky-400 to-blue-600"      },
  { banner: "from-green-600 via-emerald-600 to-emerald-700", avatar: "from-green-500 to-emerald-600" },
];

function palette(name: string) {
  return PALETTES[(name.charCodeAt(0) || 0) % PALETTES.length];
}

function pwdStrength(pwd: string): { label: string; color: string; width: string } {
  if (!pwd) return { label: "", color: "", width: "w-0" };
  if (pwd.length < 6)  return { label: "Fraca",  color: "bg-red-500",    width: "w-1/4" };
  if (pwd.length < 10) return { label: "Média",  color: "bg-amber-400",  width: "w-2/4" };
  if (pwd.length < 14) return { label: "Boa",    color: "bg-brand-500",  width: "w-3/4" };
  return                      { label: "Forte",  color: "bg-emerald-500", width: "w-full" };
}

interface Props {
  open: boolean;
  onClose: () => void;
}

export default function ProfileModal({ open, onClose }: Props) {
  const { user, updateUser } = useAuth();
  const [tab, setTab] = useState<Tab>("perfil");

  // Perfil
  const [name, setName] = useState("");
  const [saving, setSaving] = useState(false);

  // Senha
  const [currentPwd, setCurrentPwd]   = useState("");
  const [newPwd, setNewPwd]           = useState("");
  const [confirmPwd, setConfirmPwd]   = useState("");
  const [showCurrent, setShowCurrent] = useState(false);
  const [showNew, setShowNew]         = useState(false);
  const [changingPwd, setChangingPwd] = useState(false);

  useEffect(() => {
    if (open && user) {
      setName(user.name);
      setCurrentPwd("");
      setNewPwd("");
      setConfirmPwd("");
      setTab("perfil");
    }
  }, [open]); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    if (open) document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!open || !user) return null;

  const pal     = palette(user.name);
  const initial = user.name[0]?.toUpperCase() ?? "?";
  const strength = pwdStrength(newPwd);
  const pwdMatch = confirmPwd.length > 0 && newPwd === confirmPwd;

  async function handleSaveProfile(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) {
      toast.error("Nome é obrigatório");
      return;
    }
    setSaving(true);
    try {
      const { data } = await api.put("/auth/profile", { name: name.trim() });
      updateUser(data);
      toast.success("Perfil atualizado!");
    } catch (err: any) {
      toast.error(err?.response?.data?.error ?? "Erro ao salvar");
    } finally {
      setSaving(false);
    }
  }

  async function handleChangePassword(e: React.FormEvent) {
    e.preventDefault();
    if (newPwd !== confirmPwd) { toast.error("As senhas não coincidem"); return; }
    if (newPwd.length < 6)    { toast.error("Mínimo 6 caracteres"); return; }
    setChangingPwd(true);
    try {
      await api.put("/auth/password", { current_password: currentPwd, new_password: newPwd });
      toast.success("Senha alterada com sucesso!");
      setCurrentPwd(""); setNewPwd(""); setConfirmPwd("");
    } catch (err: any) {
      toast.error(err?.response?.data?.error ?? "Erro ao alterar senha");
    } finally {
      setChangingPwd(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 animate-fade-in">
      <div className="absolute inset-0 bg-slate-900/60 backdrop-blur-sm" onClick={onClose} />

      <div className="relative bg-[#f2f3f5] rounded-2xl shadow-2xl w-full max-w-md overflow-hidden animate-slide-up">

        {/* Banner */}
        <div className={`h-24 bg-gradient-to-br ${pal.banner}`} />

        {/* Fechar */}
        <button
          onClick={onClose}
          className="absolute top-3 right-3 z-10 p-1.5 bg-black/25 hover:bg-black/45 text-white rounded-full transition"
        >
          <X className="w-4 h-4" />
        </button>

        {/* Avatar + info */}
        <div className="bg-white mx-3 -mt-4 rounded-xl shadow-sm px-5 pt-0 pb-5">
          {/* Avatar sobreposição */}
          <div className="flex items-end gap-4 -mt-8 mb-4">
            <div
              className={`w-20 h-20 rounded-full bg-gradient-to-br ${pal.avatar} ring-4 ring-white flex items-center justify-center text-white text-3xl font-bold shadow-lg shrink-0`}
            >
              {initial}
            </div>
          </div>

          <div>
            <h2 className="text-xl font-bold text-slate-900 leading-tight">{user.name}</h2>
            <p className="text-sm text-slate-500 mt-0.5">{user.email}</p>
            {user.created_at && (
              <p className="text-xs text-slate-400 mt-1.5 flex items-center gap-1.5">
                <Calendar className="w-3 h-3" />
                Membro desde {formatDate(user.created_at)}
              </p>
            )}
          </div>
        </div>

        {/* Painel de edição */}
        <div className="bg-white mx-3 mt-2 mb-3 rounded-xl shadow-sm overflow-hidden">
          {/* Tabs */}
          <div className="flex border-b border-slate-100 px-2 pt-2 gap-1">
            {(["perfil", "seguranca"] as Tab[]).map((t) => (
              <button
                key={t}
                onClick={() => setTab(t)}
                className={`flex items-center gap-1.5 px-4 py-2 text-sm font-medium rounded-t-lg transition border-b-2 ${
                  tab === t
                    ? "text-brand-700 border-brand-600 bg-brand-50/60"
                    : "text-slate-500 border-transparent hover:text-slate-800 hover:bg-slate-50"
                }`}
              >
                {t === "perfil" ? (
                  <><User className="w-3.5 h-3.5" /> Perfil</>
                ) : (
                  <><ShieldCheck className="w-3.5 h-3.5" /> Segurança</>
                )}
              </button>
            ))}
          </div>

          <div className="p-5">
            {/* ───── TAB PERFIL ───── */}
            {tab === "perfil" && (
              <form onSubmit={handleSaveProfile} className="space-y-4">
                <div>
                  <label className="block text-xs font-semibold text-slate-400 uppercase tracking-widest mb-1.5">
                    Nome de exibição
                  </label>
                  <div className="relative">
                    <User className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
                    <input
                      className="input pl-9 bg-[#f2f3f5] border-transparent focus:bg-white"
                      value={name}
                      onChange={(e) => setName(e.target.value)}
                      placeholder="Seu nome"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-400 uppercase tracking-widest mb-1.5">
                    Endereço de e-mail
                  </label>
                  <div className="flex items-center gap-2 px-3 py-2.5 rounded-lg bg-slate-100 text-slate-500 text-sm select-all cursor-default">
                    {user.email}
                  </div>
                  <p className="text-xs text-slate-400 mt-1">O e-mail não pode ser alterado após o cadastro.</p>
                </div>

                <button
                  type="submit"
                  disabled={saving || name === user.name}
                  className="btn-primary w-full disabled:opacity-50"
                >
                  {saving ? "Salvando…" : "Salvar alterações"}
                </button>
              </form>
            )}

            {/* ───── TAB SEGURANÇA ───── */}
            {tab === "seguranca" && (
              <form onSubmit={handleChangePassword} className="space-y-4">
                <div>
                  <label className="block text-xs font-semibold text-slate-400 uppercase tracking-widest mb-1.5">
                    Senha atual
                  </label>
                  <div className="relative">
                    <Lock className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
                    <input
                      className="input pl-9 pr-10 bg-[#f2f3f5] border-transparent focus:bg-white"
                      type={showCurrent ? "text" : "password"}
                      value={currentPwd}
                      onChange={(e) => setCurrentPwd(e.target.value)}
                      placeholder="••••••••"
                      autoComplete="current-password"
                    />
                    <button
                      type="button"
                      onClick={() => setShowCurrent(!showCurrent)}
                      className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600"
                    >
                      {showCurrent ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                    </button>
                  </div>
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-400 uppercase tracking-widest mb-1.5">
                    Nova senha
                  </label>
                  <div className="relative">
                    <Lock className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
                    <input
                      className="input pl-9 pr-10 bg-[#f2f3f5] border-transparent focus:bg-white"
                      type={showNew ? "text" : "password"}
                      value={newPwd}
                      onChange={(e) => setNewPwd(e.target.value)}
                      placeholder="Mínimo 6 caracteres"
                      autoComplete="new-password"
                    />
                    <button
                      type="button"
                      onClick={() => setShowNew(!showNew)}
                      className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600"
                    >
                      {showNew ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                    </button>
                  </div>
                  {newPwd && (
                    <div className="mt-2">
                      <div className="h-1.5 bg-slate-100 rounded-full overflow-hidden">
                        <div className={`h-full rounded-full transition-all duration-300 ${strength.color} ${strength.width}`} />
                      </div>
                      <p className={`text-xs mt-1 ${strength.color.replace("bg-", "text-")}`}>
                        Força: {strength.label}
                      </p>
                    </div>
                  )}
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-400 uppercase tracking-widest mb-1.5">
                    Confirmar nova senha
                  </label>
                  <div className="relative">
                    <Lock className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
                    <input
                      className={`input pl-9 pr-10 bg-[#f2f3f5] border-transparent focus:bg-white ${
                        confirmPwd && !pwdMatch ? "border-red-300 focus:ring-red-200" : ""
                      }`}
                      type="password"
                      value={confirmPwd}
                      onChange={(e) => setConfirmPwd(e.target.value)}
                      placeholder="••••••••"
                      autoComplete="new-password"
                    />
                    {confirmPwd && (
                      <div className={`absolute right-3 top-1/2 -translate-y-1/2 ${pwdMatch ? "text-emerald-500" : "text-red-400"}`}>
                        {pwdMatch ? <Check className="w-4 h-4" /> : <X className="w-4 h-4" />}
                      </div>
                    )}
                  </div>
                  {confirmPwd && !pwdMatch && (
                    <p className="text-xs text-red-500 mt-1">As senhas não coincidem</p>
                  )}
                </div>

                <button
                  type="submit"
                  disabled={changingPwd || !currentPwd || !newPwd || !confirmPwd || !pwdMatch}
                  className="btn-primary w-full disabled:opacity-50"
                >
                  {changingPwd ? "Alterando…" : "Alterar senha"}
                </button>
              </form>
            )}
          </div>
        </div>

      </div>
    </div>
  );
}
