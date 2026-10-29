import { useMemo, useState } from "react";
import { AcoesLinha } from "../components/AcoesLinha";
import { ClienteDrawer } from "../drawers/ClienteDrawer";
import { useExclusoes } from "../lib/exclusoes";
import { brl, downloadCsv, formatarData, plural } from "../lib/format";
import { pedidoConta, totaisPedido } from "../lib/pedido";
import { useStore } from "../store";
import type { Cliente } from "../types";

export function Clientes() {
  const store = useStore();
  const [q, setQ] = useState("");
  const [form, setForm] = useState<Partial<Cliente> | null>(null);
  const excluir = useExclusoes();

  const resumo = useMemo(() => {
    const porCliente = new Map<string, { pedidos: number; total: number }>();
    for (const p of store.pedidos) {
      if (!pedidoConta(p)) continue;
      const r = porCliente.get(p.clienteId) ?? { pedidos: 0, total: 0 };
      r.pedidos += 1;
      r.total += totaisPedido(p).total;
      porCliente.set(p.clienteId, r);
    }
    return porCliente;
  }, [store.pedidos]);

  const lista = useMemo(() => {
    const s = q.trim().toLowerCase();
    const digitos = s.replace(/\D/g, "");
    return [...store.clientes]
      .filter((c) => {
        if (!s) return true;
        const texto = `${c.nome} ${c.whatsapp} ${c.profissao} ${c.faculdade} ${c.instagram} ${c.cidade}`.toLowerCase();
        return texto.includes(s) || (digitos.length >= 3 && c.whatsapp.replace(/\D/g, "").includes(digitos));
      })
      .sort((a, b) => a.nome.localeCompare(b.nome));
  }, [store.clientes, q]);

  const exportar = () =>
    downloadCsv("clientes.csv", [
      [
        "Nome",
        "WhatsApp",
        "E-mail",
        "Profissão",
        "Faculdade/empresa",
        "Instagram",
        "Cidade",
        "Aniversário",
        "Tamanho",
        "Como conheceu",
        "Endereço",
        "Pedidos",
        "Total comprado",
      ],
      ...store.clientes.map((c) => {
        const r = resumo.get(c.id);
        return [
          c.nome,
          c.whatsapp,
          c.email,
          c.profissao,
          c.faculdade,
          c.instagram,
          c.cidade,
          formatarData(c.aniversario),
          c.tamanhoHabitual,
          c.comoConheceu,
          c.endereco,
          String(r?.pedidos ?? 0),
          (r?.total ?? 0).toFixed(2).replace(".", ","),
        ];
      }),
    ]);

  return (
    <main className="page container">
      <div className="page-head">
        <div>
          <h1>Clientes</h1>
          <p className="sub">
            {plural(store.clientes.length, "cliente cadastrado", "clientes cadastrados")}
          </p>
        </div>
        <div className="actions">
          {store.perm.exportar ? (
            <button className="btn btn-outline" onClick={exportar}>
              Exportar planilha
            </button>
          ) : null}
          <button className="btn btn-pink" onClick={() => setForm({})}>
            Novo cliente
          </button>
        </div>
      </div>

      <input
        className="search"
        type="search"
        aria-label="Buscar clientes"
        placeholder="Buscar por nome, WhatsApp, profissão, faculdade..."
        value={q}
        onChange={(e) => setQ(e.target.value)}
      />

      {lista.length === 0 ? (
        <div className="empty">
          {store.clientes.length === 0
            ? "Nenhum cliente ainda. Clique em Novo cliente para cadastrar o primeiro."
            : "Nenhum cliente encontrado para essa busca."}
        </div>
      ) : (
        <div className="list">
          {lista.map((c) => {
            const r = resumo.get(c.id);
            const meta = [c.whatsapp, c.profissao, c.faculdade, c.cidade].filter(Boolean);
            return (
              <div key={c.id} className="row">
                <div className="row-main">
                  <div className="row-title">{c.nome}</div>
                  <div className="row-meta">{meta.length ? meta.join(" · ") : "sem contato"}</div>
                </div>
                {c.tamanhoHabitual ? <span className="badge">{c.tamanhoHabitual}</span> : <span />}
                <div className="row-money">
                  <strong>{plural(r?.pedidos ?? 0, "pedido", "pedidos")}</strong>
                  {store.perm.verFaturamento && r ? (
                    <small className="muted">{brl(r.total)}</small>
                  ) : null}
                </div>
                <AcoesLinha
                  rotulo={c.nome}
                  onEditar={() => setForm(c)}
                  onExcluir={store.perm.excluir ? () => excluir.cliente(c) : undefined}
                />
              </div>
            );
          })}
        </div>
      )}

      {form ? <ClienteDrawer key={form.id ?? "novo"} value={form} onClose={() => setForm(null)} /> : null}
    </main>
  );
}
