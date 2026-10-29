import { useMemo, useState } from "react";
import { Navigate } from "react-router-dom";
import { AcoesLinha } from "../components/AcoesLinha";
import { useConfirmacao } from "../components/Confirmacao";
import { IconeLink } from "../components/Icones";
import { LinkAcessoDialog } from "../components/LinkAcesso";
import { VendedorDrawer } from "../drawers/VendedorDrawer";
import { acessoApi, type LinkAcesso } from "../lib/api";
import { useExclusoes } from "../lib/exclusoes";
import { brl, mesISO, plural } from "../lib/format";
import { metaDoMes } from "../lib/meta";
import { pedidoConta } from "../lib/pedido";
import { useStore } from "../store";
import type { Vendedor } from "../types";

export function Equipe() {
  const store = useStore();
  const [form, setForm] = useState<Partial<Vendedor> | null>(null);
  const excluir = useExclusoes();
  const confirmar = useConfirmacao();
  const [link, setLink] = useState<{ vendedor: Vendedor; link: LinkAcesso } | null>(null);

  const gerarLink = async (v: Vendedor) => {
    if (!v.email) {
      await confirmar({
        somenteAviso: true,
        titulo: "Falta o e-mail",
        mensagem: (
          <>
            Cadastre o e-mail de <strong>{v.nome}</strong> (ícone de editar) antes de gerar o link de acesso: é
            com ele que a pessoa entra no sistema.
          </>
        ),
      });
      return;
    }
    try {
      setLink({ vendedor: v, link: await acessoApi.gerarLink(v.id) });
    } catch (e) {
      await confirmar({ somenteAviso: true, titulo: "Não foi possível gerar o link", mensagem: (e as Error).message });
    }
  };

  const vendasMes = useMemo(() => {
    const ym = mesISO();
    const map = new Map<string, number>();
    for (const p of store.pedidos) {
      if (pedidoConta(p) && p.dataPedido.startsWith(ym)) {
        map.set(p.vendedorId, (map.get(p.vendedorId) ?? 0) + 1);
      }
    }
    return map;
  }, [store.pedidos]);

  if (!store.perm.verEquipe) return <Navigate to="/" replace />;
  const podeEditar = store.perm.gerirEquipe;

  return (
    <main className="page container">
      <div className="page-head">
        <div>
          <h1>Equipe</h1>
          <p className="sub">
            Quem pode registrar vendas e o que cada um pode fazer. Para dar acesso, cadastre o e-mail e envie o
            link (ícone de link).
          </p>
        </div>
        {podeEditar ? (
          <button className="btn btn-pink" onClick={() => setForm({})}>
            Cadastrar vendedor
          </button>
        ) : null}
      </div>

      {store.vendedores.length === 0 ? (
        <div className="empty">Nenhum vendedor cadastrado. Clique em Cadastrar vendedor.</div>
      ) : (
        <div className="list">
          {[...store.vendedores]
            .sort((a, b) => a.nome.localeCompare(b.nome))
            .map((v) => (
              <div key={v.id} className="row row-equipe">
                <div className="row-main">
                  <div className="row-title">{v.nome}</div>
                  <div className="row-meta">
                    {[v.papel, `Comissão ${v.comissao.toLocaleString("pt-BR")}%`, metaDoMes(v.metas, mesISO()) ? `Meta ${brl(metaDoMes(v.metas, mesISO()))}` : "", v.whatsapp, plural(vendasMes.get(v.id) ?? 0, "venda no mês", "vendas no mês")]
                      .filter(Boolean)
                      .join(" · ")}
                  </div>
                </div>
                <span className={`badge ${v.acesso === "Ativo" ? "ok" : "off"}`}>{v.acesso}</span>
                <small className={`acesso-status${v.possuiLogin ? "" : " pendente"}`}>
                  {!v.email ? "Sem e-mail" : v.possuiLogin ? "Com acesso" : "Aguardando 1º acesso"}
                </small>
                <AcoesLinha
                  rotulo={v.nome}
                  onEditar={podeEditar ? () => setForm(v) : undefined}
                  onExcluir={podeEditar && v.id !== store.usuario.id ? () => excluir.vendedor(v) : undefined}
                  extra={
                    podeEditar && v.acesso === "Ativo" ? (
                      <button
                        type="button"
                        className="icon-btn"
                        title={v.possuiLogin ? "Gerar link para nova senha" : "Gerar link de acesso"}
                        aria-label={`${v.possuiLogin ? "Gerar link para nova senha de" : "Gerar link de acesso de"} ${v.nome}`}
                        onClick={() => gerarLink(v)}
                      >
                        <IconeLink />
                      </button>
                    ) : null
                  }
                />
              </div>
            ))}
        </div>
      )}

      {form ? <VendedorDrawer key={form.id ?? "novo"} value={form} onClose={() => setForm(null)} /> : null}
      {link ? <LinkAcessoDialog vendedor={link.vendedor} link={link.link} onFechar={() => setLink(null)} /> : null}
    </main>
  );
}
