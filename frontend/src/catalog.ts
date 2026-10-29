import type { FormaPagamento, PapelUsuario, SituacaoPagamento } from "./types";

// Os modelos vêm do banco (tabela modelos): os padrão são cadastrados pelo backend ao subir.

export const CORES = ["Branco", "Rosa", "Preto", "Azul marinho", "Verde"] as const;
export const TAMANHOS = ["PP", "P", "M", "G", "GG", "XG"] as const;
export const CORES_LINHA = ["Rosa", "Branco", "Preto", "Azul", "Dourado", "Prata"] as const;
export const ADICIONAIS = ["Nenhum", "Logo", "Nome extra", "Bolso extra", "Manga bordada"] as const;
export const CANAIS = ["Instagram", "WhatsApp", "Indicação", "Site", "Presencial"] as const;
export const COMO_CONHECEU = ["Instagram", "Indicação", "Google", "Evento", "Outro"] as const;
export const FORMAS_PAGAMENTO: FormaPagamento[] = [
  "Pix",
  "Cartão de crédito",
  "Cartão de débito",
  "Dinheiro",
  "Transferência",
];
export const SITUACOES_PAGAMENTO: SituacaoPagamento[] = ["Pendente", "Parcial", "Pago"];
export const PAPEIS: PapelUsuario[] = ["Vendedor(a)", "Gerente", "Administrador"];

export const ETAPAS = [
  "Orçamento",
  "Aguardando arte",
  "Arte aprovada",
  "Em produção",
  "Pronto",
  "Enviado",
  "Entregue",
  "Cancelado",
] as const;

export const ETAPAS_ABERTAS = [
  "Orçamento",
  "Aguardando arte",
  "Arte aprovada",
  "Em produção",
  "Pronto",
  "Enviado",
] as const;

export const FILTROS_PEDIDO = [
  "Em aberto",
  "Atrasados",
  "A receber",
  ...ETAPAS,
  "Todos",
] as const;

export const MINIMO_PADRAO = 3;
export const PRAZO_PADRAO_DIAS = 10;

/** "Jaleco Feminino Acinturado" → "Feminino Acinturado", como nos filtros do estoque. */
export function rotuloModelo(modelo: string): string {
  return modelo.replace(/^Jaleco\s+/i, "");
}

export function unicos(lista: string[]): string[] {
  const vistos = new Set<string>();
  return lista.filter((v) => {
    const k = v.trim().toLowerCase();
    if (!k || vistos.has(k)) return false;
    vistos.add(k);
    return true;
  });
}
