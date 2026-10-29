import { useState } from "react";
import { COMO_CONHECEU, TAMANHOS } from "../catalog";
import { Drawer } from "../components/Drawer";
import { maskWhatsapp } from "../lib/format";
import { useStore } from "../store";
import type { Cliente } from "../types";

export const clienteEmBranco: Omit<Cliente, "id"> = {
  nome: "",
  whatsapp: "",
  email: "",
  profissao: "",
  faculdade: "",
  instagram: "",
  cidade: "",
  aniversario: "",
  tamanhoHabitual: "",
  comoConheceu: "",
  endereco: "",
  observacoes: "",
};

type Props = {
  value: Partial<Cliente>;
  onClose: () => void;
};

export function ClienteDrawer({ value, onClose }: Props) {
  const store = useStore();
  const [form, setForm] = useState<Omit<Cliente, "id"> & { id?: string }>({
    ...clienteEmBranco,
    ...value,
  });
  const [erro, setErro] = useState("");
  const editando = Boolean(form.id);

  const set = (k: keyof typeof clienteEmBranco, v: string) => setForm((f) => ({ ...f, [k]: v }));

  const salvar = () => {
    if (!form.nome.trim()) {
      setErro("Informe o nome do cliente.");
      return;
    }
    store.salvarCliente({ ...form, nome: form.nome.trim() });
    onClose();
  };

  return (
    <Drawer
      title={editando ? "Editar cliente" : "Novo cliente"}
      onClose={onClose}
      footer={
        <button className="btn btn-pink" type="button" onClick={salvar}>
          Salvar cliente
        </button>
      }
    >
      <div className="fields">
        <div className="field">
          <label htmlFor="cli-nome">Nome completo</label>
          <input
            id="cli-nome"
            autoFocus
            value={form.nome}
            aria-invalid={Boolean(erro) && !form.nome.trim()}
            onChange={(e) => set("nome", e.target.value)}
          />
        </div>
        <div className="fields two">
          <div className="field">
            <label htmlFor="cli-whats">WhatsApp</label>
            <input
              id="cli-whats"
              inputMode="tel"
              placeholder="(00) 00000-0000"
              value={form.whatsapp}
              onChange={(e) => set("whatsapp", maskWhatsapp(e.target.value))}
            />
          </div>
          <div className="field">
            <label htmlFor="cli-email">E-mail</label>
            <input
              id="cli-email"
              type="email"
              value={form.email}
              onChange={(e) => set("email", e.target.value)}
            />
          </div>
        </div>
        <div className="fields two">
          <div className="field">
            <label htmlFor="cli-prof">Profissão / curso</label>
            <input
              id="cli-prof"
              placeholder="Odontologia, Enfermagem..."
              value={form.profissao}
              onChange={(e) => set("profissao", e.target.value)}
            />
          </div>
          <div className="field">
            <label htmlFor="cli-fac">Faculdade, clínica ou empresa</label>
            <input
              id="cli-fac"
              value={form.faculdade}
              onChange={(e) => set("faculdade", e.target.value)}
            />
          </div>
        </div>
        <div className="fields three-even">
          <div className="field">
            <label htmlFor="cli-insta">Instagram</label>
            <input
              id="cli-insta"
              placeholder="@"
              value={form.instagram}
              onChange={(e) => set("instagram", e.target.value)}
            />
          </div>
          <div className="field">
            <label htmlFor="cli-cidade">Cidade</label>
            <input
              id="cli-cidade"
              value={form.cidade}
              onChange={(e) => set("cidade", e.target.value)}
            />
          </div>
          <div className="field">
            <label htmlFor="cli-aniv">Aniversário</label>
            <input
              id="cli-aniv"
              type="date"
              value={form.aniversario}
              onChange={(e) => set("aniversario", e.target.value)}
            />
          </div>
        </div>
        <div className="fields two">
          <div className="field">
            <label htmlFor="cli-tam">Tamanho habitual</label>
            <select
              id="cli-tam"
              value={form.tamanhoHabitual}
              onChange={(e) => set("tamanhoHabitual", e.target.value)}
            >
              <option value="">–</option>
              {TAMANHOS.map((t) => (
                <option key={t}>{t}</option>
              ))}
            </select>
          </div>
          <div className="field">
            <label htmlFor="cli-como">Como conheceu a loja</label>
            <select
              id="cli-como"
              value={form.comoConheceu}
              onChange={(e) => set("comoConheceu", e.target.value)}
            >
              <option value="">–</option>
              {COMO_CONHECEU.map((c) => (
                <option key={c}>{c}</option>
              ))}
            </select>
          </div>
        </div>
        <div className="field">
          <label htmlFor="cli-end">Endereço de entrega</label>
          <textarea
            id="cli-end"
            rows={2}
            value={form.endereco}
            onChange={(e) => set("endereco", e.target.value)}
          />
        </div>
        <div className="field">
          <label htmlFor="cli-obs">Observações</label>
          <textarea
            id="cli-obs"
            rows={2}
            placeholder="Preferências de bordado, medidas, etc."
            value={form.observacoes}
            onChange={(e) => set("observacoes", e.target.value)}
          />
        </div>
        {erro ? <p className="erro">{erro}</p> : null}
      </div>
    </Drawer>
  );
}
