package loja

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Papéis de quem usa o sistema.
const (
	PapelAdministrador = "Administrador"
	PapelGerente       = "Gerente"
	PapelVendedor      = "Vendedor(a)"
)

const (
	DuracaoSessao  = 30 * 24 * time.Hour
	DuracaoConvite = 72 * time.Hour

	// Tipos de link de uso único.
	ConvitePrimeiroAdmin = "primeiro-admin" // criado pelo servidor quando não há administrador
	ConviteNovo          = "convite"        // pessoa da equipe que ainda não tem senha
	ConviteRedefinicao   = "redefinicao"    // pessoa que esqueceu a senha

	senhaMinima = 8
	senhaMaxima = 72 // limite do bcrypt, em bytes
)

var (
	// ErrNaoAutenticado: sessão ausente, vencida, ou de alguém que ficou inativo.
	ErrNaoAutenticado = errors.New("não autenticado")
	// ErrCredenciais não diz se o erro foi no e-mail ou na senha, de propósito.
	ErrCredenciais     = ErrValidacao{"E-mail ou senha incorretos."}
	ErrConviteInvalido = ErrValidacao{"Este link não vale mais: já foi usado ou venceu. Peça um novo ao administrador."}
)

type Usuario struct {
	ID    string `json:"id"`
	Nome  string `json:"nome"`
	Papel string `json:"papel"`
	Email string `json:"email"`
}

func (u Usuario) Gestor() bool { return u.Papel == PapelAdministrador || u.Papel == PapelGerente }

// novoToken devolve o valor que vai para a pessoa (cookie ou link) e o hash que
// fica no banco. Quem lê o banco não consegue entrar com o que encontra lá.
func novoToken() (token, hash string) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err) // sem fonte de aleatoriedade não há como seguir com segurança
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, hashToken(token)
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func novoID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func normalizarEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func validarEmail(email string) error {
	at := strings.LastIndex(email, "@")
	if at < 1 || !strings.Contains(email[at:], ".") || strings.ContainsAny(email, " \t") {
		return ErrValidacao{"Informe um e-mail válido."}
	}
	return nil
}

func validarSenha(senha string) error {
	if len(senha) < senhaMinima {
		return ErrValidacao{"A senha precisa ter pelo menos 8 caracteres."}
	}
	if len(senha) > senhaMaxima {
		return ErrValidacao{"A senha pode ter no máximo 72 caracteres."}
	}
	return nil
}

// hashFalso é comparado quando o e-mail não existe, para o tempo de resposta não
// revelar quais e-mails têm conta.
var hashFalso, _ = bcrypt.GenerateFromPassword([]byte("senha-que-ninguem-usa"), bcrypt.DefaultCost)

// ---------- sessão ----------

func criarSessao(ctx context.Context, q queryer, vendedorID string, agora time.Time) (string, error) {
	token, hash := novoToken()
	_, err := q.ExecContext(ctx, `INSERT INTO sessoes (token_hash, vendedor_id, expira_em) VALUES ($1, $2, $3)`,
		hash, vendedorID, agora.Add(DuracaoSessao))
	return token, err
}

// CriarSessao abre uma sessão para a pessoa (usado após login e convite, e nos testes).
func (s *Store) CriarSessao(ctx context.Context, vendedorID string) (string, error) {
	return criarSessao(ctx, s.db, vendedorID, s.agora())
}

// UsuarioDaSessao resolve o cookie. Inativo perde o acesso na hora, mesmo com sessão aberta.
func (s *Store) UsuarioDaSessao(ctx context.Context, token string) (Usuario, error) {
	var u Usuario
	if token == "" {
		return u, ErrNaoAutenticado
	}
	err := s.db.QueryRowContext(ctx, `SELECT v.id, v.nome, v.papel, coalesce(v.email, '')
		FROM sessoes s JOIN vendedores v ON v.id = s.vendedor_id
		WHERE s.token_hash = $1 AND s.expira_em > $2 AND v.acesso = 'Ativo'`, hashToken(token), s.agora()).
		Scan(&u.ID, &u.Nome, &u.Papel, &u.Email)
	if errors.Is(err, sql.ErrNoRows) {
		return u, ErrNaoAutenticado
	}
	return u, err
}

func (s *Store) EncerrarSessao(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessoes WHERE token_hash = $1`, hashToken(token))
	return err
}

// Entrar confere e-mail e senha e abre uma sessão.
func (s *Store) Entrar(ctx context.Context, email, senha string) (Usuario, string, error) {
	var u Usuario
	var hash string
	err := s.db.QueryRowContext(ctx, `SELECT id, nome, papel, email, senha_hash FROM vendedores
		WHERE lower(email) = $1 AND senha_hash IS NOT NULL AND acesso = 'Ativo'`, normalizarEmail(email)).
		Scan(&u.ID, &u.Nome, &u.Papel, &u.Email, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		bcrypt.CompareHashAndPassword(hashFalso, []byte(senha))
		return u, "", ErrCredenciais
	}
	if err != nil {
		return u, "", err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(senha)) != nil {
		return u, "", ErrCredenciais
	}
	token, err := s.CriarSessao(ctx, u.ID)
	return u, token, err
}

// ---------- convites ----------

// TemAdministrador diz se o sistema já foi configurado. Enquanto não, a tela de
// entrada oferece criar o primeiro administrador.
func (s *Store) TemAdministrador(ctx context.Context) (bool, error) {
	var existe bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM vendedores WHERE papel = $1)`, PapelAdministrador).
		Scan(&existe)
	return existe, err
}

// GarantirConvitePrimeiroAdmin gera o link de primeiro acesso enquanto não houver
// administrador. Links gerados antes continuam valendo (um clique em outro
// aparelho não derruba quem está preenchendo); o primeiro a concluir vira admin
// e os outros morrem em buscarConvite. Com administrador, devolve "".
func (s *Store) GarantirConvitePrimeiroAdmin(ctx context.Context) (string, error) {
	var token string
	err := s.emTransacao(ctx, func(tx *sql.Tx) error {
		var existe bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM vendedores WHERE papel = $1)`,
			PapelAdministrador).Scan(&existe); err != nil {
			return err
		}
		if existe {
			return nil
		}
		// Só a faxina dos vencidos: cada clique cria uma linha.
		if _, err := tx.ExecContext(ctx, `DELETE FROM convites WHERE tipo = $1 AND usado_em IS NULL AND expira_em <= $2`,
			ConvitePrimeiroAdmin, s.agora()); err != nil {
			return err
		}
		var hash string
		token, hash = novoToken()
		_, err := tx.ExecContext(ctx, `INSERT INTO convites (token_hash, tipo, expira_em) VALUES ($1, $2, $3)`,
			hash, ConvitePrimeiroAdmin, s.agora().Add(DuracaoConvite))
		return err
	})
	return token, err
}

// GerarConvite cria o link de acesso de uma pessoa da equipe: convite se ela
// ainda não tem senha, redefinição se já tem. O link anterior deixa de valer.
func (s *Store) GerarConvite(ctx context.Context, vendedorID string) (token, tipo string, expira time.Time, err error) {
	err = s.emTransacao(ctx, func(tx *sql.Tx) error {
		var email, acesso string
		var temSenha bool
		e := tx.QueryRowContext(ctx, `SELECT coalesce(email, ''), acesso, senha_hash IS NOT NULL
			FROM vendedores WHERE id = $1`, vendedorID).Scan(&email, &acesso, &temSenha)
		if errors.Is(e, sql.ErrNoRows) {
			return ErrNaoEncontrado
		}
		if e != nil {
			return e
		}
		if email == "" {
			return ErrValidacao{"Cadastre o e-mail da pessoa antes de gerar o link de acesso."}
		}
		if acesso != "Ativo" {
			return ErrValidacao{"Esta pessoa está Inativa. Mude o acesso para Ativo antes de gerar o link."}
		}
		tipo = ConviteNovo
		if temSenha {
			tipo = ConviteRedefinicao
		}
		if _, e := tx.ExecContext(ctx, `DELETE FROM convites WHERE vendedor_id = $1 AND usado_em IS NULL`, vendedorID); e != nil {
			return e
		}
		var hash string
		token, hash = novoToken()
		expira = s.agora().Add(DuracaoConvite)
		_, e = tx.ExecContext(ctx, `INSERT INTO convites (token_hash, tipo, vendedor_id, expira_em) VALUES ($1, $2, $3, $4)`,
			hash, tipo, vendedorID, expira)
		return e
	})
	return token, tipo, expira, err
}

// GerarConvitePorEmail é o caminho de quem tem acesso ao servidor (comando
// link-acesso): resolve o caso do único administrador que esqueceu a senha,
// quando ninguém mais pode gerar o link pela tela.
func (s *Store) GerarConvitePorEmail(ctx context.Context, email string) (token, tipo, nome string, expira time.Time, err error) {
	var id string
	err = s.db.QueryRowContext(ctx, `SELECT id, nome FROM vendedores WHERE lower(email) = $1`, normalizarEmail(email)).
		Scan(&id, &nome)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", "", expira, ErrNaoEncontrado
	}
	if err != nil {
		return "", "", "", expira, err
	}
	token, tipo, expira, err = s.GerarConvite(ctx, id)
	return token, tipo, nome, expira, err
}

// Convite é o que a tela mostra antes de a pessoa criar a senha.
type Convite struct {
	Tipo  string `json:"tipo"`
	Nome  string `json:"nome"`
	Email string `json:"email"`
}

type conviteValido struct {
	Convite
	vendedorID string
}

func buscarConvite(ctx context.Context, q queryer, token string, travar bool, agora time.Time) (conviteValido, error) {
	var c conviteValido
	var vendedorID sql.NullString
	query := `SELECT c.tipo, c.vendedor_id, coalesce(v.nome, ''), coalesce(v.email, '')
		FROM convites c LEFT JOIN vendedores v ON v.id = c.vendedor_id
		WHERE c.token_hash = $1 AND c.usado_em IS NULL AND c.expira_em > $2
		AND (c.vendedor_id IS NULL OR v.acesso = 'Ativo')
		AND (c.tipo <> 'primeiro-admin' OR NOT EXISTS (SELECT 1 FROM vendedores WHERE papel = 'Administrador'))`
	if travar {
		query += ` FOR UPDATE OF c`
	}
	err := q.QueryRowContext(ctx, query, hashToken(token), agora).Scan(&c.Tipo, &vendedorID, &c.Nome, &c.Email)
	if errors.Is(err, sql.ErrNoRows) {
		return c, ErrConviteInvalido
	}
	c.vendedorID = vendedorID.String
	return c, err
}

// ConsultarConvite diz se o link ainda vale e para quem é.
func (s *Store) ConsultarConvite(ctx context.Context, token string) (Convite, error) {
	c, err := buscarConvite(ctx, s.db, token, false, s.agora())
	return c.Convite, err
}

// DadosConvite é o que a pessoa preenche ao abrir o link. Nome e e-mail só
// valem no primeiro acesso do administrador; nos outros já vêm do cadastro.
type DadosConvite struct {
	Nome  string `json:"nome"`
	Email string `json:"email"`
	Senha string `json:"senha"`
}

// UsarConvite grava a senha (ou cria o primeiro administrador), queima o link
// e já abre a sessão. Tudo numa transação: link usado pela metade não existe.
func (s *Store) UsarConvite(ctx context.Context, token string, d DadosConvite) (Usuario, string, error) {
	var u Usuario
	var sessao string
	if err := validarSenha(d.Senha); err != nil {
		return u, "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(d.Senha), bcrypt.DefaultCost)
	if err != nil {
		return u, "", err
	}
	err = s.emTransacao(ctx, func(tx *sql.Tx) error {
		c, err := buscarConvite(ctx, tx, token, true, s.agora())
		if err != nil {
			return err
		}
		if c.Tipo == ConvitePrimeiroAdmin {
			if u, err = criarPrimeiroAdmin(ctx, tx, d, string(hash)); err != nil {
				return err
			}
		} else {
			if _, err := tx.ExecContext(ctx, `UPDATE vendedores SET senha_hash = $1 WHERE id = $2`, string(hash), c.vendedorID); err != nil {
				return err
			}
			// Senha nova derruba as sessões abertas com a antiga.
			if _, err := tx.ExecContext(ctx, `DELETE FROM sessoes WHERE vendedor_id = $1`, c.vendedorID); err != nil {
				return err
			}
			if err := tx.QueryRowContext(ctx, `SELECT id, nome, papel, email FROM vendedores WHERE id = $1`, c.vendedorID).
				Scan(&u.ID, &u.Nome, &u.Papel, &u.Email); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE convites SET usado_em = $2 WHERE token_hash = $1`, hashToken(token), s.agora()); err != nil {
			return err
		}
		sessao, err = criarSessao(ctx, tx, u.ID, s.agora())
		return err
	})
	return u, sessao, err
}

func criarPrimeiroAdmin(ctx context.Context, tx *sql.Tx, d DadosConvite, hash string) (Usuario, error) {
	u := Usuario{ID: novoID(), Nome: strings.TrimSpace(d.Nome), Papel: PapelAdministrador, Email: normalizarEmail(d.Email)}
	if u.Nome == "" {
		return u, ErrValidacao{"Informe seu nome."}
	}
	if err := validarEmail(u.Email); err != nil {
		return u, err
	}
	// Outro link de primeiro acesso pode ter sido usado no meio-tempo.
	var existe bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM vendedores WHERE papel = $1)`, PapelAdministrador).
		Scan(&existe); err != nil {
		return u, err
	}
	if existe {
		return u, ErrConviteInvalido
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO vendedores (id, nome, papel, acesso, comissao, email, senha_hash)
		VALUES ($1, $2, $3, 'Ativo', 0, $4, $5)`, u.ID, u.Nome, u.Papel, u.Email, hash); err != nil {
		return u, traduzirErroEmail(err)
	}
	// Vendas registradas no tempo do usuário fixo "admin" passam para a pessoa de verdade.
	_, err := tx.ExecContext(ctx, `UPDATE pedidos SET vendedor_id = $1 WHERE vendedor_id = 'admin'`, u.ID)
	return u, err
}
