import { BrowserRouter, Navigate, Route, Routes, useLocation } from "react-router-dom";
import { ConfirmacaoProvider } from "./components/Confirmacao";
import { Layout } from "./components/Layout";
import { Clientes } from "./pages/Clientes";
import { ConvitePagina } from "./pages/Convite";
import { Entrar } from "./pages/Entrar";
import { Equipe } from "./pages/Equipe";
import { Estoque } from "./pages/Estoque";
import { Painel } from "./pages/Painel";
import { Pedidos } from "./pages/Pedidos";
import { SessaoProvider, useSessao } from "./sessao";
import { StoreProvider } from "./store";

export function App() {
  return (
    <SessaoProvider>
      <ConfirmacaoProvider>
        <BrowserRouter>
          <Routes>
            <Route path="/entrar" element={<Entrar />} />
            <Route path="/convite/:token" element={<ConvitePagina />} />
            <Route element={<AreaLogada />}>
              <Route path="/" element={<Painel />} />
              <Route path="/pedidos" element={<Pedidos />} />
              <Route path="/clientes" element={<Clientes />} />
              <Route path="/estoque" element={<Estoque />} />
              <Route path="/equipe" element={<Equipe />} />
              <Route path="*" element={<Navigate to="/" replace />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </ConfirmacaoProvider>
    </SessaoProvider>
  );
}

/** Sem sessão, vai para a entrada e volta para onde estava depois de entrar. */
function AreaLogada() {
  const { usuario } = useSessao();
  const location = useLocation();
  if (usuario === undefined) return <div className="tela-carregando">Carregando…</div>;
  if (!usuario) return <Navigate to="/entrar" replace state={{ de: location.pathname }} />;
  // key: trocar de pessoa remonta o store do zero, sem dados da sessão anterior.
  return (
    <StoreProvider key={usuario.id} usuario={usuario}>
      <Layout />
    </StoreProvider>
  );
}
