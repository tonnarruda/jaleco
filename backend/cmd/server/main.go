// Servidor da I Love Jalecos: API em /api e, se -static apontar para o build do
// frontend, o próprio app nas demais rotas.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ilovejalecos/backend/internal/api"
	"github.com/ilovejalecos/backend/internal/loja"
)

func main() {
	// go run ./cmd/server link-acesso dona@loja.com
	if len(os.Args) > 1 && os.Args[1] == "link-acesso" {
		os.Exit(linkAcesso(os.Args[2:]))
	}

	addr := flag.String("addr", envOu("ADDR", ":8080"), "endereço HTTP")
	dbURL := flag.String("db", envOu("DATABASE_URL", "postgres://jaleco:jaleco@localhost:5433/jaleco?sslmode=disable"), "URL do Postgres")
	static := flag.String("static", envOu("STATIC_DIR", ""), "pasta do build do frontend (opcional)")
	cookieSeguro := flag.Bool("cookie-seguro", os.Getenv("COOKIE_SEGURO") == "1", "cookie de sessão só por HTTPS (ligue em produção)")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := config{addr: *addr, dbURL: *dbURL, static: *static, opcoes: api.Opcoes{CookieSeguro: *cookieSeguro}}
	if err := run(cfg, log); err != nil {
		log.Error("servidor parou", "erro", err)
		os.Exit(1)
	}
}

type config struct {
	addr, dbURL, static string
	opcoes              api.Opcoes
}

func run(cfg config, log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	inicio, cancel := context.WithTimeout(ctx, 15*time.Second)
	store, err := loja.Open(inicio, cfg.dbURL)
	cancel()
	if err != nil {
		return err
	}
	defer store.Close()

	mux := http.NewServeMux()
	mux.Handle("/api/", api.New(store, log, cfg.opcoes))
	if cfg.static != "" {
		mux.Handle("/", spa(cfg.static))
	}

	srv := &http.Server{
		Addr:              cfg.addr,
		Handler:           registrar(log, mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(shutdown)
	}()

	// Sem a URL do banco no log: ela carrega a senha.
	log.Info("ouvindo", "addr", cfg.addr, "static", cfg.static, "cookieSeguro", cfg.opcoes.CookieSeguro)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// linkAcesso gera, pelo servidor, o link de acesso de alguém da equipe. É a
// saída para quando ninguém consegue gerar pela tela, como a única
// administradora que esqueceu a senha. Quem roda o comando tem acesso ao
// servidor e ao banco, então não abre nada que essa pessoa já não tenha.
func linkAcesso(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "uso: server link-acesso <e-mail>")
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	store, err := loja.Open(ctx, envOu("DATABASE_URL", "postgres://jaleco:jaleco@localhost:5433/jaleco?sslmode=disable"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro ao abrir o banco:", err)
		return 1
	}
	defer store.Close()

	token, tipo, nome, expira, err := store.GerarConvitePorEmail(ctx, args[0])
	var ev loja.ErrValidacao
	switch {
	case errors.Is(err, loja.ErrNaoEncontrado):
		fmt.Fprintf(os.Stderr, "Nenhuma pessoa da equipe com o e-mail %q.\n", args[0])
		return 1
	case errors.As(err, &ev):
		fmt.Fprintln(os.Stderr, ev.Msg)
		return 1
	case err != nil:
		fmt.Fprintln(os.Stderr, "erro:", err)
		return 1
	}
	finalidade := "criar a senha"
	if tipo == loja.ConviteRedefinicao {
		finalidade = "definir uma nova senha (a antiga deixa de valer)"
	}
	appURL := strings.TrimRight(envOu("APP_URL", "http://localhost:5173"), "/")
	fmt.Printf("Link para %s %s, uso único, vale até %s:\n\n  %s/convite/%s\n\n",
		nome, finalidade, expira.Format("02/01 15:04"), appURL, token)
	return 0
}

// spa serve arquivos do build e devolve index.html para as rotas do React Router.
func spa(dir string) http.Handler {
	files := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat(filepath.Join(dir, filepath.Clean(r.URL.Path))); err != nil {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		files.ServeHTTP(w, r)
	})
}

func registrar(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inicio := time.Now()
		next.ServeHTTP(w, r)
		log.Info("req", "metodo", r.Method, "caminho", api.CaminhoSeguro(r.URL.Path), "duracao", time.Since(inicio))
	})
}

func envOu(chave, padrao string) string {
	if v := os.Getenv(chave); v != "" {
		return v
	}
	return padrao
}
