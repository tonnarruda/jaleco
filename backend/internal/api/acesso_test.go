package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/ilovejalecos/backend/internal/loja"
	"github.com/ilovejalecos/backend/internal/testdb"
)

func cookieDe(t *testing.T, rec interface{ Result() *http.Response }) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == CookieSessao {
			return c
		}
	}
	t.Fatal("resposta sem cookie de sessão")
	return nil
}

func TestLoginLogoutEConvitePorHTTP(t *testing.T) {
	e := novoServidor(t)
	ctx := context.Background()

	// Admin cadastra e-mail da Ana e gera o link.
	if rec := chamar(e.h, e.admin, "PUT", "/api/vendedores/vend-id",
		`{"nome":"Ana","papel":"Vendedor(a)","acesso":"Ativo","comissao":5,"metas":[],"email":"ana@loja.com"}`); rec.Code != 200 {
		t.Fatalf("salvar e-mail: %d %s", rec.Code, rec.Body)
	}
	if rec := chamar(e.h, e.ger, "POST", "/api/vendedores/vend-id/convite", ""); rec.Code != 403 {
		t.Fatalf("gerente não gera convite: %d", rec.Code)
	}
	rec := chamar(e.h, e.admin, "POST", "/api/vendedores/vend-id/convite", "")
	var conv struct{ Token, Tipo string }
	json.NewDecoder(rec.Body).Decode(&conv)
	if rec.Code != 200 || conv.Token == "" || conv.Tipo != loja.ConviteNovo {
		t.Fatalf("gerar convite: %d %+v", rec.Code, conv)
	}

	// Link público: consultar e usar, sem sessão. Já sai logada.
	if rec := chamar(e.h, "", "GET", "/api/convites/"+conv.Token, ""); rec.Code != 200 || !strings.Contains(rec.Body.String(), "Ana") {
		t.Fatalf("consultar convite: %d %s", rec.Code, rec.Body)
	}
	rec = chamar(e.h, "", "POST", "/api/convites/"+conv.Token, `{"senha":"senha-da-ana"}`)
	if rec.Code != 200 {
		t.Fatalf("usar convite: %d %s", rec.Code, rec.Body)
	}
	c := cookieDe(t, rec)
	if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie sem proteção: %+v", c)
	}
	if rec := chamar(e.h, c.Value, "GET", "/api/sessao", ""); rec.Code != 200 || !strings.Contains(rec.Body.String(), "ana@loja.com") {
		t.Fatalf("sessão após convite: %d %s", rec.Code, rec.Body)
	}

	// Login e logout.
	rec = chamar(e.h, "", "POST", "/api/login", `{"email":"ana@loja.com","senha":"senha-da-ana"}`)
	if rec.Code != 200 {
		t.Fatalf("login: %d %s", rec.Code, rec.Body)
	}
	sessao := cookieDe(t, rec).Value
	if rec := chamar(e.h, sessao, "POST", "/api/logout", ""); rec.Code != 204 || cookieDe(t, rec).MaxAge >= 0 {
		t.Fatalf("logout deveria apagar o cookie: %d", rec.Code)
	}
	if rec := chamar(e.h, sessao, "GET", "/api/estado", ""); rec.Code != 401 {
		t.Fatalf("sessão encerrada ainda vale: %d", rec.Code)
	}
	if _, err := e.loja.UsuarioDaSessao(ctx, sessao); err == nil {
		t.Fatal("logout não apagou a sessão no banco")
	}
}

func TestTentativasDeLoginLimitadas(t *testing.T) {
	e := novoServidor(t)
	for i := 0; i < 5; i++ {
		if rec := chamar(e.h, "", "POST", "/api/login", `{"email":"x@loja.com","senha":"errada"}`); rec.Code != 422 {
			t.Fatalf("tentativa %d: %d %s", i+1, rec.Code, rec.Body)
		}
	}
	// O mesmo e-mail com outra caixa ou espaços é a mesma conta: também barrado.
	for _, email := range []string{"x@loja.com", "X@loja.com", " x@LOJA.com "} {
		if rec := chamar(e.h, "", "POST", "/api/login", `{"email":"`+email+`","senha":"errada"}`); rec.Code != 429 {
			t.Fatalf("%q depois de 5 erros deveria ser barrado: %d", email, rec.Code)
		}
	}
	// Outro e-mail não herda o bloqueio.
	if rec := chamar(e.h, "", "POST", "/api/login", `{"email":"y@loja.com","senha":"errada"}`); rec.Code != 422 {
		t.Fatalf("outro e-mail não pode ser barrado: %d", rec.Code)
	}
}

func TestVendedorNaoAlcancaPedidoDeOutroPelaAPI(t *testing.T) {
	e := novoServidor(t)
	pedido := `{"clienteId":"","etapa":"Orçamento","pecas":[]}`
	if rec := chamar(e.h, e.v2, "PUT", "/api/pedidos/p-bia", pedido); rec.Code != 200 {
		t.Fatalf("Bia cria pedido: %d %s", rec.Code, rec.Body)
	}
	// Ana não vê o pedido da Bia no estado...
	rec := chamar(e.h, e.vend, "GET", "/api/estado", "")
	if strings.Contains(rec.Body.String(), "p-bia") {
		t.Fatal("Ana recebeu o pedido da Bia")
	}
	// ...nem consegue alterá-lo pela API.
	if rec := chamar(e.h, e.vend, "PUT", "/api/pedidos/p-bia", `{"etapa":"Cancelado","pecas":[]}`); rec.Code != 404 {
		t.Fatalf("Ana alterou pedido da Bia: %d %s", rec.Code, rec.Body)
	}
	// O admin vê os dois lados.
	if rec := chamar(e.h, e.admin, "GET", "/api/estado", ""); !strings.Contains(rec.Body.String(), "p-bia") {
		t.Fatal("admin deveria ver o pedido da Bia")
	}
}

func TestAdminNaoExcluiASiMesmo(t *testing.T) {
	e := novoServidor(t)
	if rec := chamar(e.h, e.admin, "DELETE", "/api/vendedores/admin-id", ""); rec.Code != 422 {
		t.Fatalf("autoexclusão: %d %s", rec.Code, rec.Body)
	}
}

func TestTokenDeConviteForaDoLog(t *testing.T) {
	if got := CaminhoSeguro("/api/convites/segredo123"); strings.Contains(got, "segredo123") {
		t.Fatalf("token no log: %s", got)
	}
	if got := CaminhoSeguro("/convite/segredo123"); strings.Contains(got, "segredo123") {
		t.Fatalf("token no log: %s", got)
	}
}

func TestBotaoPrimeiroAdministrador(t *testing.T) {
	l := testdb.Open(t)
	h := New(l, slog.New(slog.NewTextHandler(io.Discard, nil)), Opcoes{})

	if rec := chamar(h, "", "GET", "/api/configuracao", ""); !strings.Contains(rec.Body.String(), `"temAdministrador":false`) {
		t.Fatalf("banco vazio: %s", rec.Body)
	}
	// Dois cliques (dois aparelhos): o segundo não derruba o primeiro.
	var primeiro, segundo struct{ Token string }
	json.NewDecoder(chamar(h, "", "POST", "/api/primeiro-acesso", "").Body).Decode(&primeiro)
	rec := chamar(h, "", "POST", "/api/primeiro-acesso", "")
	json.NewDecoder(rec.Body).Decode(&segundo)
	if rec.Code != 200 || segundo.Token == "" || segundo.Token == primeiro.Token {
		t.Fatalf("gerar link: %d %+v %+v", rec.Code, primeiro, segundo)
	}
	if rec := chamar(h, "", "GET", "/api/convites/"+primeiro.Token, ""); rec.Code != 200 {
		t.Fatalf("link do primeiro clique tem que continuar valendo: %d", rec.Code)
	}
	if rec := chamar(h, "", "POST", "/api/convites/"+segundo.Token,
		`{"nome":"Dona","email":"dona@loja.com","senha":"senha-da-dona"}`); rec.Code != 200 {
		t.Fatalf("criar admin: %d %s", rec.Code, rec.Body)
	}

	// Com admin, o botão some e a rota recusa.
	if rec := chamar(h, "", "GET", "/api/configuracao", ""); !strings.Contains(rec.Body.String(), `"temAdministrador":true`) {
		t.Fatalf("depois do admin: %s", rec.Body)
	}
	if rec := chamar(h, "", "POST", "/api/primeiro-acesso", ""); rec.Code != 422 {
		t.Fatalf("com admin, não pode gerar outro: %d %s", rec.Code, rec.Body)
	}
	// O link que sobrou do primeiro clique morre assim que existe admin.
	if rec := chamar(h, "", "GET", "/api/convites/"+primeiro.Token, ""); rec.Code != 422 {
		t.Fatalf("link que sobrou deveria morrer com o admin criado: %d", rec.Code)
	}
}
