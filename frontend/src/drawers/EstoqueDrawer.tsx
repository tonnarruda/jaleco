import { useState } from "react";
import { CORES, MINIMO_PADRAO, TAMANHOS, unicos } from "../catalog";
import { Drawer } from "../components/Drawer";
import { useConfirmacao } from "../components/Confirmacao";
import { brl, plural } from "../lib/format";
import { chaveVariacao } from "../lib/pedido";
import { useStore } from "../store";
import type { AjusteEstoque, ItemEstoque } from "../types";

const NOVO_MODELO = "__novo__";

const inteiro = (v: string) => Math.max(0, Math.floor(Number(v) || 0));

/** "" → 0 (sem preço); vírgula vale como separador decimal. NaN = inválido. */
const lerPreco = (v: string) => (v.trim() === "" ? 0 : Number(v.replace(",", ".")));

export function EntradaEstoqueDrawer({ onClose }: { onClose: () => void }) {
  const store = useStore();
  // Sem nenhum modelo cadastrado, a gaveta já abre no cadastro de modelo novo.
  const [modeloSel, setModeloSel] = useState<string>(store.modelos[0] ?? NOVO_MODELO);
  const [novoModelo, setNovoModelo] = useState("");
  const [cor, setCor] = useState("Branco");
  const [tamanho, setTamanho] = useState("M");
  const [quantidade, setQuantidade] = useState(1);
  const [minimo, setMinimo] = useState(MINIMO_PADRAO);
  const [preco, setPreco] = useState("");
  const [erro, setErro] = useState("");

  const modelos = unicos([...store.modelos, ...store.estoque.map((i) => i.modelo)]);
  // Nome digitado que já existe (outra caixa/espaços) usa a grafia cadastrada: não duplica o modelo.
  const modelo =
    modeloSel === NOVO_MODELO
      ? (modelos.find((m) => m.toLowerCase() === novoModelo.trim().toLowerCase()) ?? novoModelo.trim())
      : modeloSel;

  const existente = store.estoque.find(
    (i) => chaveVariacao(i) === chaveVariacao({ modelo, cor, tamanho }),
  );
  const cores = unicos([...CORES, ...store.estoque.map((i) => i.cor)]);

  const salvar = () => {
    if (!modelo) {
      setErro("Informe o nome do modelo novo.");
      return;
    }
    if (!cor.trim()) {
      setErro("Informe a cor.");
      return;
    }
    if (quantidade < 1) {
      setErro("A quantidade que entrou precisa ser pelo menos 1.");
      return;
    }
    const precoVenda = lerPreco(preco);
    if (!Number.isFinite(precoVenda) || precoVenda < 0) {
      setErro("Informe o preço de venda em reais, ou deixe em branco.");
      return;
    }
    store.entradaEstoque({ modelo, cor: cor.trim(), tamanho, quantidade, minimo, precoVenda });
    onClose();
  };

  return (
    <Drawer
      title="Entrada de estoque"
      onClose={onClose}
      footer={
        <button className="btn btn-pink" type="button" onClick={salvar}>
          Lançar entrada
        </button>
      }
    >
      <p className="help">
        Lance peças lisas que chegaram. Se a combinação já existe, a quantidade é somada.
      </p>
      <div className="fields">
        <div className="field">
          <label htmlFor="est-modelo">Modelo</label>
          <select
            id="est-modelo"
            autoFocus
            value={modeloSel}
            onChange={(e) => {
              setErro("");
              setModeloSel(e.target.value);
            }}
          >
            {modelos.map((m) => (
              <option key={m}>{m}</option>
            ))}
            <option value={NOVO_MODELO}>+ Cadastrar modelo novo</option>
          </select>
        </div>
        {modeloSel === NOVO_MODELO ? (
          <div className="field">
            <label htmlFor="est-novo-modelo">Nome do modelo novo</label>
            <input
              id="est-novo-modelo"
              autoFocus
              placeholder="Jaleco Feminino Slim"
              value={novoModelo}
              aria-invalid={erro.startsWith("Informe o nome do modelo")}
              onChange={(e) => {
                setErro("");
                setNovoModelo(e.target.value);
              }}
            />
          </div>
        ) : null}
        <div className="fields two">
          <div className="field">
            <label htmlFor="est-cor">Cor</label>
            <input id="est-cor" list="est-cores" value={cor} onChange={(e) => setCor(e.target.value)} />
            <datalist id="est-cores">
              {cores.map((c) => (
                <option key={c} value={c} />
              ))}
            </datalist>
          </div>
          <div className="field">
            <label htmlFor="est-tam">Tamanho</label>
            <select id="est-tam" value={tamanho} onChange={(e) => setTamanho(e.target.value)}>
              {TAMANHOS.map((t) => (
                <option key={t}>{t}</option>
              ))}
            </select>
          </div>
        </div>
        <div className="fields two">
          <div className="field">
            <label htmlFor="est-qtd">Quantidade que entrou (por tamanho)</label>
            <input
              id="est-qtd"
              type="number"
              min={1}
              value={quantidade}
              onChange={(e) => setQuantidade(inteiro(e.target.value))}
            />
          </div>
          <div className="field">
            <label htmlFor="est-min">Estoque mínimo (alerta)</label>
            <input
              id="est-min"
              type="number"
              min={0}
              value={minimo}
              onChange={(e) => setMinimo(inteiro(e.target.value))}
            />
          </div>
        </div>
        <div className="field">
          <label htmlFor="est-preco">Preço de venda (R$)</label>
          <input
            id="est-preco"
            type="number"
            inputMode="decimal"
            min={0}
            step="0.01"
            placeholder={existente?.precoVenda ? String(existente.precoVenda) : "Opcional"}
            value={preco}
            aria-invalid={erro.startsWith("Informe o preço")}
            onChange={(e) => setPreco(e.target.value)}
          />
        </div>
        <p className="hint">
          {existente?.precoVenda
            ? `Preço atual desta variação: ${brl(existente.precoVenda)}. Deixe em branco para manter.`
            : "Usado como preço da peça ao registrar um pedido."}
        </p>
        {erro ? <p className="erro">{erro}</p> : null}
      </div>
    </Drawer>
  );
}

/** Tamanhos de um produto na ordem da grade (PP → XG). */
function ordenar(itens: ItemEstoque[]): ItemEstoque[] {
  const pos = (t: string) => {
    const i = (TAMANHOS as readonly string[]).indexOf(t);
    return i < 0 ? TAMANHOS.length : i;
  };
  return [...itens].sort((a, b) => pos(a.tamanho) - pos(b.tamanho));
}

type Linha = { quantidade: string; minimo: string; preco: string };

export function EditarProdutoDrawer({ itens, onClose }: { itens: ItemEstoque[]; onClose: () => void }) {
  const store = useStore();
  const tamanhos = ordenar(itens);
  const [linhas, setLinhas] = useState<Record<string, Linha>>(() =>
    Object.fromEntries(
      tamanhos.map((i) => [
        i.id,
        { quantidade: String(i.quantidade), minimo: String(i.minimo), preco: i.precoVenda ? String(i.precoVenda) : "" },
      ]),
    ),
  );
  const [erro, setErro] = useState("");

  const set = (id: string, campo: keyof Linha, valor: string) => {
    setErro("");
    setLinhas((l) => ({ ...l, [id]: { ...l[id], [campo]: valor } }));
  };

  const salvar = () => {
    const ajustes: AjusteEstoque[] = [];
    for (const item of tamanhos) {
      const l = linhas[item.id];
      const quantidade = Number(l.quantidade);
      const minimo = Number(l.minimo);
      const precoVenda = lerPreco(l.preco);
      if (!Number.isInteger(quantidade) || quantidade < 0 || !Number.isInteger(minimo) || minimo < 0) {
        setErro(`Tamanho ${item.tamanho}: quantidade e mínimo precisam ser números inteiros, a partir de 0.`);
        return;
      }
      if (!Number.isFinite(precoVenda) || precoVenda < 0) {
        setErro(`Tamanho ${item.tamanho}: informe o preço em reais, ou deixe em branco.`);
        return;
      }
      if (quantidade !== item.quantidade || minimo !== item.minimo || precoVenda !== item.precoVenda) {
        ajustes.push({ id: item.id, quantidade, minimo, precoVenda });
      }
    }
    if (ajustes.length) store.ajustarEstoque(ajustes);
    onClose();
  };

  const { modelo, cor } = tamanhos[0];
  return (
    <Drawer
      title="Editar produto"
      onClose={onClose}
      footer={
        <button className="btn btn-pink" type="button" onClick={salvar}>
          Salvar alterações
        </button>
      }
    >
      <p className="help">
        <strong className="text">{modelo}</strong> · {cor}. Corrija a contagem (peça com defeito,
        inventário), o alerta e o preço de cada tamanho. Entradas novas vão em Entrada de estoque.
      </p>
      <div className="grade-tamanhos" role="table" aria-label={`Tamanhos de ${modelo} ${cor}`}>
        <div className="grade-cab" role="row">
          <span role="columnheader">Tam.</span>
          <span role="columnheader">Quantidade</span>
          <span role="columnheader">Mínimo</span>
          <span role="columnheader">Preço (R$)</span>
        </div>
        {tamanhos.map((item) => (
          <div className="grade-linha field" role="row" key={item.id}>
            <span className="grade-tam" role="rowheader">
              {item.tamanho}
            </span>
            <input
              type="number"
              min={0}
              aria-label={`Quantidade ${item.tamanho}`}
              value={linhas[item.id].quantidade}
              onChange={(e) => set(item.id, "quantidade", e.target.value)}
            />
            <input
              type="number"
              min={0}
              aria-label={`Mínimo ${item.tamanho}`}
              value={linhas[item.id].minimo}
              onChange={(e) => set(item.id, "minimo", e.target.value)}
            />
            <input
              type="number"
              inputMode="decimal"
              min={0}
              step="0.01"
              placeholder="Sem preço"
              aria-label={`Preço ${item.tamanho}`}
              value={linhas[item.id].preco}
              onChange={(e) => set(item.id, "preco", e.target.value)}
            />
          </div>
        ))}
      </div>
      <p className="hint">Mudar o preço não altera pedidos já registrados.</p>
      {erro ? <p className="erro">{erro}</p> : null}
    </Drawer>
  );
}

export function ExcluirProdutoDrawer({ itens, onClose }: { itens: ItemEstoque[]; onClose: () => void }) {
  const store = useStore();
  const confirmar = useConfirmacao();
  const tamanhos = ordenar(itens);
  const [marcados, setMarcados] = useState<Set<string>>(new Set());

  // Tamanho que está em qualquer pedido (inclusive entregue ou cancelado) não sai: o
  // histórico de vendas aponta para ele. O servidor confere de novo ao excluir.
  const pedidosDo = (item: ItemEstoque) =>
    store.pedidos
      .filter((p) => p.pecas.some((x) => chaveVariacao(x) === chaveVariacao(item)))
      .map((p) => p.numero)
      .sort((a, b) => a - b);
  const livres = tamanhos.filter((i) => pedidosDo(i).length === 0);

  const alternar = (id: string) =>
    setMarcados((m) => {
      const n = new Set(m);
      if (n.has(id)) n.delete(id);
      else n.add(id);
      return n;
    });
  const todos = livres.length > 0 && livres.every((i) => marcados.has(i.id));

  const selecionados = livres.filter((i) => marcados.has(i.id));

  const excluir = async () => {
    const ok = await confirmar({
      titulo: "Excluir do estoque",
      mensagem: (
        <>
          Excluir {selecionados.length === 1 ? "o tamanho" : "os tamanhos"}{" "}
          <strong>{selecionados.map((i) => i.tamanho).join(", ")}</strong> de {modelo} · {cor}? Saem{" "}
          {plural(
            selecionados.reduce((acc, i) => acc + i.quantidade, 0),
            "peça",
            "peças",
          )}{" "}
          do estoque. Essa ação não pode ser desfeita.
        </>
      ),
    });
    if (ok) {
      store.removerEstoque(selecionados.map((i) => i.id));
      onClose();
    }
  };

  const { modelo, cor } = tamanhos[0];
  return (
    <Drawer
      title="Excluir do estoque"
      onClose={onClose}
      footer={
        <button className="btn btn-danger" type="button" disabled={!selecionados.length} onClick={excluir}>
          {selecionados.length
            ? `Excluir ${plural(selecionados.length, "tamanho", "tamanhos")}`
            : livres.length
              ? "Marque os tamanhos"
              : "Nenhum tamanho pode ser excluído"}
        </button>
      }
    >
      <p className="help">
        <strong className="text">{modelo}</strong> · {cor}. Marque os tamanhos que saem do estoque.
        Tamanho que está em algum pedido não pode ser excluído; para tirar de circulação, zere a
        quantidade em Editar.
      </p>
      <div className="check-lista">
        {livres.length ? (
          <label className="check-todos">
            <input
              type="checkbox"
              checked={todos}
              onChange={() => setMarcados(todos ? new Set() : new Set(livres.map((i) => i.id)))}
            />
            {livres.length === tamanhos.length ? "Selecionar todos" : "Selecionar todos os que podem sair"}
          </label>
        ) : null}
        {tamanhos.map((item) => {
          const pedidos = pedidosDo(item);
          const travado = pedidos.length > 0;
          return (
            <label key={item.id} className={`check-item${travado ? " travado" : ""}`}>
              <input
                type="checkbox"
                disabled={travado}
                checked={!travado && marcados.has(item.id)}
                onChange={() => alternar(item.id)}
              />
              <span className="strong">{item.tamanho}</span>
              <span className="muted">
                {travado
                  ? `Em ${plural(pedidos.length, "pedido", "pedidos")} (#${pedidos.join(", #")}): não pode excluir`
                  : `${plural(item.quantidade, "peça", "peças")}${item.precoVenda ? ` · ${brl(item.precoVenda)}` : ""}`}
              </span>
            </label>
          );
        })}
      </div>
    </Drawer>
  );
}
