import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { acessoApi, ErroApi } from "./lib/api";
import type { Usuario } from "./types";

type Sessao = {
  /** undefined enquanto pergunta ao servidor; null quando não há ninguém logado. */
  usuario: Usuario | null | undefined;
  /** Mensagem para a tela de entrada (ex.: sessão expirou). */
  aviso: string;
  entrou: (u: Usuario) => void;
  sair: () => Promise<void>;
  /**
   * Chamado quando a API responde 401 no meio do uso. Com `perdeuAlteracao`, a
   * mensagem avisa que o que a pessoa acabou de salvar não foi gravado.
   */
  expirou: (perdeuAlteracao?: boolean) => void;
  /** Relê quem está logado (ex.: a pessoa mudou o próprio papel). */
  atualizar: () => Promise<void>;
};

const SessaoContext = createContext<Sessao | null>(null);

export function SessaoProvider({ children }: { children: ReactNode }) {
  const [usuario, setUsuario] = useState<Usuario | null | undefined>(undefined);
  const [aviso, setAviso] = useState("");

  useEffect(() => {
    acessoApi
      .sessao()
      .then(setUsuario)
      .catch((e) => {
        setUsuario(null);
        if (e instanceof ErroApi && e.status !== 401) setAviso(e.message);
      });
  }, []);

  const entrou = useCallback((u: Usuario) => {
    setAviso("");
    setUsuario(u);
  }, []);

  const sair = useCallback(async () => {
    try {
      await acessoApi.sair();
    } finally {
      setAviso("");
      setUsuario(null);
    }
  }, []);

  const expirou = useCallback((perdeuAlteracao = false) => {
    setAviso(
      perdeuAlteracao
        ? "Sua sessão terminou e a última alteração NÃO foi salva. Entre de novo e refaça."
        : "Sua sessão terminou. Entre de novo.",
    );
    setUsuario(null);
  }, []);

  const atualizar = useCallback(async () => {
    try {
      setUsuario(await acessoApi.sessao());
    } catch (e) {
      if (e instanceof ErroApi && e.status === 401) expirou();
    }
  }, [expirou]);

  const valor = useMemo(
    () => ({ usuario, aviso, entrou, sair, expirou, atualizar }),
    [usuario, aviso, entrou, sair, expirou, atualizar],
  );
  return <SessaoContext.Provider value={valor}>{children}</SessaoContext.Provider>;
}

export function useSessao() {
  const ctx = useContext(SessaoContext);
  if (!ctx) throw new Error("useSessao fora do SessaoProvider");
  return ctx;
}
