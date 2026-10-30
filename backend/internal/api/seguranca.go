package api

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// limitador conta tentativas erradas de login por chave (o e-mail) numa
// janela de tempo. Fica em memória: reinício do servidor zera a contagem.
type limitador struct {
	mu     sync.Mutex
	max    int
	janela time.Duration
	falhas map[string][]time.Time
	agora  func() time.Time // trocável nos testes
}

func novoLimitador(max int, janela time.Duration) *limitador {
	return &limitador{max: max, janela: janela, falhas: map[string][]time.Time{}, agora: time.Now}
}

func (l *limitador) recentes(chave string, agora time.Time) []time.Time {
	var ok []time.Time
	for _, t := range l.falhas[chave] {
		if agora.Sub(t) < l.janela {
			ok = append(ok, t)
		}
	}
	if len(ok) == 0 {
		delete(l.falhas, chave)
	} else {
		l.falhas[chave] = ok
	}
	return ok
}

func (l *limitador) permitido(chave string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.recentes(chave, l.agora())) < l.max
}

func (l *limitador) falhou(chave string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	agora := l.agora()
	l.falhas[chave] = append(l.recentes(chave, agora), agora)
}

func (l *limitador) limpar(chave string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.falhas, chave)
}

// cabecalhos protege as respostas da API, que também é alcançável direto no
// Render, sem passar pelo Vercel. A API só devolve JSON: nada nela carrega
// script nem pode ser emoldurado, e resposta com dados da loja não vai para cache.
func cabecalhos(next http.Handler, https bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cache-Control", "no-store")
		if https {
			h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

// CaminhoSeguro tira o token de links de uso único antes de ir para o log:
// quem lê o log não pode usar o convite de outra pessoa.
func CaminhoSeguro(caminho string) string { return caminhoSeguro(caminho) }

func caminhoSeguro(caminho string) string {
	for _, prefixo := range []string{"/api/convites/", "/convite/"} {
		if strings.HasPrefix(caminho, prefixo) {
			return prefixo + "…"
		}
	}
	return caminho
}
