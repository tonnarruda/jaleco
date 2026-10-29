import type { ReactNode } from "react";
import { IconeEditar, IconeExcluir } from "./Icones";

type Props = {
  /** Nome do item, para o leitor de tela: "Editar Maria", "Excluir Maria". */
  rotulo: string;
  onEditar?: () => void;
  onExcluir?: () => void;
  /** Ações próprias da lista, antes de editar e excluir (ex.: link de acesso na Equipe). */
  extra?: ReactNode;
};

/** Ícones de editar e excluir no fim da linha. Sem a função, o ícone não aparece (sem permissão). */
export function AcoesLinha({ rotulo, onEditar, onExcluir, extra }: Props) {
  return (
    <div className="acoes">
      {extra}
      {onEditar ? (
        <button type="button" className="icon-btn" title="Editar" aria-label={`Editar ${rotulo}`} onClick={onEditar}>
          <IconeEditar />
        </button>
      ) : null}
      {onExcluir ? (
        <button
          type="button"
          className="icon-btn danger"
          title="Excluir"
          aria-label={`Excluir ${rotulo}`}
          onClick={onExcluir}
        >
          <IconeExcluir />
        </button>
      ) : null}
    </div>
  );
}
