// Package api expõe a loja por HTTP/JSON. Toda escrita responde com o estado
// que o papel de quem pediu pode ver, para a tela substituir o que montou localmente.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ilovejalecos/backend/internal/loja"
)

// CookieSessao guarda o token da sessão. HttpOnly: o JavaScript da página não o lê.
const CookieSessao = "jaleco_sessao"

type Permissao int

const (
	qualquerUsuario Permissao = iota
	gestor                    // Administrador ou Gerente
	administrador
)

type Opcoes struct {
	// CookieSeguro manda o cookie só por HTTPS. Ligado em produção.
	CookieSeguro bool
}

type Server struct {
	loja   *loja.Store
	log    *slog.Logger
	opcoes Opcoes
	login  *limitador
}

func New(l *loja.Store, log *slog.Logger, opcoes Opcoes) http.Handler {
	s := &Server{loja: l, log: log, opcoes: opcoes, login: novoLimitador(5, 15*time.Minute)}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/saude", func(w http.ResponseWriter, r *http.Request) {
		escreverJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Sem sessão: entrar, sair e os links de uso único.
	mux.Handle("POST /api/login", s.publico(s.entrar))
	mux.Handle("POST /api/logout", s.publico(s.sair))
	mux.Handle("GET /api/convites/{token}", s.publico(s.consultarConvite))
	mux.Handle("POST /api/convites/{token}", s.publico(s.usarConvite))
	mux.Handle("GET /api/configuracao", s.publico(s.configuracao))
	mux.Handle("POST /api/primeiro-acesso", s.publico(s.primeiroAcesso))

	mux.Handle("GET /api/sessao", s.com(qualquerUsuario, s.sessao))
	mux.Handle("GET /api/estado", s.com(qualquerUsuario, s.estado))
	mux.Handle("PUT /api/clientes/{id}", s.com(qualquerUsuario, s.salvarCliente))
	mux.Handle("DELETE /api/clientes/{id}", s.com(gestor, s.excluirCliente))
	mux.Handle("PUT /api/pedidos/{id}", s.com(qualquerUsuario, s.salvarPedido))
	mux.Handle("DELETE /api/pedidos/{id}", s.com(gestor, s.excluirPedido))
	mux.Handle("POST /api/estoque/entrada", s.com(gestor, s.entradaEstoque))
	mux.Handle("PATCH /api/estoque", s.com(gestor, s.ajustarEstoque))
	mux.Handle("POST /api/estoque/remover", s.com(gestor, s.removerVariosEstoque))
	mux.Handle("PUT /api/vendedores/{id}", s.com(administrador, s.salvarVendedor))
	mux.Handle("DELETE /api/vendedores/{id}", s.com(administrador, s.excluirVendedor))
	mux.Handle("POST /api/vendedores/{id}/convite", s.com(administrador, s.gerarConvite))
	return mux
}

type handlerFunc func(w http.ResponseWriter, r *http.Request) error

type chaveUsuario struct{}

// usuarioDe devolve quem está logado (só dentro de rotas montadas com `com`).
func usuarioDe(r *http.Request) loja.Usuario {
	u, _ := r.Context().Value(chaveUsuario{}).(loja.Usuario)
	return u
}

// com resolve a sessão pelo cookie, confere o papel e traduz o erro do handler em resposta.
func (s *Server) com(p Permissao, h handlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string
		if c, err := r.Cookie(CookieSessao); err == nil {
			token = c.Value
		}
		u, err := s.loja.UsuarioDaSessao(r.Context(), token)
		if errors.Is(err, loja.ErrNaoAutenticado) {
			escreverErro(w, http.StatusUnauthorized, "Sua sessão terminou. Entre de novo.")
			return
		}
		if err != nil {
			s.responderErro(w, r, err)
			return
		}
		if !pode(u.Papel, p) {
			escreverErro(w, http.StatusForbidden, "Seu papel não permite essa ação.")
			return
		}
		r = r.WithContext(context.WithValue(r.Context(), chaveUsuario{}, u))
		if err := h(w, r); err != nil {
			s.responderErro(w, r, err)
		}
	})
}

func (s *Server) publico(h handlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			s.responderErro(w, r, err)
		}
	})
}

func pode(papel string, p Permissao) bool {
	switch p {
	case administrador:
		return papel == loja.PapelAdministrador
	case gestor:
		return papel == loja.PapelAdministrador || papel == loja.PapelGerente
	default:
		return true
	}
}

var errMuitasTentativas = errors.New("muitas tentativas")

func (s *Server) responderErro(w http.ResponseWriter, r *http.Request, err error) {
	var v loja.ErrValidacao
	switch {
	case errors.As(err, &v):
		escreverErro(w, http.StatusUnprocessableEntity, v.Msg)
	case errors.Is(err, loja.ErrNaoEncontrado):
		escreverErro(w, http.StatusNotFound, "Registro não encontrado.")
	case errors.Is(err, errCorpoInvalido):
		escreverErro(w, http.StatusBadRequest, "Corpo da requisição inválido.")
	case errors.Is(err, errMuitasTentativas):
		escreverErro(w, http.StatusTooManyRequests, "Muitas tentativas erradas. Espere alguns minutos e tente de novo.")
	default:
		s.log.Error("falha na requisição", "metodo", r.Method, "caminho", caminhoSeguro(r.URL.Path), "erro", err)
		escreverErro(w, http.StatusInternalServerError, "Erro interno ao salvar. Tente de novo.")
	}
}

// ---------- sessão ----------

func (s *Server) gravarSessao(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieSessao,
		Value:    token,
		Path:     "/",
		MaxAge:   int(loja.DuracaoSessao.Seconds()),
		HttpOnly: true,
		Secure:   s.opcoes.CookieSeguro,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) apagarSessao(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieSessao,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.opcoes.CookieSeguro,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) entrar(w http.ResponseWriter, r *http.Request) error {
	var corpo struct {
		Email string `json:"email"`
		Senha string `json:"senha"`
	}
	if err := lerJSON(w, r, &corpo, limitePadrao); err != nil {
		return err
	}
	// Mesma normalização do login: "X@loja.com" e " x@loja.com" são a mesma conta.
	chave := ipDe(r) + "|" + strings.ToLower(strings.TrimSpace(corpo.Email))
	if !s.login.permitido(chave) {
		return errMuitasTentativas
	}
	u, token, err := s.loja.Entrar(r.Context(), corpo.Email, corpo.Senha)
	if errors.Is(err, loja.ErrCredenciais) {
		s.login.falhou(chave)
		return err
	}
	if err != nil {
		return err
	}
	s.login.limpar(chave)
	s.gravarSessao(w, token)
	escreverJSON(w, http.StatusOK, u)
	return nil
}

func (s *Server) sair(w http.ResponseWriter, r *http.Request) error {
	if c, err := r.Cookie(CookieSessao); err == nil {
		if err := s.loja.EncerrarSessao(r.Context(), c.Value); err != nil {
			return err
		}
	}
	s.apagarSessao(w)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) sessao(w http.ResponseWriter, r *http.Request) error {
	escreverJSON(w, http.StatusOK, usuarioDe(r))
	return nil
}

// ---------- convites ----------

func (s *Server) consultarConvite(w http.ResponseWriter, r *http.Request) error {
	c, err := s.loja.ConsultarConvite(r.Context(), r.PathValue("token"))
	if err != nil {
		return err
	}
	escreverJSON(w, http.StatusOK, c)
	return nil
}

func (s *Server) usarConvite(w http.ResponseWriter, r *http.Request) error {
	var d loja.DadosConvite
	if err := lerJSON(w, r, &d, limitePadrao); err != nil {
		return err
	}
	u, token, err := s.loja.UsarConvite(r.Context(), r.PathValue("token"), d)
	if err != nil {
		return err
	}
	s.gravarSessao(w, token)
	escreverJSON(w, http.StatusOK, u)
	return nil
}

// configuracao diz à tela de entrada se ainda falta o primeiro administrador.
func (s *Server) configuracao(w http.ResponseWriter, r *http.Request) error {
	tem, err := s.loja.TemAdministrador(r.Context())
	if err != nil {
		return err
	}
	escreverJSON(w, http.StatusOK, map[string]bool{"temAdministrador": tem})
	return nil
}

// primeiroAcesso é o botão "Gerar usuário Administrador": só funciona enquanto
// não existir nenhum. Quem chegar primeiro vira admin, por isso, ao publicar,
// o admin deve ser criado antes de divulgar o endereço.
func (s *Server) primeiroAcesso(w http.ResponseWriter, r *http.Request) error {
	token, err := s.loja.GarantirConvitePrimeiroAdmin(r.Context())
	if err != nil {
		return err
	}
	if token == "" {
		return loja.ErrValidacao{Msg: "O sistema já tem administrador. Entre com seu e-mail e senha."}
	}
	escreverJSON(w, http.StatusOK, map[string]string{"token": token})
	return nil
}

// gerarConvite devolve só o token; a tela monta o link com o endereço em que está.
func (s *Server) gerarConvite(w http.ResponseWriter, r *http.Request) error {
	token, tipo, expira, err := s.loja.GerarConvite(r.Context(), r.PathValue("id"))
	if err != nil {
		return err
	}
	escreverJSON(w, http.StatusOK, map[string]any{"token": token, "tipo": tipo, "expiraEm": expira})
	return nil
}

// ---------- dados ----------

func (s *Server) estado(w http.ResponseWriter, r *http.Request) error {
	return s.responderEstado(w, r)
}

func (s *Server) salvarCliente(w http.ResponseWriter, r *http.Request) error {
	var c loja.Cliente
	if err := lerJSON(w, r, &c, limitePadrao); err != nil {
		return err
	}
	c.ID = r.PathValue("id")
	if err := s.loja.SalvarCliente(r.Context(), c); err != nil {
		return err
	}
	return s.responderEstado(w, r)
}

func (s *Server) excluirCliente(w http.ResponseWriter, r *http.Request) error {
	if err := s.loja.ExcluirCliente(r.Context(), r.PathValue("id")); err != nil {
		return err
	}
	return s.responderEstado(w, r)
}

func (s *Server) salvarPedido(w http.ResponseWriter, r *http.Request) error {
	var p loja.Pedido
	if err := lerJSON(w, r, &p, limitePadrao); err != nil {
		return err
	}
	p.ID = r.PathValue("id")
	if _, err := s.loja.SalvarPedidoComo(r.Context(), usuarioDe(r), p); err != nil {
		return err
	}
	return s.responderEstado(w, r)
}

func (s *Server) excluirPedido(w http.ResponseWriter, r *http.Request) error {
	if err := s.loja.ExcluirPedido(r.Context(), r.PathValue("id")); err != nil {
		return err
	}
	return s.responderEstado(w, r)
}

func (s *Server) entradaEstoque(w http.ResponseWriter, r *http.Request) error {
	var i loja.ItemEstoque
	if err := lerJSON(w, r, &i, limitePadrao); err != nil {
		return err
	}
	if err := s.loja.EntradaEstoque(r.Context(), i); err != nil {
		return err
	}
	return s.responderEstado(w, r)
}

// ajustarEstoque recebe os tamanhos editados de um produto e grava todos juntos.
func (s *Server) ajustarEstoque(w http.ResponseWriter, r *http.Request) error {
	var corpo struct {
		Itens []loja.Ajuste `json:"itens"`
	}
	if err := lerJSON(w, r, &corpo, limitePadrao); err != nil {
		return err
	}
	if err := s.loja.AjustarEstoque(r.Context(), corpo.Itens...); err != nil {
		return err
	}
	return s.responderEstado(w, r)
}

// removerVariosEstoque apaga um produto inteiro (todos os tamanhos) numa transação.
func (s *Server) removerVariosEstoque(w http.ResponseWriter, r *http.Request) error {
	var corpo struct {
		IDs []string `json:"ids"`
	}
	if err := lerJSON(w, r, &corpo, limitePadrao); err != nil {
		return err
	}
	if err := s.loja.RemoverEstoque(r.Context(), corpo.IDs...); err != nil {
		return err
	}
	return s.responderEstado(w, r)
}

func (s *Server) excluirVendedor(w http.ResponseWriter, r *http.Request) error {
	if r.PathValue("id") == usuarioDe(r).ID {
		return loja.ErrValidacao{Msg: "Você não pode excluir o próprio cadastro."}
	}
	if err := s.loja.ExcluirVendedor(r.Context(), r.PathValue("id")); err != nil {
		return err
	}
	return s.responderEstado(w, r)
}

func (s *Server) salvarVendedor(w http.ResponseWriter, r *http.Request) error {
	var v loja.Vendedor
	if err := lerJSON(w, r, &v, limitePadrao); err != nil {
		return err
	}
	v.ID = r.PathValue("id")
	if err := s.loja.SalvarVendedor(r.Context(), v); err != nil {
		return err
	}
	return s.responderEstado(w, r)
}

// ---------- JSON ----------

const limitePadrao = 1 << 20 // 1 MB

var errCorpoInvalido = errors.New("corpo inválido")

func lerJSON(w http.ResponseWriter, r *http.Request, dst any, limite int64) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, limite))
	if err := dec.Decode(dst); err != nil {
		return errors.Join(errCorpoInvalido, err)
	}
	return nil
}

func (s *Server) responderEstado(w http.ResponseWriter, r *http.Request) error {
	e, err := s.loja.EstadoPara(r.Context(), usuarioDe(r))
	if err != nil {
		return err
	}
	escreverJSON(w, http.StatusOK, e)
	return nil
}

func escreverJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func escreverErro(w http.ResponseWriter, status int, msg string) {
	escreverJSON(w, status, map[string]string{"erro": msg})
}
