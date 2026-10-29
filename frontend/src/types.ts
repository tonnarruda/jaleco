/** Papel de quem está na equipe; o Administrador também é uma pessoa cadastrada. */
export type PapelUsuario = "Administrador" | "Gerente" | "Vendedor(a)";
export type AcessoVendedor = "Ativo" | "Inativo";

export type EtapaPedido =
  | "Orçamento"
  | "Aguardando arte"
  | "Arte aprovada"
  | "Em produção"
  | "Pronto"
  | "Enviado"
  | "Entregue"
  | "Cancelado";

export type FiltroPedido =
  | "Em aberto"
  | "Atrasados"
  | "A receber"
  | EtapaPedido
  | "Todos";

export type FormaPagamento = "Pix" | "Cartão de crédito" | "Cartão de débito" | "Dinheiro" | "Transferência";
export type SituacaoPagamento = "Pendente" | "Parcial" | "Pago";

export type Cliente = {
  id: string;
  nome: string;
  whatsapp: string;
  email: string;
  profissao: string;
  faculdade: string;
  instagram: string;
  cidade: string;
  aniversario: string;
  tamanhoHabitual: string;
  comoConheceu: string;
  endereco: string;
  observacoes: string;
};

export type PecaPedido = {
  id: string;
  modelo: string;
  cor: string;
  tamanho: string;
  quantidade: number;
  nomeBordado: string;
  profissaoBordado: string;
  corLinha: string;
  adicional: string;
  precoUnitario: number;
};

export type Pedido = {
  id: string;
  numero: number;
  clienteId: string;
  vendedorId: string;
  canal: string;
  dataPedido: string;
  pecas: PecaPedido[];
  etapa: EtapaPedido;
  prazoEntrega: string;
  codigoRastreio: string;
  observacoes: string;
  formaPagamento: FormaPagamento;
  situacaoPagamento: SituacaoPagamento;
  desconto: number;
  frete: number;
  valorPago: number;
};

export type PedidoForm = Omit<Pedido, "id" | "numero"> & { id?: string; numero?: number };

export type ItemEstoque = {
  id: string;
  modelo: string;
  cor: string;
  tamanho: string;
  quantidade: number;
  minimo: number;
  /** Preço sugerido no pedido. 0 = sem preço cadastrado. */
  precoVenda: number;
};

/** Meta mensal em R$ que vale a partir de `desde` (AAAA-MM) até a próxima mudança. */
/** Correção de uma variação do estoque: contagem, alerta e preço. */
export type AjusteEstoque = Pick<ItemEstoque, "id" | "quantidade" | "minimo" | "precoVenda">;

export type MetaVendedor = {
  desde: string;
  valor: number;
};

export type Vendedor = {
  id: string;
  nome: string;
  whatsapp: string;
  papel: PapelUsuario;
  acesso: AcessoVendedor;
  /** Percentual sobre a venda, ex.: 5 = 5%. */
  comissao: number;
  /** Histórico de metas; nunca é sobrescrito, só ganha novas vigências. */
  metas: MetaVendedor[];
  /** Login da pessoa. Vazio = ainda sem acesso ao sistema. */
  email: string;
  /** Só o servidor preenche: a pessoa já criou a senha. */
  possuiLogin?: boolean;
};

export type Usuario = {
  id: string;
  nome: string;
  papel: PapelUsuario;
  email: string;
};

export type StoreState = {
  clientes: Cliente[];
  pedidos: Pedido[];
  estoque: ItemEstoque[];
  vendedores: Vendedor[];
  /** Modelos cadastrados no banco, na ordem de cadastro (os padrão primeiro). */
  modelos: string[];
};
