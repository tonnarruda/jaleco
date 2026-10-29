import type { MetaVendedor } from "../types";

/** Meta que valia no mês: a última vigência que começou até ele. 0 = sem meta. */
export function metaDoMes(metas: MetaVendedor[], ym: string): number {
  let valor = 0;
  for (const m of [...metas].sort((a, b) => a.desde.localeCompare(b.desde))) {
    if (m.desde <= ym) valor = m.valor;
  }
  return valor;
}

/** Grava a meta a partir do mês sem apagar as vigências anteriores. */
export function registrarMeta(metas: MetaVendedor[], desde: string, valor: number): MetaVendedor[] {
  if (metaDoMes(metas, desde) === valor && metas.some((m) => m.desde <= desde)) return metas;
  if (!metas.length && valor === 0) return metas;
  return [...metas.filter((m) => m.desde !== desde), { desde, valor }].sort((a, b) =>
    a.desde.localeCompare(b.desde),
  );
}
