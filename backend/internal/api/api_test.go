package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ilovejalecos/backend/internal/loja"
	"github.com/ilovejalecos/backend/internal/testdb"
)

// Papéis semeados direto na loja, cada um com uma sessão aberta (o valor do cookie).
type equipe struct {
	h                    http.Handler
	loja                 *loja.Store
	admin, ger, vend, v2 string
}

func novoServidor(t *testing.T) equipe {
	t.Helper()
	ctx := context.Background()
	l := testdb.Open(t)
	e := equipe{h: New(l, slog.New(slog.NewTextHandler(io.Discard, nil)), Opcoes{}), loja: l}
	sessao := map[string]*string{"admin": &e.admin, "ger": &e.ger, "vend": &e.vend, "vend2": &e.v2}
	for _, v := range []loja.Vendedor{
		{ID: "admin", Nome: "Dona", Papel: loja.PapelAdministrador, Acesso: "Ativo"},
		{ID: "ger", Nome: "Gil", Papel: loja.PapelGerente, Acesso: "Ativo"},
		{ID: "vend", Nome: "Ana", Papel: loja.PapelVendedor, Acesso: "Ativo", Comissao: 5},
		{ID: "vend2", Nome: "Bia", Papel: loja.PapelVendedor, Acesso: "Ativo"},
	} {
		v.ID = v.ID + "-id"
		if err := l.SalvarVendedor(ctx, v); err != nil {
			t.Fatal(err)
		}
		token, err := l.CriarSessao(ctx, v.ID)
		if err != nil {
			t.Fatal(err)
		}
		*sessao[strings.TrimSuffix(v.ID, "-id")] = token
	}
	return e
}

func chamar(h http.Handler, sessao, metodo, caminho, corpo string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(metodo, caminho, strings.NewReader(corpo))
	if sessao != "" {
		req.AddCookie(&http.Cookie{Name: CookieSessao, Value: sessao})
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestPermissoes(t *testing.T) {
	e := novoServidor(t)
	h := e.h
	casos := []struct {
		nome, usuario, metodo, caminho, corpo string
		status                                int
	}{
		{"sem sessão", "", "GET", "/api/estado", "", 401},
		{"sessão inventada", "token-falso", "GET", "/api/estado", "", 401},
		{"vendedora lê", e.vend, "GET", "/api/estado", "", 200},
		{"vendedora cria cliente", e.vend, "PUT", "/api/clientes/c1", `{"nome":"Maria"}`, 200},
		{"vendedora não exclui cliente", e.vend, "DELETE", "/api/clientes/c1", "", 403},
		{"vendedora não dá entrada no estoque", e.vend, "POST", "/api/estoque/entrada", `{"id":"e1","modelo":"J","cor":"B","tamanho":"M","quantidade":1}`, 403},
		{"gerente dá entrada no estoque", e.ger, "POST", "/api/estoque/entrada", `{"id":"e1","modelo":"J","cor":"B","tamanho":"M","quantidade":1}`, 200},
		{"gerente não mexe na equipe", e.ger, "PUT", "/api/vendedores/x", `{"nome":"X","papel":"Gerente","acesso":"Ativo"}`, 403},
		{"gerente exclui cliente", e.ger, "DELETE", "/api/clientes/c1", "", 200},
		{"vendedora não ajusta estoque", e.vend, "PATCH", "/api/estoque", `{"itens":[{"id":"e1","quantidade":2}]}`, 403},
		{"gerente ajusta estoque", e.ger, "PATCH", "/api/estoque", `{"itens":[{"id":"e1","quantidade":2}]}`, 200},
		{"gerente não exclui vendedor", e.ger, "DELETE", "/api/vendedores/vend-id", "", 403},
		{"vendedora não exclui produto", e.vend, "POST", "/api/estoque/remover", `{"ids":["e1"]}`, 403},
		{"gerente exclui produto", e.ger, "POST", "/api/estoque/remover", `{"ids":["e1"]}`, 200},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			rec := chamar(h, c.usuario, c.metodo, c.caminho, c.corpo)
			if rec.Code != c.status {
				t.Fatalf("esperava %d, veio %d: %s", c.status, rec.Code, rec.Body)
			}
		})
	}
}

func TestErrosViramMensagem(t *testing.T) {
	e := novoServidor(t)
	h := e.h
	if rec := chamar(h, e.admin, "PUT", "/api/vendedores/x", `{"nome":"X","papel":"Gerente","acesso":"Ativo","comissao":120}`); rec.Code != 422 ||
		!strings.Contains(rec.Body.String(), "comissão entre 0 e 100") {
		t.Fatalf("comissão inválida: %d %s", rec.Code, rec.Body)
	}
	if rec := chamar(h, e.admin, "PUT", "/api/clientes/c1", `{nome`); rec.Code != 400 {
		t.Fatalf("JSON quebrado: %d %s", rec.Code, rec.Body)
	}
}
