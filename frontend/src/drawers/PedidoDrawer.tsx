import { useState } from "react";
import {
  ADICIONAIS,
  CANAIS,
  CORES,
  CORES_LINHA,
  ETAPAS,
  FORMAS_PAGAMENTO,
  SITUACOES_PAGAMENTO,
  TAMANHOS,
  unicos,
} from "../catalog";
import { Drawer } from "../components/Drawer";
import { brl, maskWhatsapp } from "../lib/format";
import { chaveVariacao, consomeEstoque, novaPeca, totaisPedido } from "../lib/pedido";
import { useStore } from "../store";
import type { PecaPedido, PedidoForm } from "../types";
import { clienteEmBranco } from "./ClienteDrawer";

const NOVO_CLIENTE = "__novo__";

type Props = {
  value: PedidoForm;
  onClose: () => void;
};

const num = (v: string) => {
  const n = Number(v.replace(",", "."));
  return Number.isFinite(n) && n >= 0 ? n : 0;
};

export function PedidoDrawer({ value, onClose }: Props) {
  const store = useStore();
  const precoCadastrado = (peca: PecaPedido) =>
    store.estoque.find((i) => chaveVariacao(i) === chaveVariacao(peca))?.precoVenda ?? 0;

  /** Peça com o preço de venda cadastrado da variação; sem cadastro, 0 (nunca um valor herdado). */
  const comPrecoDoEstoque = (peca: PecaPedido): PecaPedido => ({
    ...peca,
    precoUnitario: precoCadastrado(peca),
  });
  // Pedido novo já nasce com o preço do estoque; pedido existente mantém o preço combinado.
  const [form, setForm] = useState<PedidoForm>(() =>
    value.id ? value : { ...value, pecas: value.pecas.map(comPrecoDoEstoque) },
  );
  const [novoCliente, setNovoCliente] = useState({ nome: "", whatsapp: "" });
  const [erro, setErro] = useState("");
  const editando = Boolean(form.id);
  const original = store.pedidos.find((p) => p.id === form.id);

  const set = <K extends keyof PedidoForm>(k: K, v: PedidoForm[K]) =>
    setForm((f) => ({ ...f, [k]: v }));

  const setPeca = (id: string, patch: Partial<PecaPedido>) => {
    // Trocar modelo, cor ou tamanho puxa o preço daquela variação.
    const mudouVariacao = "modelo" in patch || "cor" in patch || "tamanho" in patch;
    setForm((f) => ({
      ...f,
      pecas: f.pecas.map((p) => {
        if (p.id !== id) return p;
        const nova = { ...p, ...patch };
        return mudouVariacao ? comPrecoDoEstoque(nova) : nova;
      }),
    }));
  };

  const totais = totaisPedido(form);
  const modelos = unicos([...store.modelos, ...store.estoque.map((i) => i.modelo)]);
  const cores = unicos([...CORES, ...store.estoque.map((i) => i.cor)]);

  /** Quanto existe da variação, contando de volta o que este pedido já tinha separado. */
  const avisoEstoque = (peca: PecaPedido) => {
    const item = store.estoque.find((i) => chaveVariacao(i) === chaveVariacao(peca));
    if (!item) return { cls: "", texto: "Sem cadastro no estoque para esta combinação." };
    const reservado =
      original && consomeEstoque(original)
        ? original.pecas
            .filter((p) => chaveVariacao(p) === chaveVariacao(peca))
            .reduce((acc, p) => acc + p.quantidade, 0)
        : 0;
    const disponivel = item.quantidade + reservado;
    if (peca.quantidade > disponivel) {
      return { cls: "warn", texto: `Só ${disponivel} em estoque para esta combinação.` };
    }
    return { cls: "", texto: `${disponivel} em estoque.` };
  };

  const salvar = () => {
    // Antes de criar cliente novo: pedido recusado não pode deixar cliente gravado pela metade.
    if (form.pecas.some((p) => !(p.precoUnitario > 0))) {
      setErro("Há peça sem preço. Cadastre o preço de venda no estoque ou informe o preço na peça.");
      return;
    }
    let clienteId = form.clienteId;
    if (clienteId === NOVO_CLIENTE) {
      if (!novoCliente.nome.trim()) {
        setErro("Informe o nome do cliente novo.");
        return;
      }
      clienteId = store.salvarCliente({
        ...clienteEmBranco,
        nome: novoCliente.nome.trim(),
        whatsapp: novoCliente.whatsapp,
      }).id;
    }
    if (!clienteId) {
      setErro("Selecione o cliente.");
      return;
    }
    if (form.pecas.length === 0) {
      setErro("Adicione pelo menos uma peça.");
      return;
    }
    store.salvarPedido({ ...form, clienteId });
    onClose();
  };

  const marcarPago = () =>
    setForm((f) => ({ ...f, situacaoPagamento: "Pago", valorPago: totaisPedido(f).total }));

  const aplicarPix = () =>
    setForm((f) => ({
      ...f,
      formaPagamento: "Pix",
      desconto: Math.round(totaisPedido(f).pecas * 0.05 * 100) / 100,
    }));

  return (
    <Drawer
      title={editando ? `Pedido #${form.numero}` : "Novo pedido"}
      onClose={onClose}
      footer={
        <>
          {erro ? <p className="erro foot-erro">{erro}</p> : null}
          <button className="btn btn-pink" type="button" onClick={salvar}>
            {editando ? "Salvar pedido" : "Registrar pedido"}
          </button>
        </>
      }
    >
      <fieldset className="section">
        <legend>CLIENTE</legend>
        <div className="fields">
          <div className="field">
            <label htmlFor="ped-cliente">Cliente</label>
            <select
              id="ped-cliente"
              autoFocus
              value={form.clienteId}
              aria-invalid={Boolean(erro) && !form.clienteId}
              onChange={(e) => {
                setErro("");
                set("clienteId", e.target.value);
              }}
            >
              <option value="">Selecione...</option>
              {[...store.clientes]
                .sort((a, b) => a.nome.localeCompare(b.nome))
                .map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.nome}
                  </option>
                ))}
              <option value={NOVO_CLIENTE}>+ Cadastrar cliente novo</option>
            </select>
          </div>
          {form.clienteId === NOVO_CLIENTE ? (
            <div className="fields two">
              <div className="field">
                <label htmlFor="ped-novo-nome">Nome do cliente novo</label>
                <input
                  id="ped-novo-nome"
                  autoFocus
                  value={novoCliente.nome}
                  onChange={(e) => setNovoCliente((n) => ({ ...n, nome: e.target.value }))}
                />
              </div>
              <div className="field">
                <label htmlFor="ped-novo-whats">WhatsApp</label>
                <input
                  id="ped-novo-whats"
                  inputMode="tel"
                  placeholder="(00) 00000-0000"
                  value={novoCliente.whatsapp}
                  onChange={(e) =>
                    setNovoCliente((n) => ({ ...n, whatsapp: maskWhatsapp(e.target.value) }))
                  }
                />
              </div>
            </div>
          ) : null}
          <div className="fields two">
            <div className="field">
              <label htmlFor="ped-canal">Canal da venda</label>
              <select id="ped-canal" value={form.canal} onChange={(e) => set("canal", e.target.value)}>
                {CANAIS.map((c) => (
                  <option key={c}>{c}</option>
                ))}
              </select>
            </div>
            <div className="field">
              <label htmlFor="ped-data">Data do pedido</label>
              <input
                id="ped-data"
                type="date"
                value={form.dataPedido}
                onChange={(e) => set("dataPedido", e.target.value)}
              />
            </div>
          </div>
        </div>
      </fieldset>

      <fieldset className="section">
        <legend>PEÇAS E BORDADO</legend>
        {form.pecas.map((peca, i) => {
          const aviso = avisoEstoque(peca);
          const pid = `peca-${peca.id}`;
          return (
            <div className="peca" key={peca.id}>
              <div className="peca-head">
                <strong>Peça {i + 1}</strong>
                {form.pecas.length > 1 ? (
                  <button
                    className="link-muted"
                    type="button"
                    onClick={() =>
                      setForm((f) => ({ ...f, pecas: f.pecas.filter((p) => p.id !== peca.id) }))
                    }
                  >
                    remover
                  </button>
                ) : null}
              </div>
              <div className="fields">
                <div className="field">
                  <label htmlFor={`${pid}-modelo`}>Modelo</label>
                  <select
                    id={`${pid}-modelo`}
                    value={peca.modelo}
                    onChange={(e) => setPeca(peca.id, { modelo: e.target.value })}
                  >
                    {unicos([...modelos, peca.modelo]).map((m) => (
                      <option key={m}>{m}</option>
                    ))}
                  </select>
                </div>
                <div className="fields three-even">
                  <div className="field">
                    <label htmlFor={`${pid}-cor`}>Cor</label>
                    <select
                      id={`${pid}-cor`}
                      value={peca.cor}
                      onChange={(e) => setPeca(peca.id, { cor: e.target.value })}
                    >
                      {unicos([...cores, peca.cor]).map((c) => (
                        <option key={c}>{c}</option>
                      ))}
                    </select>
                  </div>
                  <div className="field">
                    <label htmlFor={`${pid}-tam`}>Tamanho</label>
                    <select
                      id={`${pid}-tam`}
                      value={peca.tamanho}
                      onChange={(e) => setPeca(peca.id, { tamanho: e.target.value })}
                    >
                      {TAMANHOS.map((t) => (
                        <option key={t}>{t}</option>
                      ))}
                    </select>
                  </div>
                  <div className="field">
                    <label htmlFor={`${pid}-qtd`}>Qtd.</label>
                    <input
                      id={`${pid}-qtd`}
                      type="number"
                      min={1}
                      value={peca.quantidade}
                      onChange={(e) =>
                        setPeca(peca.id, { quantidade: Math.max(1, Math.floor(num(e.target.value))) })
                      }
                    />
                  </div>
                </div>
                <p className={`hint ${aviso.cls}`}>{aviso.texto}</p>
                {!(peca.precoUnitario > 0) ? (
                  <p className="hint warn">
                    {precoCadastrado(peca)
                      ? "Informe o preço desta peça."
                      : "Sem preço de venda cadastrado no estoque para esta combinação. Informe o preço abaixo."}
                  </p>
                ) : null}
                <div className="fields two">
                  <div className="field">
                    <label htmlFor={`${pid}-nome`}>Nome no bordado</label>
                    <input
                      id={`${pid}-nome`}
                      placeholder="Dra. Ana Lima"
                      value={peca.nomeBordado}
                      onChange={(e) => setPeca(peca.id, { nomeBordado: e.target.value })}
                    />
                  </div>
                  <div className="field">
                    <label htmlFor={`${pid}-prof`}>Profissão no bordado</label>
                    <input
                      id={`${pid}-prof`}
                      placeholder="Odontologia"
                      value={peca.profissaoBordado}
                      onChange={(e) => setPeca(peca.id, { profissaoBordado: e.target.value })}
                    />
                  </div>
                </div>
                <div className="fields three-even">
                  <div className="field">
                    <label htmlFor={`${pid}-linha`}>Cor da linha</label>
                    <select
                      id={`${pid}-linha`}
                      value={peca.corLinha}
                      onChange={(e) => setPeca(peca.id, { corLinha: e.target.value })}
                    >
                      {CORES_LINHA.map((c) => (
                        <option key={c}>{c}</option>
                      ))}
                    </select>
                  </div>
                  <div className="field">
                    <label htmlFor={`${pid}-adic`}>Adicional</label>
                    <select
                      id={`${pid}-adic`}
                      value={peca.adicional}
                      onChange={(e) => setPeca(peca.id, { adicional: e.target.value })}
                    >
                      {ADICIONAIS.map((c) => (
                        <option key={c}>{c}</option>
                      ))}
                    </select>
                  </div>
                  <div className="field">
                    <label htmlFor={`${pid}-preco`}>Preço un. (R$)</label>
                    <input
                      id={`${pid}-preco`}
                      type="number"
                      min={0}
                      step="0.01"
                      value={peca.precoUnitario}
                      onChange={(e) => setPeca(peca.id, { precoUnitario: num(e.target.value) })}
                    />
                  </div>
                </div>
              </div>
            </div>
          );
        })}
        <button
          className="btn btn-outline btn-sm"
          type="button"
          onClick={() => setForm((f) => ({ ...f, pecas: [...f.pecas, comPrecoDoEstoque(novaPeca(modelos[0] ?? ""))] }))}
        >
          + Adicionar peça
        </button>
      </fieldset>

      <fieldset className="section">
        <legend>ANDAMENTO</legend>
        <div className="fields">
          <div className="fields two">
            <div className="field">
              <label htmlFor="ped-etapa">Etapa</label>
              <select
                id="ped-etapa"
                value={form.etapa}
                onChange={(e) => set("etapa", e.target.value as PedidoForm["etapa"])}
              >
                {ETAPAS.map((e) => (
                  <option key={e}>{e}</option>
                ))}
              </select>
            </div>
            <div className="field">
              <label htmlFor="ped-prazo">Prazo de entrega</label>
              <input
                id="ped-prazo"
                type="date"
                value={form.prazoEntrega}
                onChange={(e) => set("prazoEntrega", e.target.value)}
              />
            </div>
          </div>
          <div className="field">
            <label htmlFor="ped-rastreio">Código de rastreio</label>
            <input
              id="ped-rastreio"
              value={form.codigoRastreio}
              onChange={(e) => set("codigoRastreio", e.target.value)}
            />
          </div>
          <div className="field">
            <label htmlFor="ped-obs">Observações</label>
            <textarea
              id="ped-obs"
              rows={2}
              placeholder="Arte aprovada em..., detalhes do logo, endereço diferente..."
              value={form.observacoes}
              onChange={(e) => set("observacoes", e.target.value)}
            />
          </div>
        </div>
      </fieldset>

      <fieldset className="section">
        <legend>PAGAMENTO</legend>
        <div className="fields">
          <div className="fields two">
            <div className="field">
              <label htmlFor="ped-forma">Forma</label>
              <select
                id="ped-forma"
                value={form.formaPagamento}
                onChange={(e) => set("formaPagamento", e.target.value as PedidoForm["formaPagamento"])}
              >
                {FORMAS_PAGAMENTO.map((f) => (
                  <option key={f}>{f}</option>
                ))}
              </select>
            </div>
            <div className="field">
              <label htmlFor="ped-situacao">Situação</label>
              <select
                id="ped-situacao"
                value={form.situacaoPagamento}
                onChange={(e) =>
                  set("situacaoPagamento", e.target.value as PedidoForm["situacaoPagamento"])
                }
              >
                {SITUACOES_PAGAMENTO.map((s) => (
                  <option key={s}>{s}</option>
                ))}
              </select>
            </div>
          </div>
          <div className="fields three-even">
            <div className="field">
              <label htmlFor="ped-desc">Desconto (R$)</label>
              <input
                id="ped-desc"
                type="number"
                min={0}
                step="0.01"
                value={form.desconto}
                onChange={(e) => set("desconto", num(e.target.value))}
              />
            </div>
            <div className="field">
              <label htmlFor="ped-frete">Frete (R$)</label>
              <input
                id="ped-frete"
                type="number"
                min={0}
                step="0.01"
                value={form.frete}
                onChange={(e) => set("frete", num(e.target.value))}
              />
            </div>
            <div className="field">
              <label htmlFor="ped-pago">Valor já pago (R$)</label>
              <input
                id="ped-pago"
                type="number"
                min={0}
                step="0.01"
                value={form.valorPago}
                onChange={(e) => set("valorPago", num(e.target.value))}
              />
            </div>
          </div>
          <div className="inline-actions">
            <button className="btn btn-outline btn-sm" type="button" onClick={aplicarPix}>
              Aplicar 5% Pix
            </button>
            <button className="btn btn-outline btn-sm" type="button" onClick={marcarPago}>
              Marcar como pago
            </button>
          </div>
          <div className="totals">
            <div>
              <span>Peças</span>
              <span>{brl(totais.pecas)}</span>
            </div>
            <div>
              <span>Desconto</span>
              <span>− {brl(totais.desconto)}</span>
            </div>
            <div>
              <span>Frete</span>
              <span>{brl(totais.frete)}</span>
            </div>
            <div className="total">
              <span>Total</span>
              <span>{brl(totais.total)}</span>
            </div>
            <div className={totais.falta > 0 ? "falta" : "quitado"}>
              <span>{totais.falta > 0 ? "Falta receber" : "Quitado"}</span>
              <span>{brl(totais.falta)}</span>
            </div>
          </div>
        </div>
      </fieldset>
    </Drawer>
  );
}
