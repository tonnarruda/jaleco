import type { StoreState, Usuario } from "../types";

type Metodo = "GET" | "PUT" | "POST" | "PATCH" | "DELETE";

/** Erro com o status HTTP: 401 quer dizer que a sessão terminou. */
export class ErroApi extends Error {
  constructor(
    mensagem: string,
    readonly status: number,
  ) {
    super(mensagem);
  }
}

/** A sessão vai no cookie (HttpOnly, mesmo endereço do app): nada de token no JavaScript. */
export async function requisitar<T>(metodo: Metodo, caminho: string, corpo?: unknown): Promise<T> {
  let resp: Response;
  try {
    resp = await fetch(`/api${caminho}`, {
      method: metodo,
      credentials: "same-origin",
      headers: corpo === undefined ? {} : { "Content-Type": "application/json" },
      body: corpo === undefined ? undefined : JSON.stringify(corpo),
    });
  } catch {
    throw new ErroApi("Sem conexão com o servidor.", 0);
  }
  if (resp.status === 204) return undefined as T;
  const dados = await resp.json().catch(() => null);
  if (!resp.ok) {
    // Erro do backend sempre traz { erro }; 5xx sem isso é proxy sem resposta da API.
    if (dados?.erro) throw new ErroApi(dados.erro, resp.status);
    throw new ErroApi(
      resp.status >= 500 ? "Sem conexão com o servidor." : `O servidor respondeu ${resp.status}.`,
      resp.status,
    );
  }
  return dados as T;
}

/** Toda escrita nos dados devolve o estado que o papel pode ver, já com as regras do servidor. */
export const chamarApi = (metodo: Metodo, caminho: string, corpo?: unknown) =>
  requisitar<StoreState>(metodo, caminho, corpo);

export type Convite = { tipo: "primeiro-admin" | "convite" | "redefinicao"; nome: string; email: string };
export type LinkAcesso = { token: string; tipo: "convite" | "redefinicao"; expiraEm: string };

export const acessoApi = {
  sessao: () => requisitar<Usuario>("GET", "/sessao"),
  entrar: (email: string, senha: string) => requisitar<Usuario>("POST", "/login", { email, senha }),
  sair: () => requisitar<void>("POST", "/logout"),
  consultarConvite: (token: string) => requisitar<Convite>("GET", `/convites/${encodeURIComponent(token)}`),
  usarConvite: (token: string, dados: { nome?: string; email?: string; senha: string }) =>
    requisitar<Usuario>("POST", `/convites/${encodeURIComponent(token)}`, dados),
  configuracao: () => requisitar<{ temAdministrador: boolean }>("GET", "/configuracao"),
  /** Só funciona enquanto não existe nenhum administrador. */
  primeiroAcesso: () => requisitar<{ token: string }>("POST", "/primeiro-acesso"),
  gerarLink: (vendedorId: string) => requisitar<LinkAcesso>("POST", `/vendedores/${vendedorId}/convite`),
};
