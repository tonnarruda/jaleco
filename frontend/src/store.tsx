import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { chamarApi, ErroApi } from "./lib/api";
import { uid } from "./lib/format";
import { chaveVariacao, consomeEstoque } from "./lib/pedido";
import type {
  AjusteEstoque,
  Cliente,
  ItemEstoque,
  PapelUsuario,
  PecaPedido,
  Pedido,
  PedidoForm,
  StoreState,
  Usuario,
  Vendedor,
} from "./types";
import { useSessao } from "./sessao";

const empty: StoreState = {
  clientes: [],
  pedidos: [],
  estoque: [],
  vendedores: [],
  modelos: [],
};

/** Soma (sinal 1) ou baixa (sinal -1) as peças no estoque. Variação sem cadastro é ignorada. */
function movimentar(estoque: ItemEstoque[], pecas: PecaPedido[], sinal: 1 | -1): ItemEstoque[] {
  const next = estoque.map((i) => ({ ...i }));
  for (const peca of pecas) {
    const item = next.find((i) => chaveVariacao(i) === chaveVariacao(peca));
    if (item) item.quantidade = Math.max(0, item.quantidade + sinal * peca.quantidade);
  }
  return next;
}

export type Permissoes = {
  exportar: boolean;
  excluir: boolean;
  ajustarEstoque: boolean;
  verFaturamento: boolean;
  verEquipe: boolean;
  gerirEquipe: boolean;
};

export function permissoesDe(papel: PapelUsuario): Permissoes {
  const admin = papel === "Administrador";
  const gestor = admin || papel === "Gerente";
  return {
    exportar: gestor,
    excluir: gestor,
    ajustarEstoque: gestor,
    verFaturamento: gestor,
    verEquipe: gestor,
    gerirEquipe: admin,
  };
}

export type Conexao =
  | { status: "sincronizando" }
  | { status: "ok" }
  | { status: "erro"; mensagem: string };

export type Notificacao = { id: string; texto: string };

/** Texto do aviso de sucesso; função quando depende do que o servidor devolveu (ex.: número do pedido). */
type MensagemSucesso = string | ((servidor: StoreState) => string);

const DURACAO_NOTIFICACAO_MS = 4000;

type StoreApi = StoreState & {
  conexao: Conexao;
  notificacoes: Notificacao[];
  dispensarNotificacao: (id: string) => void;
  usuario: Usuario;
  perm: Permissoes;
  nomeCliente: (id: string) => string;
  nomeVendedor: (id: string) => string;
  salvarCliente: (cliente: Omit<Cliente, "id"> & { id?: string }) => Cliente;
  excluirCliente: (id: string) => void;
  salvarPedido: (pedido: PedidoForm) => void;
  excluirPedido: (id: string) => void;
  entradaEstoque: (item: Omit<ItemEstoque, "id">) => void;
  /** Os tamanhos editados juntos são gravados numa transação só. */
  ajustarEstoque: (ajustes: AjusteEstoque[]) => void;
  /** Várias ids saem juntas (produto com todos os tamanhos) numa transação só. */
  removerEstoque: (ids: string[]) => void;
  salvarVendedor: (vendedor: Omit<Vendedor, "id"> & { id?: string }) => void;
  excluirVendedor: (id: string) => void;
};

const StoreContext = createContext<StoreApi | null>(null);

/** Só monta com alguém logado (App garante): o usuário vem da sessão, não da tela. */
export function StoreProvider({ usuario, children }: { usuario: Usuario; children: ReactNode }) {
  const { expirou, atualizar: atualizarSessao } = useSessao();
  const [state, setState] = useState<StoreState>(empty);
  const [conexao, setConexao] = useState<Conexao>({ status: "sincronizando" });
  const [notificacoes, setNotificacoes] = useState<Notificacao[]>([]);

  const dispensarNotificacao = useCallback(
    (id: string) => setNotificacoes((ns) => ns.filter((n) => n.id !== id)),
    [],
  );
  const notificar = useCallback(
    (texto: string) => {
      const id = uid();
      setNotificacoes((ns) => [...ns, { id, texto }]);
      setTimeout(() => dispensarNotificacao(id), DURACAO_NOTIFICACAO_MS);
    },
    [dispensarNotificacao],
  );
  // Fila: uma escrita por vez, para a resposta de uma não sobrescrever a seguinte fora de ordem.
  const fila = useRef<Promise<unknown>>(Promise.resolve());

  // A resposta do servidor é a verdade: substitui a prévia montada na tela.
  const aplicarServidor = setState;

  const sincronizar = useCallback(async () => {
    setConexao({ status: "sincronizando" });
    try {
      aplicarServidor(await chamarApi("GET", "/estado"));
      setConexao({ status: "ok" });
    } catch (e) {
      if (e instanceof ErroApi && e.status === 401) return expirou();
      setConexao({
        status: "erro",
        mensagem: `${(e as Error).message} Recarregue a página quando o servidor voltar.`,
      });
    }
  }, [aplicarServidor, expirou]);

  const sincronizou = useRef(false);
  useEffect(() => {
    // O StrictMode monta duas vezes em desenvolvimento; a carga inicial roda uma só.
    if (sincronizou.current) return;
    sincronizou.current = true;
    sincronizar();
  }, [sincronizar]);

  /** Mostra a mudança na hora e manda para o servidor; se ele recusar, volta ao que está salvo lá. */
  const commit = useCallback(
    (
      updater: (prev: StoreState) => StoreState,
      requisicao: {
        metodo: "PUT" | "POST" | "PATCH" | "DELETE";
        caminho: string;
        corpo?: unknown;
        sucesso: MensagemSucesso;
        /** Roda depois que o servidor confirmou (ex.: reler a sessão). */
        depois?: () => void;
      },
    ) => {
      setState(updater);
      fila.current = fila.current.then(async () => {
        try {
          const servidor = await chamarApi(requisicao.metodo, requisicao.caminho, requisicao.corpo);
          aplicarServidor(servidor);
          setConexao({ status: "ok" });
          // Só aqui, com a resposta do servidor: sucesso antes disso poderia ser mentira.
          notificar(typeof requisicao.sucesso === "function" ? requisicao.sucesso(servidor) : requisicao.sucesso);
          requisicao.depois?.();
        } catch (e) {
          // A mudança que estava na tela não foi gravada: a mensagem da entrada diz isso.
          if (e instanceof ErroApi && e.status === 401) return expirou(true);
          setConexao({ status: "erro", mensagem: `Não foi salvo: ${(e as Error).message}` });
          try {
            aplicarServidor(await chamarApi("GET", "/estado"));
          } catch {
            // servidor fora do ar: a tela fica com a mudança local e o aviso de erro
          }
        }
      });
    },
    [aplicarServidor, notificar, expirou],
  );

  const salvarCliente = useCallback(
    (cliente: Omit<Cliente, "id"> & { id?: string }) => {
      const record: Cliente = { ...cliente, id: cliente.id ?? uid() };
      commit(
        (prev) => ({
          ...prev,
          clientes: cliente.id
            ? prev.clientes.map((c) => (c.id === record.id ? record : c))
            : [...prev.clientes, record],
        }),
        {
          metodo: "PUT",
          caminho: `/clientes/${record.id}`,
          corpo: record,
          sucesso: cliente.id ? "Cliente atualizado com sucesso." : "Cliente cadastrado com sucesso.",
        },
      );
      return record;
    },
    [commit],
  );

  const excluirCliente = useCallback(
    (id: string) =>
      commit((prev) => ({ ...prev, clientes: prev.clientes.filter((c) => c.id !== id) }), {
        metodo: "DELETE",
        caminho: `/clientes/${id}`,
        sucesso: "Cliente excluído com sucesso.",
      }),
    [commit],
  );

  const salvarPedido = useCallback(
    (form: PedidoForm) => {
      const id = form.id ?? uid();
      // Número e estoque calculados aqui são só prévia: o servidor decide e a resposta substitui.
      commit((prev) => {
        const existente = form.id ? prev.pedidos.find((p) => p.id === form.id) : undefined;
        const numero =
          existente?.numero ?? prev.pedidos.reduce((max, p) => Math.max(max, p.numero), 0) + 1;
        const saved: Pedido = { ...form, id, numero };

        let estoque = prev.estoque;
        if (existente && consomeEstoque(existente)) estoque = movimentar(estoque, existente.pecas, 1);
        if (consomeEstoque(saved)) estoque = movimentar(estoque, saved.pecas, -1);

        const pedidos = existente
          ? prev.pedidos.map((p) => (p.id === saved.id ? saved : p))
          : [...prev.pedidos, saved];
        return { ...prev, pedidos, estoque };
      }, {
        metodo: "PUT",
        caminho: `/pedidos/${id}`,
        corpo: { ...form, id },
        sucesso: (servidor) => {
          const numero = servidor.pedidos.find((p) => p.id === id)?.numero;
          const rotulo = numero ? `Pedido #${numero}` : "Pedido";
          return form.id ? `${rotulo} atualizado com sucesso.` : `${rotulo} registrado com sucesso.`;
        },
      });
    },
    [commit],
  );

  const excluirPedido = useCallback(
    (id: string) => {
      commit((prev) => {
        const pedido = prev.pedidos.find((p) => p.id === id);
        if (!pedido) return prev;
        return {
          ...prev,
          pedidos: prev.pedidos.filter((p) => p.id !== id),
          estoque: consomeEstoque(pedido) ? movimentar(prev.estoque, pedido.pecas, 1) : prev.estoque,
        };
      }, { metodo: "DELETE", caminho: `/pedidos/${id}`, sucesso: "Pedido excluído com sucesso." });
    },
    [commit],
  );

  const entradaEstoque = useCallback(
    (item: Omit<ItemEstoque, "id">) => {
      const id = uid();
      commit((prev) => {
        const atual = prev.estoque.find((i) => chaveVariacao(i) === chaveVariacao(item));
        // Modelo novo entra na lista na hora; o servidor também o cadastra na tabela de modelos.
        const modelos = prev.modelos.some((m) => m.trim().toLowerCase() === item.modelo.trim().toLowerCase())
          ? prev.modelos
          : [...prev.modelos, item.modelo];
        if (!atual) {
          return { ...prev, modelos, estoque: [...prev.estoque, { ...item, id }] };
        }
        return {
          ...prev,
          modelos,
          estoque: prev.estoque.map((i) =>
            i.id === atual.id
              ? {
                  ...i,
                  quantidade: i.quantidade + item.quantidade,
                  minimo: item.minimo,
                  // Entrada sem preço mantém o que já estava cadastrado.
                  precoVenda: item.precoVenda > 0 ? item.precoVenda : i.precoVenda,
                }
              : i,
          ),
        };
      }, {
        metodo: "POST",
        caminho: "/estoque/entrada",
        corpo: { ...item, id },
        sucesso: "Entrada de estoque lançada com sucesso.",
      });
    },
    [commit],
  );

  const ajustarEstoque = useCallback(
    (ajustes: AjusteEstoque[]) =>
      commit(
        (prev) => ({
          ...prev,
          estoque: prev.estoque.map((i) => {
            const a = ajustes.find((x) => x.id === i.id);
            return a ? { ...i, ...a } : i;
          }),
        }),
        { metodo: "PATCH", caminho: "/estoque", corpo: { itens: ajustes }, sucesso: "Estoque atualizado com sucesso." },
      ),
    [commit],
  );

  const removerEstoque = useCallback(
    (ids: string[]) =>
      commit((prev) => ({ ...prev, estoque: prev.estoque.filter((i) => !ids.includes(i.id)) }), {
        metodo: "POST",
        caminho: "/estoque/remover",
        corpo: { ids },
        sucesso:
          ids.length === 1
            ? "Tamanho excluído do estoque com sucesso."
            : `${ids.length} tamanhos excluídos do estoque com sucesso.`,
      }),
    [commit],
  );

  const salvarVendedor = useCallback(
    (vendedor: Omit<Vendedor, "id"> & { id?: string }) => {
      const record: Vendedor = { ...vendedor, id: vendedor.id ?? uid() };
      commit(
        (prev) => ({
          ...prev,
          vendedores: vendedor.id
            ? prev.vendedores.map((v) => (v.id === record.id ? record : v))
            : [...prev.vendedores, record],
        }),
        {
          metodo: "PUT",
          caminho: `/vendedores/${record.id}`,
          corpo: record,
          sucesso: vendedor.id ? "Vendedor atualizado com sucesso." : "Vendedor cadastrado com sucesso.",
          // Mudou o próprio cadastro (papel, nome): relê a sessão e o que o papel novo pode ver.
          depois:
            record.id === usuario.id
              ? () => {
                  atualizarSessao();
                  sincronizar();
                }
              : undefined,
        },
      );
    },
    [commit, usuario.id, atualizarSessao, sincronizar],
  );

  const excluirVendedor = useCallback(
    (id: string) =>
      commit((prev) => ({ ...prev, vendedores: prev.vendedores.filter((v) => v.id !== id) }), {
        metodo: "DELETE",
        caminho: `/vendedores/${id}`,
        sucesso: "Vendedor excluído com sucesso.",
      }),
    [commit],
  );

  const api = useMemo<StoreApi>(() => {
    return {
      ...state,
      conexao,
      notificacoes,
      dispensarNotificacao,
      usuario,
      perm: permissoesDe(usuario.papel),
      nomeCliente: (id) => state.clientes.find((c) => c.id === id)?.nome ?? "Sem cliente",
      nomeVendedor: (id) =>
        state.vendedores.find((v) => v.id === id)?.nome ?? (id === usuario.id ? usuario.nome : "—"),
      salvarCliente,
      excluirCliente,
      salvarPedido,
      excluirPedido,
      entradaEstoque,
      ajustarEstoque,
      removerEstoque,
      salvarVendedor,
      excluirVendedor,
    };
  }, [
    state,
    conexao,
    notificacoes,
    dispensarNotificacao,
    usuario,
    salvarCliente,
    excluirCliente,
    salvarPedido,
    excluirPedido,
    entradaEstoque,
    ajustarEstoque,
    removerEstoque,
    salvarVendedor,
    excluirVendedor,
  ]);

  return <StoreContext.Provider value={api}>{children}</StoreContext.Provider>;
}

export function useStore() {
  const ctx = useContext(StoreContext);
  if (!ctx) throw new Error("useStore fora do StoreProvider");
  return ctx;
}
