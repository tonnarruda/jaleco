import { NavLink, Outlet } from "react-router-dom";
import { useSessao } from "../sessao";
import { useStore } from "../store";
import { Notificacoes } from "./Notificacoes";

const links = [
  { to: "/", label: "Painel" },
  { to: "/pedidos", label: "Pedidos" },
  { to: "/clientes", label: "Clientes" },
  { to: "/estoque", label: "Estoque" },
  { to: "/equipe", label: "Equipe" },
];

export function Layout() {
  const store = useStore();
  const { sair } = useSessao();
  const visiveis = links.filter((l) => l.to !== "/equipe" || store.perm.verEquipe);

  return (
    <>
      <header className="app-header">
        <div className="container header-inner">
          <NavLink to="/" className="brand">
            <span className="heart" aria-hidden>
              ♥
            </span>
            <span className="brand-name">I Love Jalecos</span>
            <span className="brand-sub">GESTÃO</span>
          </NavLink>
          <nav className="nav">
            {visiveis.map((l) => (
              <NavLink key={l.to} to={l.to} end={l.to === "/"}>
                {l.label}
              </NavLink>
            ))}
            <div className="nav-user" title={store.usuario.email}>
              <span className="nav-user-nome">
                <strong>{store.usuario.nome}</strong>
                <small>{store.usuario.papel}</small>
              </span>
              <button type="button" className="btn btn-outline btn-sm" onClick={sair}>
                Sair
              </button>
            </div>
          </nav>
        </div>
      </header>
      {store.conexao.status === "erro" ? (
        <div className="aviso-conexao" role="alert">
          <div className="container">{store.conexao.mensagem}</div>
        </div>
      ) : null}
      <Outlet />
      <Notificacoes />
    </>
  );
}
