import { ETAPAS_ABERTAS, PRAZO_PADRAO_DIAS } from "../catalog";
import type { ItemEstoque, PecaPedido, Pedido, PedidoForm } from "../types";
import { hojeISO, somarDiasISO, uid } from "./format";

type ComPecas = Pick<Pedido, "pecas" | "desconto" | "frete" | "valorPago">;

export function subtotalPecas(pedido: Pick<Pedido, "pecas">): number {
  return pedido.pecas.reduce((acc, p) => acc + p.precoUnitario * p.quantidade, 0);
}

export function qtdPecas(pedido: Pick<Pedido, "pecas">): number {
  return pedido.pecas.reduce((acc, p) => acc + p.quantidade, 0);
}

export function totaisPedido(pedido: ComPecas) {
  const pecas = subtotalPecas(pedido);
  const total = Math.max(0, pecas - pedido.desconto + pedido.frete);
  const falta = Math.max(0, total - pedido.valorPago);
  return { pecas, desconto: pedido.desconto, frete: pedido.frete, total, falta };
}

export function pedidoAberto(pedido: Pedido): boolean {
  return (ETAPAS_ABERTAS as readonly string[]).includes(pedido.etapa);
}

export function pedidoAtrasado(pedido: Pedido, hoje = hojeISO()): boolean {
  return (
    pedidoAberto(pedido) &&
    pedido.etapa !== "Enviado" &&
    Boolean(pedido.prazoEntrega) &&
    pedido.prazoEntrega < hoje
  );
}

/** Orçamento ainda não é venda: não entra no faturamento nem no "a receber". */
export function pedidoConta(pedido: Pick<Pedido, "etapa">): boolean {
  return pedido.etapa !== "Cancelado" && pedido.etapa !== "Orçamento";
}

/** Só pedido que virou venda segura peça no estoque. */
export const consomeEstoque = pedidoConta;

export function pedidoAReceber(pedido: Pedido): boolean {
  return pedidoConta(pedido) && totaisPedido(pedido).falta > 0;
}

export function chaveVariacao(v: Pick<ItemEstoque, "modelo" | "cor" | "tamanho">): string {
  return [v.modelo, v.cor, v.tamanho].map((s) => s.trim().toLowerCase()).join("|");
}

/** `modelo` é o primeiro da lista cadastrada (store.modelos). */
export function novaPeca(modelo: string): PecaPedido {
  return {
    id: uid(),
    modelo,
    cor: "Branco",
    tamanho: "M",
    quantidade: 1,
    nomeBordado: "",
    profissaoBordado: "",
    corLinha: "Rosa",
    adicional: "Nenhum",
    // O preço vem do estoque (PedidoDrawer); sem cadastro, fica 0 até alguém informar.
    precoUnitario: 0,
  };
}

export function pedidoEmBranco(vendedorId: string, modelo: string): PedidoForm {
  return {
    clienteId: "",
    vendedorId,
    canal: "Instagram",
    dataPedido: hojeISO(),
    pecas: [novaPeca(modelo)],
    etapa: "Aguardando arte",
    prazoEntrega: somarDiasISO(PRAZO_PADRAO_DIAS),
    codigoRastreio: "",
    observacoes: "",
    formaPagamento: "Pix",
    situacaoPagamento: "Pendente",
    desconto: 0,
    frete: 0,
    valorPago: 0,
  };
}
