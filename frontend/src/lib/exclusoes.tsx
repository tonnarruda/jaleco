import { useConfirmacao } from "../components/Confirmacao";
import { useStore } from "../store";
import type { Cliente, Pedido, Vendedor } from "../types";
import { plural } from "./format";
import { consomeEstoque } from "./pedido";

/** "#2, #5" a partir dos pedidos, em ordem. */
const numeros = (pedidos: Pedido[]) =>
  pedidos
    .map((p) => p.numero)
    .sort((a, b) => a - b)
    .map((n) => `#${n}`)
    .join(", ");

/**
 * Exclusões disparadas pelo ícone de lixeira das listas. Cada uma confere o que
 * bloqueia (o servidor confere de novo) e pede confirmação antes de apagar.
 */
export function useExclusoes() {
  const store = useStore();
  const confirmar = useConfirmacao();

  const cliente = async (c: Cliente) => {
    const pedidos = store.pedidos.filter((p) => p.clienteId === c.id);
    if (pedidos.length) {
      await confirmar({
        somenteAviso: true,
        titulo: "Não é possível excluir",
        mensagem: (
          <>
            <strong>{c.nome}</strong> está em {plural(pedidos.length, "pedido", "pedidos")} (
            {numeros(pedidos)}). Exclua os pedidos antes.
          </>
        ),
      });
      return;
    }
    const ok = await confirmar({
      titulo: "Excluir cliente",
      mensagem: (
        <>
          Excluir <strong>{c.nome}</strong>? Essa ação não pode ser desfeita.
        </>
      ),
    });
    if (ok) store.excluirCliente(c.id);
  };

  const pedido = async (p: Pedido) => {
    const ok = await confirmar({
      titulo: "Excluir pedido",
      mensagem: (
        <>
          Excluir o pedido <strong>#{p.numero}</strong> de {store.nomeCliente(p.clienteId)}?{" "}
          {consomeEstoque(p) ? "As peças voltam ao estoque. " : ""}Essa ação não pode ser desfeita.
        </>
      ),
    });
    if (ok) store.excluirPedido(p.id);
  };

  const vendedor = async (v: Vendedor) => {
    const pedidos = store.pedidos.filter((p) => p.vendedorId === v.id);
    if (pedidos.length) {
      await confirmar({
        somenteAviso: true,
        titulo: "Não é possível excluir",
        mensagem: (
          <>
            <strong>{v.nome}</strong> está em {plural(pedidos.length, "pedido", "pedidos")} (
            {numeros(pedidos)}). Para tirar o acesso, edite e marque como Inativo.
          </>
        ),
      });
      return;
    }
    const ok = await confirmar({
      titulo: "Excluir vendedor",
      mensagem: (
        <>
          Excluir <strong>{v.nome}</strong>? As metas cadastradas saem junto. Essa ação não pode ser
          desfeita.
        </>
      ),
    });
    if (ok) store.excluirVendedor(v.id);
  };

  return { cliente, pedido, vendedor };
}
