import { useState } from "react";
import { PAPEIS } from "../catalog";
import { Drawer } from "../components/Drawer";
import { brl, maskWhatsapp, mesISO, mesRotulo } from "../lib/format";
import { metaDoMes, registrarMeta } from "../lib/meta";
import { useStore } from "../store";
import type { AcessoVendedor, PapelUsuario, Vendedor } from "../types";

type Props = {
  value: Partial<Vendedor>;
  onClose: () => void;
};

export function VendedorDrawer({ value, onClose }: Props) {
  const store = useStore();
  const [nome, setNome] = useState(value.nome ?? "");
  const [whatsapp, setWhatsapp] = useState(value.whatsapp ?? "");
  const [papel, setPapel] = useState<PapelUsuario>(value.papel ?? "Vendedor(a)");
  const [email, setEmail] = useState(value.email ?? "");
  const [acesso, setAcesso] = useState<AcessoVendedor>(value.acesso ?? "Ativo");
  const [comissao, setComissao] = useState(String(value.comissao ?? 0));
  const metas = value.metas ?? [];
  const [mesMeta, setMesMeta] = useState(mesISO());
  const [meta, setMeta] = useState(String(metaDoMes(metas, mesISO())));
  const [erro, setErro] = useState("");

  const salvar = () => {
    if (!nome.trim()) {
      setErro("Informe o nome que vai aparecer nas vendas.");
      return;
    }
    if (email.trim() && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim())) {
      setErro("Informe um e-mail válido, ou deixe em branco.");
      return;
    }
    const percentual = Number(comissao.replace(",", "."));
    if (comissao.trim() === "" || !Number.isFinite(percentual) || percentual < 0 || percentual > 100) {
      setErro("Informe a comissão entre 0 e 100%.");
      return;
    }
    const metaMes = meta.trim() === "" ? 0 : Number(meta.replace(",", "."));
    if (!Number.isFinite(metaMes) || metaMes < 0) {
      setErro("Informe a meta em reais, ou 0 para sem meta.");
      return;
    }
    if (!/^\d{4}-\d{2}$/.test(mesMeta)) {
      setErro("Informe o mês a partir do qual a meta vale.");
      return;
    }
    store.salvarVendedor({
      id: value.id,
      nome: nome.trim(),
      whatsapp,
      email: email.trim().toLowerCase(),
      papel,
      acesso,
      comissao: percentual,
      metas: registrarMeta(metas, mesMeta, metaMes),
    });
    onClose();
  };

  return (
    <Drawer
      title={value.id ? "Editar vendedor" : "Cadastrar vendedor"}
      onClose={onClose}
      footer={
        <button className="btn btn-pink" type="button" onClick={salvar}>
          Salvar vendedor
        </button>
      }
    >
      <div className="fields">
        <div className="field">
          <label htmlFor="vend-nome">Nome (aparece nas vendas)</label>
          <input id="vend-nome" autoFocus value={nome} onChange={(e) => setNome(e.target.value)} />
        </div>
        <div className="field">
          <label htmlFor="vend-whats">WhatsApp</label>
          <input
            id="vend-whats"
            inputMode="tel"
            value={whatsapp}
            onChange={(e) => setWhatsapp(maskWhatsapp(e.target.value))}
          />
        </div>
        <div className="field">
          <label htmlFor="vend-email">E-mail (login no sistema)</label>
          <input
            id="vend-email"
            type="email"
            inputMode="email"
            autoComplete="off"
            placeholder="nome@exemplo.com"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
          <p className="hint">
            {value.possuiLogin
              ? "Esta pessoa já tem acesso. Trocar o e-mail muda o login dela."
              : "Depois de salvar, use o ícone de link na lista da Equipe para gerar o acesso."}
          </p>
        </div>
        <div className="fields two">
          <div className="field">
            <label htmlFor="vend-papel">Papel</label>
            <select
              id="vend-papel"
              value={papel}
              onChange={(e) => setPapel(e.target.value as PapelUsuario)}
            >
              {PAPEIS.map((p) => (
                <option key={p}>{p}</option>
              ))}
            </select>
          </div>
          <div className="field">
            <label htmlFor="vend-acesso">Acesso</label>
            <select
              id="vend-acesso"
              value={acesso}
              onChange={(e) => setAcesso(e.target.value as AcessoVendedor)}
            >
              <option>Ativo</option>
              <option>Inativo</option>
            </select>
          </div>
        </div>
        <div className="fields two">
          <div className="field">
            <label htmlFor="vend-comissao">Comissão (% sobre a venda)</label>
            <input
              id="vend-comissao"
              type="number"
              inputMode="decimal"
              min={0}
              max={100}
              step={0.5}
              value={comissao}
              aria-invalid={erro.startsWith("Informe a comissão")}
              onChange={(e) => setComissao(e.target.value)}
            />
          </div>
        </div>
        <div className="fields two">
          <div className="field">
            <label htmlFor="vend-meta-mes">Meta vale a partir de</label>
            <input
              id="vend-meta-mes"
              type="month"
              value={mesMeta}
              aria-invalid={erro.startsWith("Informe o mês")}
              onChange={(e) => {
                setMesMeta(e.target.value);
                setMeta(String(metaDoMes(metas, e.target.value)));
              }}
            />
          </div>
          <div className="field">
            <label htmlFor="vend-meta">Meta de vendas no mês (R$)</label>
            <input
              id="vend-meta"
              type="number"
              inputMode="decimal"
              min={0}
              step={100}
              value={meta}
              aria-invalid={erro.startsWith("Informe a meta")}
              onChange={(e) => setMeta(e.target.value)}
            />
          </div>
        </div>
        {metas.length > 0 ? (
          <div className="meta-historico">
            <div className="field-label">Histórico de metas</div>
            <ul className="mini-list">
              {[...metas].reverse().map((m) => (
                <li key={m.desde}>
                  <div>
                    <span>desde {mesRotulo(m.desde)}</span>
                    <strong>{m.valor ? brl(m.valor) : "sem meta"}</strong>
                  </div>
                </li>
              ))}
            </ul>
          </div>
        ) : null}
        <p className="help">
          Marcar como Inativo tira o acesso na hora: a pessoa sai do sistema e o link pendente deixa de valer.
        </p>
        {erro ? <p className="erro">{erro}</p> : null}
      </div>
    </Drawer>
  );
}
