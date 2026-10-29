import { useStore } from "../store";

/** Avisos de sucesso: só aparecem depois que o servidor confirmou a gravação. */
export function Notificacoes() {
  const { notificacoes, dispensarNotificacao } = useStore();
  return (
    <div className="notificacoes" role="status" aria-live="polite">
      {notificacoes.map((n) => (
        <div className="notificacao" key={n.id}>
          <svg
            className="notificacao-icone"
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2.5"
            strokeLinecap="round"
            strokeLinejoin="round"
            aria-hidden
          >
            <circle cx="12" cy="12" r="10" />
            <path d="m8 12 3 3 5-6" />
          </svg>
          <span>{n.texto}</span>
          <button
            type="button"
            className="notificacao-fechar"
            aria-label="Fechar aviso"
            onClick={() => dispensarNotificacao(n.id)}
          >
            ×
          </button>
        </div>
      ))}
    </div>
  );
}
