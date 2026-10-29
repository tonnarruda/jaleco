import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { ETAPAS_ABERTAS } from "../catalog";
import { ClienteDrawer } from "../drawers/ClienteDrawer";
import { PedidoDrawer } from "../drawers/PedidoDrawer";
import { brl, formatarData, mesCurto, mesISO, mesRotulo, plural } from "../lib/format";
import { metaDoMes } from "../lib/meta";
import {
  pedidoAberto,
  pedidoAReceber,
  pedidoAtrasado,
  pedidoConta,
  pedidoEmBranco,
  qtdPecas,
  totaisPedido,
} from "../lib/pedido";
import { useStore } from "../store";
import type { Cliente, PedidoForm } from "../types";

export function Painel() {
  const store = useStore();
  const { perm, usuario } = store;
  const [pedido, setPedido] = useState<PedidoForm | null>(null);
  const [cliente, setCliente] = useState<Partial<Cliente> | null>(null);
  const mesNome = new Date().toLocaleDateString("pt-BR", { month: "long" });
  const [mesVendas, setMesVendas] = useState(mesISO());
  const ultimosMeses = useMemo(() => {
    const now = new Date();
    return Array.from({ length: 12 }, (_, i) => mesISO(new Date(now.getFullYear(), now.getMonth() - i, 1)));
  }, []);

  const metricas = useMemo(() => {
    const ym = mesISO();
    const doMes = store.pedidos.filter((p) => pedidoConta(p) && p.dataPedido.startsWith(ym));
    const meus = doMes.filter((p) => p.vendedorId === usuario.id);
    const vendas = doMes.reduce((acc, p) => acc + totaisPedido(p).total, 0);
    const abertos = store.pedidos.filter(pedidoAberto);
    return {
      vendas,
      pedidosMes: doMes.length,
      pecasMes: doMes.reduce((acc, p) => acc + qtdPecas(p), 0),
      ticket: doMes.length ? vendas / doMes.length : 0,
      receber: store.pedidos
        .filter(pedidoAReceber)
        .reduce((acc, p) => acc + totaisPedido(p).falta, 0),
      abertos: abertos.length,
      atrasados: abertos.filter((p) => pedidoAtrasado(p)).length,
      meusPedidos: meus.length,
      meusValor: meus.reduce((acc, p) => acc + totaisPedido(p).total, 0),
      minhasPecas: meus.reduce((acc, p) => acc + qtdPecas(p), 0),
    };
  }, [store.pedidos, usuario.id]);

  const faturamento = useMemo(() => {
    const now = new Date();
    const pontos = [];
    for (let i = 5; i >= 0; i--) {
      const d = new Date(now.getFullYear(), now.getMonth() - i, 1);
      const key = mesISO(d);
      const valor = store.pedidos
        .filter((p) => pedidoConta(p) && p.dataPedido.startsWith(key))
        .reduce((acc, p) => acc + totaisPedido(p).total, 0);
      pontos.push({ label: mesCurto(d.getMonth()), valor });
    }
    const max = Math.max(...pontos.map((p) => p.valor), 1);
    return pontos.map((p) => ({ ...p, pct: (p.valor / max) * 100 }));
  }, [store.pedidos]);

  // Gestor acompanha a equipe toda; vendedor vê só a própria linha.
  const porVendedor = useMemo(() => {
    const ym = mesVendas;
    const doMes = store.pedidos.filter((p) => pedidoConta(p) && p.dataPedido.startsWith(ym));
    const pessoas = store.vendedores.map((v) => ({
      id: v.id,
      nome: v.nome,
      meta: metaDoMes(v.metas, ym),
      ativo: v.acesso === "Ativo",
    }));
    return pessoas
      .map((v) => {
        const pedidos = doMes.filter((p) => p.vendedorId === v.id);
        const vendas = pedidos.reduce((acc, p) => acc + totaisPedido(p).total, 0);
        return { ...v, pedidos: pedidos.length, vendas, pct: v.meta ? (vendas / v.meta) * 100 : 0 };
      })
      .filter((v) => (perm.verFaturamento ? v.ativo || v.pedidos > 0 : v.id === usuario.id))
      .sort((a, b) => b.vendas - a.vendas || a.nome.localeCompare(b.nome));
  }, [store.pedidos, store.vendedores, perm.verFaturamento, usuario.id, mesVendas]);

  const porEtapa = ETAPAS_ABERTAS.map((etapa) => ({
    etapa,
    n: store.pedidos.filter((p) => p.etapa === etapa).length,
  }));
  const maxEtapa = Math.max(...porEtapa.map((e) => e.n), 1);

  const entregas = store.pedidos
    .filter((p) => p.etapa === "Em produção" || p.etapa === "Pronto")
    .sort((a, b) => a.prazoEntrega.localeCompare(b.prazoEntrega))
    .slice(0, 6);

  const baixo = store.estoque
    .filter((i) => i.quantidade <= i.minimo)
    .sort((a, b) => a.quantidade - b.quantidade)
    .slice(0, 6);

  return (
    <main className="page container">
      <div className="page-head">
        <div>
          <h1>Painel</h1>
          <p className="sub">Resumo de {mesNome} e o que precisa de atenção hoje.</p>
        </div>
        <div className="actions">
          <button className="btn btn-pink" onClick={() => setPedido(pedidoEmBranco(usuario.id, store.modelos[0] ?? ""))}>
            Novo pedido
          </button>
          <button className="btn btn-outline" onClick={() => setCliente({})}>
            Novo cliente
          </button>
        </div>
      </div>

      <section className={`kpis${perm.verFaturamento ? "" : " kpis-2"}`}>
        {perm.verFaturamento ? (
          <>
            <article className="card">
              <div className="kpi-label">VENDAS NO MÊS</div>
              <div className="kpi-value">{brl(metricas.vendas)}</div>
              <div className="kpi-hint">
                {plural(metricas.pedidosMes, "pedido", "pedidos")} ·{" "}
                {plural(metricas.pecasMes, "peça", "peças")}
              </div>
            </article>
            <article className="card">
              <div className="kpi-label">TICKET MÉDIO</div>
              <div className="kpi-value">{brl(metricas.ticket)}</div>
              <div className="kpi-hint">por pedido em {mesNome}</div>
            </article>
            <article className="card">
              <div className="kpi-label">A RECEBER</div>
              <div className="kpi-value">{brl(metricas.receber)}</div>
              <div className="kpi-hint">saldo de pedidos não quitados</div>
            </article>
          </>
        ) : (
          <article className="card">
            <div className="kpi-label">MINHAS VENDAS NO MÊS</div>
            <div className="kpi-value">{brl(metricas.meusValor)}</div>
            <div className="kpi-hint">
              {plural(metricas.meusPedidos, "pedido", "pedidos")} ·{" "}
              {plural(metricas.minhasPecas, "peça", "peças")}
            </div>
          </article>
        )}
        <article className="card">
          <div className="kpi-label">PEDIDOS EM ABERTO</div>
          <div className="kpi-value">{metricas.abertos}</div>
          <div className={`kpi-hint${metricas.atrasados ? " danger" : ""}`}>
            {metricas.atrasados === 0
              ? "nenhum atrasado"
              : plural(metricas.atrasados, "atrasado", "atrasados")}
          </div>
        </article>
      </section>

      {porVendedor.length > 0 ? (
        <section className="grid-1">
          <article className="card">
            <div className="card-title">
              <strong>{perm.verFaturamento ? "Vendas por vendedor" : "Minha meta"}</strong>
              <span className="card-title-aside">
                sem orçamentos e cancelados
                <select
                  className="mes-select"
                  aria-label="Mês"
                  value={mesVendas}
                  onChange={(e) => setMesVendas(e.target.value)}
                >
                  {ultimosMeses.map((m) => (
                    <option key={m} value={m}>
                      {mesRotulo(m)}
                    </option>
                  ))}
                </select>
              </span>
            </div>
            <div className="metas">
              {porVendedor.map((v) => (
                <div className="meta-row" key={v.id}>
                  <span className="meta-nome">
                    {v.nome}
                    <small>{plural(v.pedidos, "pedido", "pedidos")}</small>
                  </span>
                  <div className="etapa-bar">
                    <span
                      className={v.pct >= 100 ? "atingida" : undefined}
                      style={{ width: `${Math.min(v.pct, 100)}%` }}
                    />
                  </div>
                  <span className="meta-valor">
                    <strong>{brl(v.vendas)}</strong>
                    <small>{v.meta ? `de ${brl(v.meta)}` : "sem meta"}</small>
                  </span>
                  <span className={`meta-pct${v.pct >= 100 ? " ok" : ""}`}>
                    {v.meta ? `${Math.round(v.pct)}%` : "—"}
                  </span>
                </div>
              ))}
            </div>
          </article>
        </section>
      ) : null}

      <section className={perm.verFaturamento ? "grid-2" : "grid-1"}>
        {perm.verFaturamento ? (
          <article className="card">
            <div className="card-title">
              <strong>Faturamento</strong>
              <span>últimos 6 meses, sem orçamentos e cancelados</span>
            </div>
            <div className="chart" role="img" aria-label="Faturamento dos últimos 6 meses">
              <div className="chart-grid" aria-hidden>
                <span />
                <span />
                <span />
                <span />
              </div>
              {faturamento.map((p, i) => (
                <div className="bar-col" key={p.label} title={`${p.label}: ${brl(p.valor)}`}>
                  <div className="bar-track">
                    <div
                      className={`bar${i === faturamento.length - 1 ? " current" : ""}`}
                      style={{ height: `max(3px, ${p.pct}%)` }}
                    />
                  </div>
                  <div className="bar-label">{p.label}</div>
                </div>
              ))}
            </div>
          </article>
        ) : null}
        <article className="card">
          <div className="card-title">
            <strong>Pedidos por etapa</strong>
            <Link className="link-pink" to="/pedidos">
              ver todos
            </Link>
          </div>
          <div className="etapas">
            {porEtapa.map((e) => (
              <div className="etapa-row" key={e.etapa}>
                <span>{e.etapa}</span>
                <div className="etapa-bar">
                  <span style={{ width: `${(e.n / maxEtapa) * 100}%` }} />
                </div>
                <span className="etapa-n">{e.n}</span>
              </div>
            ))}
          </div>
        </article>
      </section>

      <section className="grid-2">
        <article className="card">
          <div className="card-title">
            <strong>Próximas entregas</strong>
            <span>por prazo</span>
          </div>
          {entregas.length === 0 ? (
            <p className="card-empty">Nenhum pedido em produção agora.</p>
          ) : (
            <ul className="mini-list">
              {entregas.map((p) => (
                <li key={p.id}>
                  <button type="button" onClick={() => setPedido(p)}>
                    <span>
                      <strong>#{p.numero}</strong> {store.nomeCliente(p.clienteId)}
                      <small>{p.etapa}</small>
                    </span>
                    <span className={pedidoAtrasado(p) ? "danger" : "muted"}>
                      {formatarData(p.prazoEntrega)}
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </article>
        <article className="card">
          <div className="card-title">
            <strong>Estoque baixo</strong>
            <Link className="link-pink" to="/estoque">
              repor
            </Link>
          </div>
          {baixo.length === 0 ? (
            <p className="card-empty">Tudo acima do estoque mínimo.</p>
          ) : (
            <ul className="mini-list">
              {baixo.map((i) => (
                <li key={i.id}>
                  <div>
                    <span>
                      {i.modelo}
                      <small>
                        {i.cor} · {i.tamanho}
                      </small>
                    </span>
                    <span className={`qty ${i.quantidade === 0 ? "danger" : "warn"}`}>
                      {i.quantidade}
                    </span>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </article>
      </section>

      {pedido ? <PedidoDrawer key={pedido.id ?? "novo"} value={pedido} onClose={() => setPedido(null)} /> : null}
      {cliente ? <ClienteDrawer value={cliente} onClose={() => setCliente(null)} /> : null}
    </main>
  );
}
