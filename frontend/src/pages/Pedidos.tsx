import { useMemo, useState } from "react";
import { FILTROS_PEDIDO } from "../catalog";
import { AcoesLinha } from "../components/AcoesLinha";
import { PedidoDrawer } from "../drawers/PedidoDrawer";
import { useExclusoes } from "../lib/exclusoes";
import { brl, downloadCsv, formatarData } from "../lib/format";
import {
  pedidoAberto,
  pedidoAReceber,
  pedidoAtrasado,
  pedidoEmBranco,
  qtdPecas,
  totaisPedido,
} from "../lib/pedido";
import { useStore } from "../store";
import type { FiltroPedido, Pedido, PedidoForm } from "../types";

function passaFiltro(p: Pedido, filtro: FiltroPedido): boolean {
  if (filtro === "Todos") return true;
  if (filtro === "Em aberto") return pedidoAberto(p);
  if (filtro === "Atrasados") return pedidoAtrasado(p);
  if (filtro === "A receber") return pedidoAReceber(p);
  return p.etapa === filtro;
}

export function Pedidos() {
  const store = useStore();
  const [q, setQ] = useState("");
  const [filtro, setFiltro] = useState<FiltroPedido>("Em aberto");
  const [pedido, setPedido] = useState<PedidoForm | null>(null);
  const excluir = useExclusoes();

  const filtrados = useMemo(() => {
    const busca = q.trim().toLowerCase().replace(/^#/, "");
    return store.pedidos
      .filter((p) => {
        if (!passaFiltro(p, filtro)) return false;
        if (!busca) return true;
        const texto = [
          store.nomeCliente(p.clienteId),
          String(p.numero),
          ...p.pecas.map((x) => x.nomeBordado),
        ]
          .join(" ")
          .toLowerCase();
        return texto.includes(busca);
      })
      .sort((a, b) => b.numero - a.numero);
  }, [store, q, filtro]);

  const exportar = () => {
    downloadCsv("pedidos.csv", [
      [
        "Nº",
        "Data",
        "Cliente",
        "Vendedor",
        "Canal",
        "Etapa",
        "Prazo",
        "Peças",
        "Bordados",
        "Forma",
        "Situação",
        "Total",
        "Pago",
        "Falta",
        "Rastreio",
      ],
      ...[...store.pedidos]
        .sort((a, b) => a.numero - b.numero)
        .map((p) => {
          const t = totaisPedido(p);
          return [
            String(p.numero),
            formatarData(p.dataPedido),
            store.nomeCliente(p.clienteId),
            store.nomeVendedor(p.vendedorId),
            p.canal,
            p.etapa,
            formatarData(p.prazoEntrega),
            p.pecas.map((x) => `${x.quantidade}x ${x.modelo} ${x.cor} ${x.tamanho}`).join(" | "),
            p.pecas.map((x) => [x.nomeBordado, x.profissaoBordado].filter(Boolean).join(" - ")).join(" | "),
            p.formaPagamento,
            p.situacaoPagamento,
            t.total.toFixed(2).replace(".", ","),
            p.valorPago.toFixed(2).replace(".", ","),
            t.falta.toFixed(2).replace(".", ","),
            p.codigoRastreio,
          ];
        }),
    ]);
  };

  return (
    <main className="page container">
      <div className="page-head">
        <div>
          <h1>Pedidos</h1>
          <p className="sub">
            {filtrados.length} de {store.pedidos.length} pedidos
          </p>
        </div>
        <div className="actions">
          {store.perm.exportar ? (
            <button className="btn btn-outline" onClick={exportar}>
              Exportar planilha
            </button>
          ) : null}
          <button className="btn btn-pink" onClick={() => setPedido(pedidoEmBranco(store.usuario.id, store.modelos[0] ?? ""))}>
            Novo pedido
          </button>
        </div>
      </div>

      <input
        className="search"
        type="search"
        aria-label="Buscar pedidos"
        placeholder="Buscar por cliente, nome bordado, nº do pedido..."
        value={q}
        onChange={(e) => setQ(e.target.value)}
      />
      <div className="chips" role="tablist" aria-label="Filtrar pedidos">
        {FILTROS_PEDIDO.map((f) => (
          <button
            key={f}
            role="tab"
            aria-selected={filtro === f}
            className={`chip${filtro === f ? " active" : ""}`}
            onClick={() => setFiltro(f)}
          >
            {f}
          </button>
        ))}
      </div>

      {filtrados.length === 0 ? (
        <div className="empty">
          {store.pedidos.length === 0
            ? "Nenhum pedido neste filtro. Clique em Novo pedido para registrar a primeira venda."
            : "Nenhum pedido neste filtro."}
        </div>
      ) : (
        <div className="list">
          {filtrados.map((p) => {
            const t = totaisPedido(p);
            const atrasado = pedidoAtrasado(p);
            const bordados = p.pecas.map((x) => x.nomeBordado).filter(Boolean);
            return (
              <div key={p.id} className="row">
                <div className="row-main">
                  <div className="row-title">
                    <span className="muted">#{p.numero}</span> {store.nomeCliente(p.clienteId)}
                  </div>
                  <div className="row-meta">
                    {qtdPecas(p)} {qtdPecas(p) === 1 ? "peça" : "peças"}
                    {bordados.length ? ` · ${bordados.join(", ")}` : ""} · pedido{" "}
                    {formatarData(p.dataPedido)} ·{" "}
                    <span className={atrasado ? "danger" : ""}>
                      prazo {formatarData(p.prazoEntrega)}
                      {atrasado ? " (atrasado)" : ""}
                    </span>
                  </div>
                </div>
                <span className={`badge etapa-${p.etapa.replace(/\s+/g, "-").toLowerCase()}`}>
                  {p.etapa}
                </span>
                <div className="row-money">
                  <strong>{brl(t.total)}</strong>
                  {t.falta > 0 && p.etapa !== "Cancelado" ? (
                    <small className="danger">falta {brl(t.falta)}</small>
                  ) : (
                    <small className="muted">{p.etapa === "Cancelado" ? "—" : "quitado"}</small>
                  )}
                </div>
                <AcoesLinha
                  rotulo={`pedido #${p.numero}`}
                  onEditar={() => setPedido(p)}
                  onExcluir={store.perm.excluir ? () => excluir.pedido(p) : undefined}
                />
              </div>
            );
          })}
        </div>
      )}

      {pedido ? <PedidoDrawer key={pedido.id ?? "novo"} value={pedido} onClose={() => setPedido(null)} /> : null}
    </main>
  );
}
