import { useMemo, useState } from "react";
import { TAMANHOS, rotuloModelo, unicos } from "../catalog";
import { AcoesLinha } from "../components/AcoesLinha";
import { EditarProdutoDrawer, EntradaEstoqueDrawer, ExcluirProdutoDrawer } from "../drawers/EstoqueDrawer";
import { brl, downloadCsv } from "../lib/format";
import { useStore } from "../store";
import type { ItemEstoque } from "../types";

type Grupo = { chave: string; modelo: string; cor: string; itens: Map<string, ItemEstoque> };

function nivel(i: ItemEstoque): string {
  if (i.quantidade === 0) return "danger";
  if (i.quantidade <= i.minimo) return "warn";
  return "";
}

export function Estoque() {
  const store = useStore();
  const [filtro, setFiltro] = useState("todos");
  const [entrada, setEntrada] = useState(false);
  const [editar, setEditar] = useState<Grupo | null>(null);
  const [excluir, setExcluir] = useState<Grupo | null>(null);

  const modelos = unicos([...store.modelos, ...store.estoque.map((i) => i.modelo)]);
  const pecas = store.estoque.reduce((acc, i) => acc + i.quantidade, 0);
  const alerta = store.estoque.filter((i) => i.quantidade <= i.minimo).length;

  const grupos = useMemo(() => {
    const map = new Map<string, Grupo>();
    for (const i of store.estoque) {
      if (filtro !== "todos" && i.modelo.toLowerCase() !== filtro.toLowerCase()) continue;
      const chave = `${i.modelo}|${i.cor}`.toLowerCase();
      const g = map.get(chave) ?? { chave, modelo: i.modelo, cor: i.cor, itens: new Map() };
      g.itens.set(i.tamanho, i);
      map.set(chave, g);
    }
    return [...map.values()].sort(
      (a, b) => a.modelo.localeCompare(b.modelo) || a.cor.localeCompare(b.cor),
    );
  }, [store.estoque, filtro]);

  const exportar = () =>
    downloadCsv("estoque.csv", [
      ["Modelo", "Cor", "Tamanho", "Quantidade", "Mínimo", "Preço de venda", "Situação"],
      ...store.estoque.map((i) => [
        i.modelo,
        i.cor,
        i.tamanho,
        String(i.quantidade),
        String(i.minimo),
        i.precoVenda ? i.precoVenda.toFixed(2).replace(".", ",") : "",
        i.quantidade === 0 ? "Esgotado" : i.quantidade <= i.minimo ? "No mínimo" : "OK",
      ]),
    ]);

  return (
    <main className="page container">
      <div className="page-head">
        <div>
          <h1>Estoque</h1>
          <p className="sub">
            {pecas} peças lisas em estoque · {alerta} variações no mínimo ou abaixo
          </p>
        </div>
        <div className="actions">
          {store.perm.exportar ? (
            <button className="btn btn-outline" onClick={exportar}>
              Exportar planilha
            </button>
          ) : null}
          {store.perm.ajustarEstoque ? (
            <button className="btn btn-pink" onClick={() => setEntrada(true)}>
              Entrada de estoque
            </button>
          ) : null}
        </div>
      </div>

      <div className="chips" role="tablist" aria-label="Filtrar por modelo">
        <button
          role="tab"
          aria-selected={filtro === "todos"}
          className={`chip${filtro === "todos" ? " active" : ""}`}
          onClick={() => setFiltro("todos")}
        >
          Todos os modelos
        </button>
        {modelos.map((m) => (
          <button
            key={m}
            role="tab"
            aria-selected={filtro === m}
            className={`chip${filtro === m ? " active" : ""}`}
            onClick={() => setFiltro(m)}
          >
            {rotuloModelo(m)}
          </button>
        ))}
      </div>
      <p className="legend">
        <span className="qty warn">3</span> no mínimo ou abaixo <span className="qty danger">0</span>{" "}
        esgotado. Cada pedido novo dá baixa automática na peça do modelo, cor e tamanho; cancelar ou
        excluir o pedido devolve ao estoque.
      </p>

      {grupos.length === 0 ? (
        <div className="empty">
          {store.estoque.length === 0
            ? "Nenhuma peça cadastrada. Use Entrada de estoque para lançar as peças que você tem."
            : "Nenhuma peça deste modelo no estoque."}
        </div>
      ) : (
        <div className="table-wrap">
          <table className="estoque">
            <thead>
              <tr>
                <th>Modelo</th>
                <th>Cor</th>
                <th className="num">Preço</th>
                {TAMANHOS.map((t) => (
                  <th key={t} className="num">
                    {t}
                  </th>
                ))}
                <th className="num">Total</th>
                {store.perm.ajustarEstoque ? <th aria-label="Ações" /> : null}
              </tr>
            </thead>
            <tbody>
              {grupos.map((g) => {
                const total = [...g.itens.values()].reduce((acc, i) => acc + i.quantidade, 0);
                return (
                  <tr key={g.chave}>
                    <td className="strong">{g.modelo}</td>
                    <td>{g.cor}</td>
                    <td className="num">{faixaPreco([...g.itens.values()])}</td>
                    {TAMANHOS.map((t) => {
                      const item = g.itens.get(t);
                      if (!item) {
                        return (
                          <td key={t} className="num muted">
                            –
                          </td>
                        );
                      }
                      return (
                        <td key={t} className="num">
                          <span
                            className={`qty ${nivel(item)}`}
                            title={`Mínimo: ${item.minimo}${item.precoVenda ? ` · ${brl(item.precoVenda)}` : ""}`}
                          >
                            {item.quantidade}
                          </span>
                        </td>
                      );
                    })}
                    <td className="num strong">{total}</td>
                    {store.perm.ajustarEstoque ? (
                      <td>
                        <AcoesLinha
                          rotulo={`${g.modelo} ${g.cor}`}
                          onEditar={() => setEditar(g)}
                          onExcluir={() => setExcluir(g)}
                        />
                      </td>
                    ) : null}
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {entrada ? <EntradaEstoqueDrawer onClose={() => setEntrada(false)} /> : null}
      {editar ? (
        <EditarProdutoDrawer key={editar.chave} itens={[...editar.itens.values()]} onClose={() => setEditar(null)} />
      ) : null}
      {excluir ? (
        <ExcluirProdutoDrawer key={excluir.chave} itens={[...excluir.itens.values()]} onClose={() => setExcluir(null)} />
      ) : null}
    </main>
  );
}

/** Um preço quando todos os tamanhos custam igual; faixa quando variam. */
function faixaPreco(itens: ItemEstoque[]): string {
  const precos = itens.map((i) => i.precoVenda).filter((p) => p > 0);
  if (precos.length === 0) return "—";
  const min = Math.min(...precos);
  const max = Math.max(...precos);
  return min === max ? brl(min) : `${brl(min)} – ${brl(max)}`;
}
