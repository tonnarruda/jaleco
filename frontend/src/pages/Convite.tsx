import { useEffect, useState, type FormEvent } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { acessoApi, type Convite } from "../lib/api";
import { useSessao } from "../sessao";
import { Marca } from "./Entrar";

const TITULOS: Record<Convite["tipo"], { titulo: string; texto: string }> = {
  "primeiro-admin": {
    titulo: "Configurar o administrador",
    texto: "Você é a primeira pessoa a entrar. Seu cadastro será o de Administrador, com acesso a tudo.",
  },
  convite: { titulo: "Criar sua senha", texto: "Crie a senha que você vai usar para entrar no sistema." },
  redefinicao: { titulo: "Nova senha", texto: "Defina uma nova senha. A antiga deixa de valer." },
};

/** Página pública do link de uso único: primeiro admin, convite ou nova senha. */
export function ConvitePagina() {
  const { token = "" } = useParams();
  const { entrou } = useSessao();
  const navigate = useNavigate();
  const [convite, setConvite] = useState<Convite | null>(null);
  const [invalido, setInvalido] = useState("");
  const [nome, setNome] = useState("");
  const [email, setEmail] = useState("");
  const [senha, setSenha] = useState("");
  const [confirmacao, setConfirmacao] = useState("");
  const [erro, setErro] = useState("");
  const [enviando, setEnviando] = useState(false);

  useEffect(() => {
    acessoApi
      .consultarConvite(token)
      .then(setConvite)
      .catch((e) => setInvalido((e as Error).message));
  }, [token]);

  const primeiroAdmin = convite?.tipo === "primeiro-admin";

  const enviar = async (e: FormEvent) => {
    e.preventDefault();
    if (primeiroAdmin && !nome.trim()) return setErro("Informe seu nome.");
    if (primeiroAdmin && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim())) return setErro("Informe um e-mail válido.");
    if (senha.length < 8) return setErro("A senha precisa ter pelo menos 8 caracteres.");
    if (senha !== confirmacao) return setErro("As duas senhas não são iguais.");
    setErro("");
    setEnviando(true);
    try {
      entrou(await acessoApi.usarConvite(token, primeiroAdmin ? { nome: nome.trim(), email: email.trim(), senha } : { senha }));
      navigate("/", { replace: true });
    } catch (err) {
      setErro((err as Error).message);
    } finally {
      setEnviando(false);
    }
  };

  if (invalido) {
    return (
      <main className="acesso-pagina">
        <div className="acesso-cartao">
          <Marca />
          <h1>Link inválido</h1>
          <p className="sub">{invalido}</p>
          <Link className="btn btn-outline acesso-botao" to="/entrar">
            Ir para a entrada
          </Link>
        </div>
      </main>
    );
  }
  if (!convite) {
    return <div className="tela-carregando">Conferindo o link…</div>;
  }

  const { titulo, texto } = TITULOS[convite.tipo];
  return (
    <main className="acesso-pagina">
      <form className="acesso-cartao" onSubmit={enviar} noValidate>
        <Marca />
        <h1>{titulo}</h1>
        <p className="sub">
          {convite.nome ? <>Olá, <strong>{convite.nome.split(" ")[0]}</strong>! </> : null}
          {texto}
        </p>
        <div className="fields">
          {primeiroAdmin ? (
            <>
              <div className="field">
                <label htmlFor="conv-nome">Seu nome (aparece nas vendas)</label>
                <input id="conv-nome" autoFocus autoComplete="name" value={nome} onChange={(e) => setNome(e.target.value)} />
              </div>
              <div className="field">
                <label htmlFor="conv-email">E-mail (seu login)</label>
                <input
                  id="conv-email"
                  type="email"
                  autoComplete="username"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                />
              </div>
            </>
          ) : (
            <div className="field">
              <label htmlFor="conv-email">E-mail (seu login)</label>
              <input id="conv-email" type="email" autoComplete="username" value={convite.email} readOnly />
            </div>
          )}
          <div className="field">
            <label htmlFor="conv-senha">Senha</label>
            <input
              id="conv-senha"
              type="password"
              autoComplete="new-password"
              autoFocus={!primeiroAdmin}
              value={senha}
              onChange={(e) => setSenha(e.target.value)}
            />
            <p className="hint">Pelo menos 8 caracteres.</p>
          </div>
          <div className="field">
            <label htmlFor="conv-confirmacao">Repita a senha</label>
            <input
              id="conv-confirmacao"
              type="password"
              autoComplete="new-password"
              value={confirmacao}
              onChange={(e) => setConfirmacao(e.target.value)}
            />
          </div>
          {erro ? <p className="erro" role="alert">{erro}</p> : null}
          <button className="btn btn-pink acesso-botao" type="submit" disabled={enviando}>
            {enviando ? "Salvando…" : primeiroAdmin ? "Criar administrador e entrar" : "Salvar senha e entrar"}
          </button>
        </div>
      </form>
    </main>
  );
}
