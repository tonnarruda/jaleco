package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/ilovejalecos/backend/internal/loja"
	"github.com/ilovejalecos/backend/internal/testdb"
)

func TestLimitadorLiberaDepoisDaJanela(t *testing.T) {
	agora := time.Now()
	l := novoLimitador(2, 15*time.Minute)
	l.agora = func() time.Time { return agora }

	l.falhou("ip|ana")
	l.falhou("ip|ana")
	if l.permitido("ip|ana") {
		t.Fatal("depois de 2 erros deveria bloquear")
	}
	if !l.permitido("ip|bia") {
		t.Fatal("o bloqueio de um e-mail não pode travar outro")
	}

	agora = agora.Add(15*time.Minute - time.Second)
	if l.permitido("ip|ana") {
		t.Fatal("um segundo antes da janela acabar ainda deveria bloquear")
	}
	agora = agora.Add(2 * time.Second)
	if !l.permitido("ip|ana") {
		t.Fatal("depois de 15 minutos deveria liberar")
	}
}

func TestLimitadorJanelaDeslizante(t *testing.T) {
	agora := time.Now()
	l := novoLimitador(2, 15*time.Minute)
	l.agora = func() time.Time { return agora }

	l.falhou("k") // t = 0
	agora = agora.Add(10 * time.Minute)
	l.falhou("k") // t = 10min
	agora = agora.Add(6 * time.Minute)
	// O primeiro erro saiu da janela (16min); o segundo ainda conta.
	if !l.permitido("k") {
		t.Fatal("só um erro dentro da janela: deveria permitir")
	}
	l.falhou("k")
	if l.permitido("k") {
		t.Fatal("dois erros dentro da janela: deveria bloquear")
	}
}

func TestAcertoZeraAsTentativas(t *testing.T) {
	l := novoLimitador(2, 15*time.Minute)
	l.falhou("k")
	l.limpar("k")
	l.falhou("k")
	if !l.permitido("k") {
		t.Fatal("login certo deveria zerar a contagem anterior")
	}
}

func TestCookieSeguroEmProducao(t *testing.T) {
	for _, seguro := range []bool{false, true} {
		l := testdb.Open(t)
		h := New(l, slog.New(slog.NewTextHandler(io.Discard, nil)), Opcoes{CookieSeguro: seguro})
		token, _ := l.GarantirConvitePrimeiroAdmin(context.Background())
		if _, _, err := l.UsarConvite(context.Background(), token,
			loja.DadosConvite{Nome: "Dona", Email: "dona@loja.com", Senha: "senha-da-dona"}); err != nil {
			t.Fatal(err)
		}

		login := chamar(h, "", "POST", "/api/login", `{"email":"dona@loja.com","senha":"senha-da-dona"}`)
		entrada := cookieDe(t, login)
		saida := cookieDe(t, chamar(h, entrada.Value, "POST", "/api/logout", ""))
		for nome, c := range map[string]*http.Cookie{"login": entrada, "logout": saida} {
			if c.Secure != seguro {
				t.Errorf("COOKIE_SEGURO=%v, cookie do %s com Secure=%v", seguro, nome, c.Secure)
			}
			if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Path != "/" {
				t.Errorf("cookie do %s sem as proteções: %+v", nome, c)
			}
		}
		if entrada.MaxAge != int(loja.DuracaoSessao.Seconds()) {
			t.Errorf("cookie deveria durar o mesmo que a sessão: %d", entrada.MaxAge)
		}
	}
}

func TestCabecalhosDeSegurancaNaAPI(t *testing.T) {
	for _, https := range []bool{false, true} {
		// /api/saude não usa o banco: dá para montar a API sem loja.
		h := New(nil, slog.New(slog.NewTextHandler(io.Discard, nil)), Opcoes{CookieSeguro: https})
		rec := chamar(h, "", "GET", "/api/saude", "")
		for chave, valor := range map[string]string{
			"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
			"X-Content-Type-Options":  "nosniff",
			"X-Frame-Options":         "DENY",
			"Cache-Control":           "no-store",
		} {
			if got := rec.Header().Get(chave); got != valor {
				t.Errorf("https=%v, %s = %q, esperado %q", https, chave, got, valor)
			}
		}
		// HSTS só faz sentido servido por HTTPS: em HTTP local o navegador ignoraria.
		if got := rec.Header().Get("Strict-Transport-Security"); (got != "") != https {
			t.Errorf("https=%v, Strict-Transport-Security = %q", https, got)
		}
	}
}
