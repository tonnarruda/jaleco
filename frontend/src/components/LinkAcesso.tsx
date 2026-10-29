import { useEffect, useRef, useState } from "react";
import type { LinkAcesso } from "../lib/api";
import type { Vendedor } from "../types";

type Props = {
  vendedor: Vendedor;
  link: LinkAcesso;
  onFechar: () => void;
};

/** Mostra o link de acesso para o admin copiar ou mandar pelo WhatsApp. */
export function LinkAcessoDialog({ vendedor, link, onFechar }: Props) {
  const [copiado, setCopiado] = useState(false);
  const fecharRef = useRef<HTMLButtonElement>(null);
  const url = `${window.location.origin}/convite/${link.token}`;
  const validade = new Date(link.expiraEm).toLocaleString("pt-BR", {
    day: "2-digit",
    month: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
  const primeiroNome = vendedor.nome.split(" ")[0];
  const mensagem =
    link.tipo === "convite"
      ? `Oi, ${primeiroNome}! Seu acesso ao sistema da I Love Jalecos: abra o link e crie sua senha (vale até ${validade}).\n${url}`
      : `Oi, ${primeiroNome}! Link para definir uma nova senha no sistema da I Love Jalecos (vale até ${validade}).\n${url}`;
  const whatsapp = vendedor.whatsapp.replace(/\D/g, "");

  useEffect(() => {
    fecharRef.current?.focus();
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.stopPropagation();
        onFechar();
      }
    };
    window.addEventListener("keydown", onKey, true);
    return () => window.removeEventListener("keydown", onKey, true);
  }, [onFechar]);

  const copiar = async () => {
    try {
      await navigator.clipboard.writeText(url);
      setCopiado(true);
    } catch {
      // Sem permissão de área de transferência: o campo fica selecionado para copiar à mão.
      document.querySelector<HTMLInputElement>("#link-acesso")?.select();
    }
  };

  return (
    <>
      <div className="confirmacao-fundo" onClick={onFechar} />
      <div className="confirmacao link-acesso" role="dialog" aria-modal="true" aria-labelledby="link-titulo">
        <h2 id="link-titulo">
          {link.tipo === "convite" ? "Link de acesso" : "Link para nova senha"} · {vendedor.nome}
        </h2>
        <p className="confirmacao-mensagem">
          Envie para <strong>{vendedor.nome}</strong>. Vale até <strong>{validade}</strong> e só pode ser usado uma
          vez. Gerar outro link invalida este.
          {link.tipo === "redefinicao" ? " Ao criar a senha nova, a pessoa sai dos aparelhos em que estava logada." : ""}
        </p>
        <input
          id="link-acesso"
          className="link-acesso-campo"
          readOnly
          value={url}
          aria-label="Link de acesso"
          onFocus={(e) => e.target.select()}
        />
        <div className="confirmacao-acoes">
          <button ref={fecharRef} type="button" className="btn btn-outline" onClick={onFechar}>
            Fechar
          </button>
          {whatsapp ? (
            <a
              className="btn btn-outline"
              href={`https://wa.me/55${whatsapp}?text=${encodeURIComponent(mensagem)}`}
              target="_blank"
              rel="noreferrer"
            >
              Enviar pelo WhatsApp
            </a>
          ) : null}
          <button type="button" className="btn btn-pink" onClick={copiar}>
            {copiado ? "Copiado!" : "Copiar link"}
          </button>
        </div>
      </div>
    </>
  );
}
