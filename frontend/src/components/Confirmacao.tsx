import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from "react";

export type OpcoesConfirmacao = {
  titulo: string;
  mensagem: ReactNode;
  /** Texto do botão de confirmar; por padrão "Excluir". */
  confirmar?: string;
  /** Só informa (ex.: exclusão bloqueada): um botão "Entendi", e a promessa resolve false. */
  somenteAviso?: boolean;
};

type Pedido = OpcoesConfirmacao & { responder: (ok: boolean) => void };

const ConfirmacaoContext = createContext<((opcoes: OpcoesConfirmacao) => Promise<boolean>) | null>(null);

/** Substitui o confirm() do navegador por uma janela nas cores do sistema. */
export function ConfirmacaoProvider({ children }: { children: ReactNode }) {
  const [pedido, setPedido] = useState<Pedido | null>(null);

  const confirmar = useCallback(
    (opcoes: OpcoesConfirmacao) =>
      new Promise<boolean>((resolve) => {
        setPedido({
          ...opcoes,
          responder: (ok) => {
            setPedido(null);
            resolve(ok);
          },
        });
      }),
    [],
  );

  return (
    <ConfirmacaoContext.Provider value={confirmar}>
      {children}
      {pedido ? <JanelaConfirmacao pedido={pedido} /> : null}
    </ConfirmacaoContext.Provider>
  );
}

/** `if (await confirmar({ titulo, mensagem })) excluir()`; com `somenteAviso`, só informa. */
export function useConfirmacao() {
  const ctx = useContext(ConfirmacaoContext);
  if (!ctx) throw new Error("useConfirmacao fora do ConfirmacaoProvider");
  return ctx;
}

function JanelaConfirmacao({ pedido }: { pedido: Pedido }) {
  const aviso = Boolean(pedido.somenteAviso);
  const cancelarRef = useRef<HTMLButtonElement>(null);
  const confirmarRef = useRef<HTMLButtonElement>(null);
  const responder = useRef(pedido.responder);
  responder.current = pedido.responder;

  useEffect(() => {
    const anterior = document.activeElement as HTMLElement | null;
    // Foco em Cancelar: Enter sem pensar não exclui nada.
    cancelarRef.current?.focus();
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        // Captura na window: o Esc fecha só esta janela, não a gaveta que está atrás.
        e.stopPropagation();
        responder.current(false);
      } else if (e.key === "Tab") {
        // Mantém o foco nos botões da janela enquanto ela está aberta (no aviso, só há um).
        e.preventDefault();
        const destino =
          document.activeElement === cancelarRef.current ? confirmarRef.current : cancelarRef.current;
        destino?.focus();
      }
    };
    window.addEventListener("keydown", onKey, true);
    return () => {
      window.removeEventListener("keydown", onKey, true);
      anterior?.focus();
    };
  }, []);

  return (
    <>
      <div className="confirmacao-fundo" onClick={() => pedido.responder(false)} />
      <div
        className="confirmacao"
        role="alertdialog"
        aria-modal="true"
        aria-labelledby="confirmacao-titulo"
        aria-describedby="confirmacao-mensagem"
      >
        <div className={`confirmacao-icone${aviso ? " aviso" : ""}`} aria-hidden>
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            {aviso ? (
              <>
                <path d="M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0Z" />
                <path d="M12 9v4M12 17h.01" />
              </>
            ) : (
              <>
                <path d="M3 6h18" />
                <path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                <path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6" />
              </>
            )}
          </svg>
        </div>
        <h2 id="confirmacao-titulo">{pedido.titulo}</h2>
        <div id="confirmacao-mensagem" className="confirmacao-mensagem">
          {pedido.mensagem}
        </div>
        <div className="confirmacao-acoes">
          {aviso ? (
            <button ref={cancelarRef} type="button" className="btn btn-pink" onClick={() => pedido.responder(false)}>
              Entendi
            </button>
          ) : (
            <>
              <button ref={cancelarRef} type="button" className="btn btn-outline" onClick={() => pedido.responder(false)}>
                Cancelar
              </button>
              <button ref={confirmarRef} type="button" className="btn btn-danger" onClick={() => pedido.responder(true)}>
                {pedido.confirmar ?? "Excluir"}
              </button>
            </>
          )}
        </div>
      </div>
    </>
  );
}
