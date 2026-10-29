import { useEffect, useState, type FormEvent } from "react";
import { Navigate, useLocation, useNavigate } from "react-router-dom";
import { acessoApi } from "../lib/api";
import { useSessao } from "../sessao";

export function Entrar() {
  const { usuario, aviso, entrou } = useSessao();
  const navigate = useNavigate();
  const destino = (useLocation().state as { de?: string } | null)?.de ?? "/";
  const [email, setEmail] = useState("");
  const [senha, setSenha] = useState("");
  const [erro, setErro] = useState("");
  const [enviando, setEnviando] = useState(false);
  // Sistema recém-instalado: sem administrador, o botão de criar o primeiro aparece.
  const [semAdministrador, setSemAdministrador] = useState(false);

  useEffect(() => {
    acessoApi
      .configuracao()
      .then((c) => setSemAdministrador(!c.temAdministrador))
      .catch(() => setSemAdministrador(false));
  }, []);

  if (usuario) return <Navigate to={destino} replace />;

  const gerarAdministrador = async () => {
    setErro("");
    try {
      const { token } = await acessoApi.primeiroAcesso();
      navigate(`/convite/${token}`);
    } catch (err) {
      // Alguém criou o admin enquanto esta tela estava aberta: o botão some.
      setSemAdministrador(false);
      setErro((err as Error).message);
    }
  };

  const enviar = async (e: FormEvent) => {
    e.preventDefault();
    setErro("");
    setEnviando(true);
    try {
      entrou(await acessoApi.entrar(email.trim(), senha));
      navigate(destino, { replace: true });
    } catch (err) {
      setErro((err as Error).message);
      setSenha("");
    } finally {
      setEnviando(false);
    }
  };

  return (
    <main className="acesso-pagina">
      <form className="acesso-cartao" onSubmit={enviar} noValidate>
        <Marca />
        <h1>Entrar</h1>
        {semAdministrador ? (
          <div className="primeiro-acesso">
            <strong>Sistema ainda sem administrador.</strong>
            <span>Se você é a pessoa responsável pela loja, crie o primeiro acesso.</span>
            <button type="button" className="btn btn-pink" onClick={gerarAdministrador}>
              Gerar usuário Administrador
            </button>
          </div>
        ) : null}
        {aviso && !erro ? <p className="hint warn">{aviso}</p> : null}
        <div className="fields">
          <div className="field">
            <label htmlFor="login-email">E-mail</label>
            <input
              id="login-email"
              type="email"
              autoComplete="username"
              autoFocus
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </div>
          <div className="field">
            <label htmlFor="login-senha">Senha</label>
            <input
              id="login-senha"
              type="password"
              autoComplete="current-password"
              value={senha}
              onChange={(e) => setSenha(e.target.value)}
            />
          </div>
          {erro ? <p className="erro" role="alert">{erro}</p> : null}
          <button className="btn btn-pink acesso-botao" type="submit" disabled={enviando || !email || !senha}>
            {enviando ? "Entrando…" : "Entrar"}
          </button>
        </div>
        <p className="help acesso-rodape">Esqueceu a senha? Peça ao administrador um link para criar uma nova.</p>
      </form>
    </main>
  );
}

export function Marca() {
  return (
    <div className="brand acesso-marca">
      <span className="heart" aria-hidden>
        ♥
      </span>
      <span className="brand-name">I Love Jalecos</span>
      <span className="brand-sub">GESTÃO</span>
    </div>
  );
}
