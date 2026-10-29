package loja_test

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/ilovejalecos/backend/internal/loja"
)

// relogio é um relógio de teste que só anda quando o teste manda.
type relogio struct{ t time.Time }

func (r *relogio) agora() time.Time        { return r.t }
func (r *relogio) avancar(d time.Duration) { r.t = r.t.Add(d) }

func lojaComRelogio(t *testing.T) (*Store, *relogio) {
	t.Helper()
	s := novaLoja(t)
	r := &relogio{t: time.Now()}
	s.DefinirRelogio(r.agora)
	return s, r
}

func TestConviteVenceEm72Horas(t *testing.T) {
	ctx := context.Background()
	s, r := lojaComRelogio(t)
	s.SalvarVendedor(ctx, Vendedor{ID: "ana", Nome: "Ana", Papel: PapelVendedor, Acesso: "Ativo", Email: "ana@loja.com"})
	token, _, expira, err := s.GerarConvite(ctx, "ana")
	if err != nil {
		t.Fatal(err)
	}
	if got := expira.Sub(r.agora()); got != DuracaoConvite {
		t.Fatalf("validade informada: %v, esperava %v", got, DuracaoConvite)
	}

	r.avancar(DuracaoConvite - time.Minute)
	if _, err := s.ConsultarConvite(ctx, token); err != nil {
		t.Fatalf("um minuto antes de vencer ainda vale: %v", err)
	}

	r.avancar(2 * time.Minute)
	if _, err := s.ConsultarConvite(ctx, token); !errors.Is(err, ErrConviteInvalido) {
		t.Fatalf("depois de 72h deveria estar vencido, veio %v", err)
	}
	if _, _, err := s.UsarConvite(ctx, token, DadosConvite{Senha: "senha-da-ana"}); !errors.Is(err, ErrConviteInvalido) {
		t.Fatalf("link vencido não pode criar senha, veio %v", err)
	}
	if _, _, err := s.Entrar(ctx, "ana@loja.com", "senha-da-ana"); !errors.Is(err, ErrCredenciais) {
		t.Fatalf("a senha do link vencido não pode ter sido gravada: %v", err)
	}

	// Um link novo resolve.
	novo, _, _, _ := s.GerarConvite(ctx, "ana")
	if _, _, err := s.UsarConvite(ctx, novo, DadosConvite{Senha: "senha-da-ana"}); err != nil {
		t.Fatalf("link novo depois do vencido: %v", err)
	}
}

func TestPrimeiroAcessoTambemVence(t *testing.T) {
	ctx := context.Background()
	s, r := lojaComRelogio(t)
	token, _ := s.GarantirConvitePrimeiroAdmin(ctx)
	r.avancar(DuracaoConvite + time.Minute)
	if _, _, err := s.UsarConvite(ctx, token, DadosConvite{Nome: "Dona", Email: "dona@loja.com", Senha: "senha-da-dona"}); !errors.Is(err, ErrConviteInvalido) {
		t.Fatalf("link de primeiro acesso vencido: %v", err)
	}
	if tem, _ := s.TemAdministrador(ctx); tem {
		t.Fatal("link vencido criou administrador")
	}
	// O botão continua disponível e gera um link que vale a partir de agora.
	novo, _ := s.GarantirConvitePrimeiroAdmin(ctx)
	if _, _, err := s.UsarConvite(ctx, novo, DadosConvite{Nome: "Dona", Email: "dona@loja.com", Senha: "senha-da-dona"}); err != nil {
		t.Fatalf("link novo: %v", err)
	}
}

func TestSessaoVenceEm30Dias(t *testing.T) {
	ctx := context.Background()
	s, r := lojaComRelogio(t)
	s.SalvarVendedor(ctx, Vendedor{ID: "ana", Nome: "Ana", Papel: PapelVendedor, Acesso: "Ativo"})
	sessao, err := s.CriarSessao(ctx, "ana")
	if err != nil {
		t.Fatal(err)
	}

	r.avancar(DuracaoSessao - time.Minute)
	if _, err := s.UsuarioDaSessao(ctx, sessao); err != nil {
		t.Fatalf("um minuto antes de vencer ainda vale: %v", err)
	}
	r.avancar(2 * time.Minute)
	if _, err := s.UsuarioDaSessao(ctx, sessao); !errors.Is(err, ErrNaoAutenticado) {
		t.Fatalf("depois de 30 dias deveria cair, veio %v", err)
	}
}
