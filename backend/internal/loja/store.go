package loja

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// schema roda a cada início; IF NOT EXISTS mantém os dados. Mudança de coluna
// em tabela que já existe vai precisar de uma ferramenta de migração.
const schema = `
CREATE TABLE IF NOT EXISTS clientes (
	id TEXT PRIMARY KEY,
	nome TEXT NOT NULL,
	whatsapp TEXT NOT NULL DEFAULT '',
	email TEXT NOT NULL DEFAULT '',
	profissao TEXT NOT NULL DEFAULT '',
	faculdade TEXT NOT NULL DEFAULT '',
	instagram TEXT NOT NULL DEFAULT '',
	cidade TEXT NOT NULL DEFAULT '',
	aniversario TEXT NOT NULL DEFAULT '',
	tamanho_habitual TEXT NOT NULL DEFAULT '',
	como_conheceu TEXT NOT NULL DEFAULT '',
	endereco TEXT NOT NULL DEFAULT '',
	observacoes TEXT NOT NULL DEFAULT ''
);

-- O número vem de uma sequência: com MAX + 1, duas vendas simultâneas pegariam o mesmo.
-- As peças só existem dentro do pedido e sempre viajam com ele: ficam em JSONB.
CREATE TABLE IF NOT EXISTS pedidos (
	id TEXT PRIMARY KEY,
	numero INTEGER GENERATED ALWAYS AS IDENTITY UNIQUE,
	cliente_id TEXT NOT NULL DEFAULT '',
	vendedor_id TEXT NOT NULL DEFAULT '',
	canal TEXT NOT NULL DEFAULT '',
	data_pedido TEXT NOT NULL DEFAULT '',
	pecas JSONB NOT NULL DEFAULT '[]',
	etapa TEXT NOT NULL,
	prazo_entrega TEXT NOT NULL DEFAULT '',
	codigo_rastreio TEXT NOT NULL DEFAULT '',
	observacoes TEXT NOT NULL DEFAULT '',
	forma_pagamento TEXT NOT NULL DEFAULT '',
	situacao_pagamento TEXT NOT NULL DEFAULT '',
	desconto NUMERIC(12,2) NOT NULL DEFAULT 0,
	frete NUMERIC(12,2) NOT NULL DEFAULT 0,
	valor_pago NUMERIC(12,2) NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS estoque (
	id TEXT PRIMARY KEY,
	chave TEXT NOT NULL UNIQUE,
	modelo TEXT NOT NULL,
	cor TEXT NOT NULL,
	tamanho TEXT NOT NULL,
	quantidade INTEGER NOT NULL CHECK (quantidade >= 0),
	minimo INTEGER NOT NULL DEFAULT 0 CHECK (minimo >= 0)
);
ALTER TABLE estoque ADD COLUMN IF NOT EXISTS preco_venda NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (preco_venda >= 0);

CREATE TABLE IF NOT EXISTS vendedores (
	id TEXT PRIMARY KEY,
	nome TEXT NOT NULL,
	whatsapp TEXT NOT NULL DEFAULT '',
	papel TEXT NOT NULL CHECK (papel IN ('Vendedor(a)', 'Gerente')),
	acesso TEXT NOT NULL CHECK (acesso IN ('Ativo', 'Inativo')),
	comissao NUMERIC(5,2) NOT NULL DEFAULT 0 CHECK (comissao BETWEEN 0 AND 100)
);

-- Modelos que aparecem nas listas. Cadastrado aqui, o modelo não some quando o estoque dele zera ou sai.
-- O índice ignora maiúsculas e espaços nas pontas: "jaleco slim " é o mesmo que "Jaleco Slim".
CREATE TABLE IF NOT EXISTS modelos (
	id SERIAL PRIMARY KEY,
	nome TEXT NOT NULL CHECK (btrim(nome) <> '')
);
CREATE UNIQUE INDEX IF NOT EXISTS modelos_nome_unico ON modelos (lower(btrim(nome)));

-- Login: a pessoa da equipe entra com e-mail e senha (guardada só como hash bcrypt).
ALTER TABLE vendedores ADD COLUMN IF NOT EXISTS email TEXT;
ALTER TABLE vendedores ADD COLUMN IF NOT EXISTS senha_hash TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS vendedores_email_unico ON vendedores (lower(email)) WHERE email IS NOT NULL;
ALTER TABLE vendedores DROP CONSTRAINT IF EXISTS vendedores_papel_check;
ALTER TABLE vendedores ADD CONSTRAINT vendedores_papel_check CHECK (papel IN ('Vendedor(a)', 'Gerente', 'Administrador'));

-- Sessões e links de uso único guardam só o hash do token.
CREATE TABLE IF NOT EXISTS sessoes (
	token_hash TEXT PRIMARY KEY,
	vendedor_id TEXT NOT NULL REFERENCES vendedores(id) ON DELETE CASCADE,
	criada_em TIMESTAMPTZ NOT NULL DEFAULT now(),
	expira_em TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS convites (
	token_hash TEXT PRIMARY KEY,
	tipo TEXT NOT NULL CHECK (tipo IN ('primeiro-admin', 'convite', 'redefinicao')),
	vendedor_id TEXT REFERENCES vendedores(id) ON DELETE CASCADE,
	criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
	expira_em TIMESTAMPTZ NOT NULL,
	usado_em TIMESTAMPTZ
);

-- Histórico de metas: o código só insere ou corrige uma vigência, nunca apaga.
CREATE TABLE IF NOT EXISTS vendedor_metas (
	vendedor_id TEXT NOT NULL REFERENCES vendedores(id),
	desde TEXT NOT NULL CHECK (desde ~ '^\d{4}-\d{2}$'),
	valor NUMERIC(12,2) NOT NULL CHECK (valor >= 0),
	PRIMARY KEY (vendedor_id, desde)
);
`

type Store struct {
	db *sql.DB
	// agora é a hora de referência para validade de sessões e links. Vem do Go,
	// não do now() do banco, para os testes poderem avançar o relógio.
	agora func() time.Time
}

// DefinirRelogio troca a hora de referência. Só para testes de vencimento.
func (s *Store) DefinirRelogio(agora func() time.Time) { s.agora = agora }

// Open conecta no Postgres (URL no formato postgres://usuario:senha@host:porta/banco)
// e garante as tabelas.
func Open(ctx context.Context, url string) (*Store, error) {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("conectando no Postgres: %w", err)
	}
	if _, err := db.ExecContext(ctx, schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("criando tabelas: %w", err)
	}
	s := &Store{db: db, agora: time.Now}
	if err := s.semearModelos(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("cadastrando modelos padrão: %w", err)
	}
	return s, nil
}

// semearModelos garante os modelos padrão, em ordem, e registra os que já
// estavam no estoque antes de a tabela existir, para nenhum sumir das listas.
func (s *Store) semearModelos(ctx context.Context) error {
	for _, nome := range ModelosPadrao {
		if _, err := s.db.ExecContext(ctx, `INSERT INTO modelos (nome) VALUES ($1) ON CONFLICT DO NOTHING`, nome); err != nil {
			return err
		}
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO modelos (nome)
		SELECT modelo FROM estoque GROUP BY modelo ORDER BY min(modelo) ON CONFLICT DO NOTHING`)
	return err
}

func (s *Store) Close() error { return s.db.Close() }

// queryer é o que *sql.DB e *sql.Tx têm em comum.
type queryer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (s *Store) emTransacao(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// ---------- leitura ----------

// Estado devolve tudo, sem filtro de papel (uso interno e testes).
func (s *Store) Estado(ctx context.Context) (Estado, error) {
	return s.estado(ctx, "")
}

// EstadoPara devolve o que o papel pode ver. Vendedor recebe só os próprios
// pedidos e o próprio cadastro; clientes, estoque e modelos são de todos.
func (s *Store) EstadoPara(ctx context.Context, u Usuario) (Estado, error) {
	if u.Gestor() {
		return s.estado(ctx, "")
	}
	return s.estado(ctx, u.ID)
}

// estado com vendedorID != "" restringe pedidos e equipe àquela pessoa.
func (s *Store) estado(ctx context.Context, vendedorID string) (Estado, error) {
	e := Estado{}
	var err error
	if e.Clientes, err = listarClientes(ctx, s.db); err != nil {
		return e, err
	}
	if e.Pedidos, err = listarPedidos(ctx, s.db, vendedorID); err != nil {
		return e, err
	}
	if e.Estoque, err = listarEstoque(ctx, s.db); err != nil {
		return e, err
	}
	if e.Vendedores, err = listarVendedores(ctx, s.db, vendedorID); err != nil {
		return e, err
	}
	if e.Modelos, err = listarModelos(ctx, s.db); err != nil {
		return e, err
	}
	return e, nil
}

func listarClientes(ctx context.Context, q queryer) ([]Cliente, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, nome, whatsapp, email, profissao, faculdade, instagram,
		cidade, aniversario, tamanho_habitual, como_conheceu, endereco, observacoes FROM clientes ORDER BY nome`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Cliente{}
	for rows.Next() {
		var c Cliente
		if err := rows.Scan(&c.ID, &c.Nome, &c.Whatsapp, &c.Email, &c.Profissao, &c.Faculdade, &c.Instagram,
			&c.Cidade, &c.Aniversario, &c.TamanhoHabitual, &c.ComoConheceu, &c.Endereco, &c.Observacoes); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

const colunasPedido = `id, numero, cliente_id, vendedor_id, canal, data_pedido, pecas, etapa, prazo_entrega,
	codigo_rastreio, observacoes, forma_pagamento, situacao_pagamento, desconto, frete, valor_pago`

type scanner interface{ Scan(dest ...any) error }

func scanPedido(r scanner) (Pedido, error) {
	var p Pedido
	var pecas []byte
	err := r.Scan(&p.ID, &p.Numero, &p.ClienteID, &p.VendedorID, &p.Canal, &p.DataPedido, &pecas, &p.Etapa,
		&p.PrazoEntrega, &p.CodigoRastreio, &p.Observacoes, &p.FormaPagamento, &p.SituacaoPagamento,
		&p.Desconto, &p.Frete, &p.ValorPago)
	if err != nil {
		return p, err
	}
	if err := json.Unmarshal(pecas, &p.Pecas); err != nil {
		return p, fmt.Errorf("peças do pedido %d: %w", p.Numero, err)
	}
	return p, nil
}

func listarPedidos(ctx context.Context, q queryer, vendedorID string) ([]Pedido, error) {
	rows, err := q.QueryContext(ctx, `SELECT `+colunasPedido+` FROM pedidos
		WHERE $1 = '' OR vendedor_id = $1 ORDER BY numero`, vendedorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Pedido{}
	for rows.Next() {
		p, err := scanPedido(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func listarModelos(ctx context.Context, q queryer) ([]string, error) {
	rows, err := q.QueryContext(ctx, `SELECT nome FROM modelos ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var nome string
		if err := rows.Scan(&nome); err != nil {
			return nil, err
		}
		out = append(out, nome)
	}
	return out, rows.Err()
}

func listarEstoque(ctx context.Context, q queryer) ([]ItemEstoque, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, modelo, cor, tamanho, quantidade, minimo, preco_venda FROM estoque ORDER BY modelo, cor, tamanho`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ItemEstoque{}
	for rows.Next() {
		var i ItemEstoque
		if err := rows.Scan(&i.ID, &i.Modelo, &i.Cor, &i.Tamanho, &i.Quantidade, &i.Minimo, &i.PrecoVenda); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func listarVendedores(ctx context.Context, q queryer, vendedorID string) ([]Vendedor, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, nome, whatsapp, papel, acesso, comissao, coalesce(email, ''),
		senha_hash IS NOT NULL FROM vendedores WHERE $1 = '' OR id = $1 ORDER BY nome`, vendedorID)
	if err != nil {
		return nil, err
	}
	out := []Vendedor{}
	idx := map[string]int{}
	for rows.Next() {
		v := Vendedor{Metas: []Meta{}}
		if err := rows.Scan(&v.ID, &v.Nome, &v.Whatsapp, &v.Papel, &v.Acesso, &v.Comissao, &v.Email, &v.PossuiLogin); err != nil {
			rows.Close()
			return nil, err
		}
		idx[v.ID] = len(out)
		out = append(out, v)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	mrows, err := q.QueryContext(ctx, `SELECT vendedor_id, desde, valor FROM vendedor_metas ORDER BY desde`)
	if err != nil {
		return nil, err
	}
	defer mrows.Close()
	for mrows.Next() {
		var id string
		var m Meta
		if err := mrows.Scan(&id, &m.Desde, &m.Valor); err != nil {
			return nil, err
		}
		if i, ok := idx[id]; ok {
			out[i].Metas = append(out[i].Metas, m)
		}
	}
	return out, mrows.Err()
}

// ---------- clientes ----------

func (s *Store) SalvarCliente(ctx context.Context, c Cliente) error {
	if c.ID == "" {
		return ErrValidacao{"Cliente sem id."}
	}
	if c.Nome == "" {
		return ErrValidacao{"Informe o nome do cliente."}
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO clientes (id, nome, whatsapp, email, profissao, faculdade, instagram,
		cidade, aniversario, tamanho_habitual, como_conheceu, endereco, observacoes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (id) DO UPDATE SET nome = excluded.nome, whatsapp = excluded.whatsapp, email = excluded.email,
		profissao = excluded.profissao, faculdade = excluded.faculdade, instagram = excluded.instagram,
		cidade = excluded.cidade, aniversario = excluded.aniversario, tamanho_habitual = excluded.tamanho_habitual,
		como_conheceu = excluded.como_conheceu, endereco = excluded.endereco, observacoes = excluded.observacoes`,
		c.ID, c.Nome, c.Whatsapp, c.Email, c.Profissao, c.Faculdade, c.Instagram,
		c.Cidade, c.Aniversario, c.TamanhoHabitual, c.ComoConheceu, c.Endereco, c.Observacoes)
	return err
}

// numerosDePedidos lista, em ordem, os números dos pedidos que a consulta devolve.
func numerosDePedidos(ctx context.Context, tx *sql.Tx, query string, args ...any) ([]int, error) {
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int
	for rows.Next() {
		var n int
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// pedidosDe vira "pedido #3" ou "pedidos #2, #5".
func pedidosDe(numeros []int) string {
	if len(numeros) == 1 {
		return "pedido " + listaPedidos(numeros)
	}
	return "pedidos " + listaPedidos(numeros)
}

// noPedido vira "no pedido #3" ou "nos pedidos #2, #5".
func noPedido(numeros []int) string {
	if len(numeros) == 1 {
		return "no " + pedidosDe(numeros)
	}
	return "nos " + pedidosDe(numeros)
}

// listaPedidos vira "#2, #5"; mais de cinco números resumem o resto.
func listaPedidos(numeros []int) string {
	partes := []string{}
	for i, n := range numeros {
		if i == 5 {
			partes = append(partes, fmt.Sprintf("e mais %d", len(numeros)-5))
			break
		}
		partes = append(partes, fmt.Sprintf("#%d", n))
	}
	return strings.Join(partes, ", ")
}

// ExcluirCliente recusa cliente com pedido: o pedido perderia de quem é.
func (s *Store) ExcluirCliente(ctx context.Context, id string) error {
	return s.emTransacao(ctx, func(tx *sql.Tx) error {
		numeros, err := numerosDePedidos(ctx, tx, `SELECT numero FROM pedidos WHERE cliente_id = $1 ORDER BY numero`, id)
		if err != nil {
			return err
		}
		if len(numeros) > 0 {
			return ErrValidacao{fmt.Sprintf("Este cliente está %s. Exclua os pedidos antes.", noPedido(numeros))}
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM clientes WHERE id = $1`, id)
		return err
	})
}

// ---------- pedidos ----------

// movimentar soma (sinal 1) ou baixa (sinal -1) as peças no estoque, sem passar de zero.
// Variação sem cadastro é ignorada, como no frontend.
func movimentar(ctx context.Context, tx *sql.Tx, pecas []Peca, sinal int) error {
	for _, p := range pecas {
		_, err := tx.ExecContext(ctx, `UPDATE estoque SET quantidade = GREATEST(0, quantidade + $1) WHERE chave = $2`,
			sinal*p.Quantidade, chaveVariacao(p.Modelo, p.Cor, p.Tamanho))
		if err != nil {
			return err
		}
	}
	return nil
}

func validarPedido(p Pedido) error {
	if p.ID == "" {
		return ErrValidacao{"Pedido sem id."}
	}
	if p.Etapa == "" {
		return ErrValidacao{"Informe a etapa do pedido."}
	}
	for _, peca := range p.Pecas {
		if peca.Quantidade < 0 || peca.PrecoUnitario < 0 {
			return ErrValidacao{"Quantidade e preço das peças não podem ser negativos."}
		}
	}
	if p.Desconto < 0 || p.Frete < 0 || p.ValorPago < 0 {
		return ErrValidacao{"Desconto, frete e valor pago não podem ser negativos."}
	}
	return nil
}

// SalvarPedido cria ou atualiza o pedido. O número é sempre do banco, e o
// estoque devolve as peças da versão anterior antes de baixar as da nova.
func (s *Store) SalvarPedido(ctx context.Context, p Pedido) (Pedido, error) {
	return s.salvarPedido(ctx, p, "")
}

// SalvarPedidoComo aplica o papel: vendedor só registra em nome próprio e só
// mexe nos pedidos dele (o de outra pessoa responde como inexistente).
func (s *Store) SalvarPedidoComo(ctx context.Context, u Usuario, p Pedido) (Pedido, error) {
	if u.Gestor() {
		return s.salvarPedido(ctx, p, "")
	}
	p.VendedorID = u.ID
	return s.salvarPedido(ctx, p, u.ID)
}

func (s *Store) salvarPedido(ctx context.Context, p Pedido, dono string) (Pedido, error) {
	if p.Pecas == nil {
		p.Pecas = []Peca{}
	}
	if err := validarPedido(p); err != nil {
		return p, err
	}
	pecas, err := json.Marshal(p.Pecas)
	if err != nil {
		return p, err
	}
	err = s.emTransacao(ctx, func(tx *sql.Tx) error {
		// FOR UPDATE: duas edições do mesmo pedido em paralelo não devolvem estoque em dobro.
		existente, err := scanPedido(tx.QueryRowContext(ctx,
			`SELECT `+colunasPedido+` FROM pedidos WHERE id = $1 FOR UPDATE`, p.ID))
		novo := errors.Is(err, sql.ErrNoRows)
		if err != nil && !novo {
			return err
		}
		if !novo && dono != "" && existente.VendedorID != dono {
			return ErrNaoEncontrado
		}
		if !novo && consomeEstoque(existente.Etapa) {
			if err := movimentar(ctx, tx, existente.Pecas, 1); err != nil {
				return err
			}
		}
		if consomeEstoque(p.Etapa) {
			if err := movimentar(ctx, tx, p.Pecas, -1); err != nil {
				return err
			}
		}
		args := []any{p.ID, p.ClienteID, p.VendedorID, p.Canal, p.DataPedido, string(pecas), p.Etapa, p.PrazoEntrega,
			p.CodigoRastreio, p.Observacoes, p.FormaPagamento, p.SituacaoPagamento, p.Desconto, p.Frete, p.ValorPago}
		if novo {
			return tx.QueryRowContext(ctx, `INSERT INTO pedidos (id, cliente_id, vendedor_id, canal, data_pedido, pecas,
				etapa, prazo_entrega, codigo_rastreio, observacoes, forma_pagamento, situacao_pagamento, desconto, frete, valor_pago)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15) RETURNING numero`, args...).
				Scan(&p.Numero)
		}
		p.Numero = existente.Numero
		_, err = tx.ExecContext(ctx, `UPDATE pedidos SET cliente_id = $2, vendedor_id = $3, canal = $4, data_pedido = $5,
			pecas = $6, etapa = $7, prazo_entrega = $8, codigo_rastreio = $9, observacoes = $10, forma_pagamento = $11,
			situacao_pagamento = $12, desconto = $13, frete = $14, valor_pago = $15 WHERE id = $1`, args...)
		return err
	})
	return p, err
}

func (s *Store) ExcluirPedido(ctx context.Context, id string) error {
	return s.emTransacao(ctx, func(tx *sql.Tx) error {
		p, err := scanPedido(tx.QueryRowContext(ctx, `SELECT `+colunasPedido+` FROM pedidos WHERE id = $1 FOR UPDATE`, id))
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if consomeEstoque(p.Etapa) {
			if err := movimentar(ctx, tx, p.Pecas, 1); err != nil {
				return err
			}
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM pedidos WHERE id = $1`, id)
		return err
	})
}

// ---------- estoque ----------

// EntradaEstoque soma na variação existente ou cria uma nova. Preço 0 na entrada
// mantém o preço que a variação já tinha.
func (s *Store) EntradaEstoque(ctx context.Context, i ItemEstoque) error {
	if i.ID == "" {
		return ErrValidacao{"Item de estoque sem id."}
	}
	if i.Modelo == "" || i.Cor == "" || i.Tamanho == "" {
		return ErrValidacao{"Informe modelo, cor e tamanho."}
	}
	if i.Quantidade < 0 || i.Minimo < 0 || i.PrecoVenda < 0 {
		return ErrValidacao{"Quantidade, mínimo e preço não podem ser negativos."}
	}
	return s.emTransacao(ctx, func(tx *sql.Tx) error {
		// Modelo novo lançado na entrada passa a existir na lista, mesmo que o estoque zere depois.
		if _, err := tx.ExecContext(ctx, `INSERT INTO modelos (nome) VALUES ($1) ON CONFLICT DO NOTHING`, i.Modelo); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO estoque (id, chave, modelo, cor, tamanho, quantidade, minimo, preco_venda)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (chave) DO UPDATE SET quantidade = estoque.quantidade + excluded.quantidade, minimo = excluded.minimo,
			preco_venda = CASE WHEN excluded.preco_venda > 0 THEN excluded.preco_venda ELSE estoque.preco_venda END`,
			i.ID, chaveVariacao(i.Modelo, i.Cor, i.Tamanho), i.Modelo, i.Cor, i.Tamanho, i.Quantidade, i.Minimo, i.PrecoVenda)
		return err
	})
}

// Ajuste é a correção de uma variação: contagem, alerta e preço.
type Ajuste struct {
	ID         string  `json:"id"`
	Quantidade int     `json:"quantidade"`
	Minimo     int     `json:"minimo"`
	PrecoVenda float64 `json:"precoVenda"`
}

// AjustarEstoque grava os ajustes numa transação: os tamanhos de um produto
// editados juntos são salvos todos ou nenhum.
func (s *Store) AjustarEstoque(ctx context.Context, ajustes ...Ajuste) error {
	if len(ajustes) == 0 {
		return ErrValidacao{"Nenhum item de estoque informado."}
	}
	for _, a := range ajustes {
		if a.Quantidade < 0 || a.Minimo < 0 || a.PrecoVenda < 0 {
			return ErrValidacao{"Quantidade, mínimo e preço não podem ser negativos."}
		}
	}
	return s.emTransacao(ctx, func(tx *sql.Tx) error {
		for _, a := range ajustes {
			res, err := tx.ExecContext(ctx, `UPDATE estoque SET quantidade = $1, minimo = $2, preco_venda = $3 WHERE id = $4`,
				a.Quantidade, a.Minimo, a.PrecoVenda, a.ID)
			if err != nil {
				return err
			}
			if n, _ := res.RowsAffected(); n == 0 {
				return ErrNaoEncontrado
			}
		}
		return nil
	})
}

// RemoverEstoque apaga as variações de uma vez: um produto com todos os
// tamanhos sai inteiro ou não sai. Pedidos não mudam (guardam a peça como texto).
func (s *Store) RemoverEstoque(ctx context.Context, ids ...string) error {
	if len(ids) == 0 {
		return ErrValidacao{"Nenhum item de estoque informado."}
	}
	return s.emTransacao(ctx, func(tx *sql.Tx) error {
		// Variação que aparece em qualquer pedido (inclusive entregue ou cancelado) fica:
		// o histórico de vendas aponta para ela. A chave repete chaveVariacao em SQL.
		rows, err := tx.QueryContext(ctx, `SELECT DISTINCT e.tamanho, p.numero
			FROM estoque e
			JOIN pedidos p ON EXISTS (
				SELECT 1 FROM jsonb_array_elements(p.pecas) x
				WHERE lower(btrim(x->>'modelo')) || '|' || lower(btrim(x->>'cor')) || '|' || lower(btrim(x->>'tamanho')) = e.chave
			)
			WHERE e.id = ANY($1)
			ORDER BY e.tamanho, p.numero`, ids)
		if err != nil {
			return err
		}
		emUso := map[string][]int{}
		var tamanhos []string
		for rows.Next() {
			var tam string
			var numero int
			if err := rows.Scan(&tam, &numero); err != nil {
				rows.Close()
				return err
			}
			if _, ok := emUso[tam]; !ok {
				tamanhos = append(tamanhos, tam)
			}
			emUso[tam] = append(emUso[tam], numero)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if len(tamanhos) > 0 {
			partes := make([]string, len(tamanhos))
			for i, t := range tamanhos {
				partes[i] = fmt.Sprintf("%s (%s)", t, pedidosDe(emUso[t]))
			}
			return ErrValidacao{"Não dá para excluir tamanho que está em pedido: " + strings.Join(partes, "; ") + "."}
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM estoque WHERE id = ANY($1)`, ids)
		return err
	})
}

// ---------- vendedores ----------

var mesValido = regexp.MustCompile(`^\d{4}-\d{2}$`)

func validarVendedor(v Vendedor) error {
	if v.ID == "" || v.ID == "admin" {
		return ErrValidacao{"Vendedor com id inválido."}
	}
	if v.Nome == "" {
		return ErrValidacao{"Informe o nome que vai aparecer nas vendas."}
	}
	if v.Papel != PapelVendedor && v.Papel != PapelGerente && v.Papel != PapelAdministrador {
		return ErrValidacao{"Papel deve ser Vendedor(a), Gerente ou Administrador."}
	}
	if v.Email != "" {
		if err := validarEmail(v.Email); err != nil {
			return err
		}
	}
	if v.Acesso != "Ativo" && v.Acesso != "Inativo" {
		return ErrValidacao{"Acesso deve ser Ativo ou Inativo."}
	}
	if v.Comissao < 0 || v.Comissao > 100 {
		return ErrValidacao{"Informe a comissão entre 0 e 100%."}
	}
	for _, m := range v.Metas {
		if !mesValido.MatchString(m.Desde) || m.Valor < 0 {
			return ErrValidacao{"Meta inválida: informe o mês (AAAA-MM) e um valor maior ou igual a zero."}
		}
	}
	return nil
}

// SalvarVendedor grava o cadastro e as metas enviadas. Vigência que não veio na
// requisição continua no banco: o histórico não depende do que a tela manda.
func (s *Store) SalvarVendedor(ctx context.Context, v Vendedor) error {
	v.Email = normalizarEmail(v.Email)
	if err := validarVendedor(v); err != nil {
		return err
	}
	return s.emTransacao(ctx, func(tx *sql.Tx) error {
		continuaAdmin := v.Papel == PapelAdministrador && v.Acesso == "Ativo"
		if err := garantirOutroAdmin(ctx, tx, v.ID, continuaAdmin); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO vendedores (id, nome, whatsapp, papel, acesso, comissao, email)
			VALUES ($1, $2, $3, $4, $5, $6, nullif($7, ''))
			ON CONFLICT (id) DO UPDATE SET nome = excluded.nome, whatsapp = excluded.whatsapp,
			papel = excluded.papel, acesso = excluded.acesso, comissao = excluded.comissao, email = excluded.email`,
			v.ID, v.Nome, v.Whatsapp, v.Papel, v.Acesso, v.Comissao, v.Email)
		if err != nil {
			return traduzirErroEmail(err)
		}
		if v.Acesso != "Ativo" {
			// Inativo sai na hora: as sessões abertas e os links pendentes deixam de valer.
			if _, err := tx.ExecContext(ctx, `DELETE FROM sessoes WHERE vendedor_id = $1`, v.ID); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `DELETE FROM convites WHERE vendedor_id = $1 AND usado_em IS NULL`, v.ID); err != nil {
				return err
			}
		}
		for _, m := range v.Metas {
			_, err := tx.ExecContext(ctx, `INSERT INTO vendedor_metas (vendedor_id, desde, valor) VALUES ($1, $2, $3)
				ON CONFLICT (vendedor_id, desde) DO UPDATE SET valor = excluded.valor`, v.ID, m.Desde, m.Valor)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// ExcluirVendedor apaga o vendedor e as metas dele. Vendedor com pedido fica:
// as vendas precisam continuar com nome; para tirar o acesso existe Inativo.
func (s *Store) ExcluirVendedor(ctx context.Context, id string) error {
	return s.emTransacao(ctx, func(tx *sql.Tx) error {
		numeros, err := numerosDePedidos(ctx, tx, `SELECT numero FROM pedidos WHERE vendedor_id = $1 ORDER BY numero`, id)
		if err != nil {
			return err
		}
		if len(numeros) > 0 {
			return ErrValidacao{fmt.Sprintf(
				"Este vendedor está %s e não pode ser excluído. Para tirar o acesso, marque como Inativo.",
				noPedido(numeros))}
		}
		if err := garantirOutroAdmin(ctx, tx, id, false); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM vendedor_metas WHERE vendedor_id = $1`, id); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM vendedores WHERE id = $1`, id)
		return err
	})
}

// garantirOutroAdmin impede que a equipe fique sem nenhum Administrador ativo:
// quem hoje é admin ativo só deixa de ser se houver outro.
func garantirOutroAdmin(ctx context.Context, tx *sql.Tx, id string, continuaAdmin bool) error {
	if continuaAdmin {
		return nil
	}
	var eraAdmin bool
	err := tx.QueryRowContext(ctx, `SELECT papel = $2 AND acesso = 'Ativo' FROM vendedores WHERE id = $1 FOR UPDATE`,
		id, PapelAdministrador).Scan(&eraAdmin)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !eraAdmin) {
		return nil
	}
	if err != nil {
		return err
	}
	var outros int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM vendedores WHERE papel = $1 AND acesso = 'Ativo' AND id <> $2`,
		PapelAdministrador, id).Scan(&outros); err != nil {
		return err
	}
	if outros == 0 {
		return ErrValidacao{"O sistema precisa de pelo menos um Administrador ativo. Cadastre outro antes de mudar este."}
	}
	return nil
}

// traduzirErroEmail transforma o e-mail repetido (índice único) em mensagem para a tela.
func traduzirErroEmail(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" && pg.ConstraintName == "vendedores_email_unico" {
		return ErrValidacao{"Já existe alguém da equipe com esse e-mail."}
	}
	return err
}
