package loja_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/ilovejalecos/backend/internal/loja"
)

func TestPrimeiroAdministrador(t *testing.T) {
	ctx := context.Background()
	s := novaLoja(t)

	antigo, err := s.GarantirConvitePrimeiroAdmin(ctx)
	if err != nil || antigo == "" {
		t.Fatalf("sem admin deveria gerar link: %q %v", antigo, err)
	}
	// Outro clique gera outro link, e o anterior continua valendo até alguém concluir.
	token, _ := s.GarantirConvitePrimeiroAdmin(ctx)
	if _, err := s.ConsultarConvite(ctx, antigo); err != nil {
		t.Fatalf("link anterior tem que continuar valendo, veio %v", err)
	}
	c, err := s.ConsultarConvite(ctx, token)
	if err != nil || c.Tipo != ConvitePrimeiroAdmin {
		t.Fatalf("link novo: %+v %v", c, err)
	}

	var ev ErrValidacao
	if _, _, err := s.UsarConvite(ctx, token, DadosConvite{Nome: "Dona", Email: "dona@loja.com", Senha: "curta"}); !errors.As(err, &ev) {
		t.Fatalf("senha curta deveria ser recusada, veio %v", err)
	}
	u, sessao, err := s.UsarConvite(ctx, token, DadosConvite{Nome: "Dona", Email: " Dona@Loja.com ", Senha: "senha-forte-1"})
	if err != nil {
		t.Fatal(err)
	}
	if u.Papel != PapelAdministrador || u.Email != "dona@loja.com" {
		t.Fatalf("admin criado errado: %+v", u)
	}
	if logado, err := s.UsuarioDaSessao(ctx, sessao); err != nil || logado.ID != u.ID {
		t.Fatalf("convite deveria já abrir a sessão: %+v %v", logado, err)
	}
	// Uso único, e com admin cadastrado o servidor não gera mais link.
	if _, _, err := s.UsarConvite(ctx, token, DadosConvite{Nome: "Intrusa", Email: "x@y.com", Senha: "senha-forte-2"}); !errors.Is(err, ErrConviteInvalido) {
		t.Fatalf("link usado deveria ser recusado, veio %v", err)
	}
	if outro, _ := s.GarantirConvitePrimeiroAdmin(ctx); outro != "" {
		t.Fatal("com admin cadastrado não pode sair link de primeiro acesso")
	}
	// O link do outro clique morre: consultar e usar.
	if _, err := s.ConsultarConvite(ctx, antigo); !errors.Is(err, ErrConviteInvalido) {
		t.Fatalf("link que sobrou deveria morrer com o admin criado, veio %v", err)
	}
	if _, _, err := s.UsarConvite(ctx, antigo, DadosConvite{Nome: "Intrusa", Email: "x@y.com", Senha: "senha-forte-2"}); !errors.Is(err, ErrConviteInvalido) {
		t.Fatalf("link que sobrou não pode criar segundo admin, veio %v", err)
	}
}

func TestLinkDeAcessoPeloServidor(t *testing.T) {
	ctx := context.Background()
	s := novaLoja(t)
	token, _ := s.GarantirConvitePrimeiroAdmin(ctx)
	s.UsarConvite(ctx, token, DadosConvite{Nome: "Dona", Email: "dona@loja.com", Senha: "senha-esquecida"})

	if _, _, _, _, err := s.GerarConvitePorEmail(ctx, "ninguem@loja.com"); !errors.Is(err, ErrNaoEncontrado) {
		t.Fatalf("e-mail inexistente: %v", err)
	}
	// A única admin esqueceu a senha: o comando do servidor gera o link de nova senha.
	link, tipo, nome, _, err := s.GerarConvitePorEmail(ctx, "  DONA@loja.com")
	if err != nil || tipo != ConviteRedefinicao || nome != "Dona" {
		t.Fatalf("link pelo servidor: %q %q %v", tipo, nome, err)
	}
	if _, _, err := s.UsarConvite(ctx, link, DadosConvite{Senha: "senha-nova-da-dona"}); err != nil {
		t.Fatal(err)
	}
	if u, _, err := s.Entrar(ctx, "dona@loja.com", "senha-nova-da-dona"); err != nil || u.Papel != PapelAdministrador {
		t.Fatalf("a admin deveria voltar a entrar: %+v %v", u, err)
	}
}

func TestConviteLoginERedefinicao(t *testing.T) {
	ctx := context.Background()
	s := novaLoja(t)
	s.SalvarVendedor(ctx, Vendedor{ID: "ana", Nome: "Ana", Papel: PapelVendedor, Acesso: "Ativo"})

	var ev ErrValidacao
	if _, _, _, err := s.GerarConvite(ctx, "ana"); !errors.As(err, &ev) {
		t.Fatalf("sem e-mail não deveria gerar link, veio %v", err)
	}
	s.SalvarVendedor(ctx, Vendedor{ID: "ana", Nome: "Ana", Papel: PapelVendedor, Acesso: "Ativo", Email: "Ana@Loja.com"})
	token, tipo, _, err := s.GerarConvite(ctx, "ana")
	if err != nil || tipo != ConviteNovo {
		t.Fatalf("convite: %q %v", tipo, err)
	}
	// Nome e e-mail enviados no convite de vendedor são ignorados: valem os do cadastro.
	u, primeira, err := s.UsarConvite(ctx, token, DadosConvite{Nome: "Outra", Email: "outra@x.com", Senha: "senha-da-ana"})
	if err != nil || u.ID != "ana" || u.Email != "ana@loja.com" {
		t.Fatalf("usar convite: %+v %v", u, err)
	}

	if _, _, err := s.Entrar(ctx, "ana@loja.com", "senha-errada"); !errors.Is(err, ErrCredenciais) {
		t.Fatalf("senha errada: %v", err)
	}
	if _, _, err := s.Entrar(ctx, "ninguem@loja.com", "senha-da-ana"); !errors.Is(err, ErrCredenciais) {
		t.Fatalf("e-mail inexistente deveria dar a mesma mensagem: %v", err)
	}
	if u, _, err := s.Entrar(ctx, "  ANA@loja.com", "senha-da-ana"); err != nil || u.ID != "ana" {
		t.Fatalf("login com e-mail em outra caixa: %+v %v", u, err)
	}

	// Já tem senha: o próximo link é de redefinição, e a senha nova derruba as sessões antigas.
	token, tipo, _, _ = s.GerarConvite(ctx, "ana")
	if tipo != ConviteRedefinicao {
		t.Fatalf("esperava redefinição, veio %q", tipo)
	}
	if _, _, err := s.UsarConvite(ctx, token, DadosConvite{Senha: "senha-nova-ana"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UsuarioDaSessao(ctx, primeira); !errors.Is(err, ErrNaoAutenticado) {
		t.Fatalf("sessão anterior à redefinição deveria cair, veio %v", err)
	}
	if _, _, err := s.Entrar(ctx, "ana@loja.com", "senha-da-ana"); !errors.Is(err, ErrCredenciais) {
		t.Fatal("senha antiga ainda entra")
	}
}

func TestInativoPerdeAcessoNaHora(t *testing.T) {
	ctx := context.Background()
	s := novaLoja(t)
	v := Vendedor{ID: "ana", Nome: "Ana", Papel: PapelVendedor, Acesso: "Ativo", Email: "ana@loja.com"}
	s.SalvarVendedor(ctx, v)
	token, _, _, _ := s.GerarConvite(ctx, "ana")
	_, sessao, _ := s.UsarConvite(ctx, token, DadosConvite{Senha: "senha-da-ana"})
	pendente, _, _, _ := s.GerarConvite(ctx, "ana")

	v.Acesso = "Inativo"
	if err := s.SalvarVendedor(ctx, v); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UsuarioDaSessao(ctx, sessao); !errors.Is(err, ErrNaoAutenticado) {
		t.Fatalf("sessão de inativo deveria cair, veio %v", err)
	}
	if _, _, err := s.Entrar(ctx, "ana@loja.com", "senha-da-ana"); !errors.Is(err, ErrCredenciais) {
		t.Fatalf("inativo não pode entrar, veio %v", err)
	}
	if _, err := s.ConsultarConvite(ctx, pendente); !errors.Is(err, ErrConviteInvalido) {
		t.Fatalf("link pendente de inativo deveria morrer, veio %v", err)
	}
}

func TestSempreUmAdministradorAtivo(t *testing.T) {
	ctx := context.Background()
	s := novaLoja(t)
	dona := Vendedor{ID: "dona", Nome: "Dona", Papel: PapelAdministrador, Acesso: "Ativo"}
	s.SalvarVendedor(ctx, dona)

	var ev ErrValidacao
	rebaixada := dona
	rebaixada.Papel = PapelGerente
	if err := s.SalvarVendedor(ctx, rebaixada); !errors.As(err, &ev) {
		t.Fatalf("único admin não pode virar gerente, veio %v", err)
	}
	inativa := dona
	inativa.Acesso = "Inativo"
	if err := s.SalvarVendedor(ctx, inativa); !errors.As(err, &ev) {
		t.Fatalf("único admin não pode ficar inativo, veio %v", err)
	}
	if err := s.ExcluirVendedor(ctx, "dona"); !errors.As(err, &ev) {
		t.Fatalf("único admin não pode ser excluído, veio %v", err)
	}
	// Com outro admin ativo, pode.
	s.SalvarVendedor(ctx, Vendedor{ID: "socio", Nome: "Sócio", Papel: PapelAdministrador, Acesso: "Ativo"})
	if err := s.SalvarVendedor(ctx, rebaixada); err != nil {
		t.Fatalf("com outro admin, rebaixar deveria passar: %v", err)
	}
}

func TestEmailRepetidoNaEquipe(t *testing.T) {
	ctx := context.Background()
	s := novaLoja(t)
	s.SalvarVendedor(ctx, Vendedor{ID: "a", Nome: "Ana", Papel: PapelVendedor, Acesso: "Ativo", Email: "ana@loja.com"})
	var ev ErrValidacao
	err := s.SalvarVendedor(ctx, Vendedor{ID: "b", Nome: "Bia", Papel: PapelVendedor, Acesso: "Ativo", Email: "ANA@loja.com"})
	if !errors.As(err, &ev) || ev.Msg != "Já existe alguém da equipe com esse e-mail." {
		t.Fatalf("e-mail repetido: %v", err)
	}
}

func TestVendedorSoVeEMexeNoQueEDele(t *testing.T) {
	ctx := context.Background()
	s := novaLoja(t)
	for _, v := range []Vendedor{
		{ID: "ana", Nome: "Ana", Papel: PapelVendedor, Acesso: "Ativo"},
		{ID: "bia", Nome: "Bia", Papel: PapelVendedor, Acesso: "Ativo"},
	} {
		s.SalvarVendedor(ctx, v)
	}
	s.SalvarCliente(ctx, Cliente{ID: "c1", Nome: "Maria"})
	ana := Usuario{ID: "ana", Papel: PapelVendedor}
	bia := Usuario{ID: "bia", Papel: PapelVendedor}

	// Ana tenta registrar em nome da Bia: o servidor troca para ela mesma.
	p, err := s.SalvarPedidoComo(ctx, ana, Pedido{ID: "p-ana", ClienteID: "c1", VendedorID: "bia", Etapa: "Orçamento"})
	if err != nil || p.VendedorID != "ana" {
		t.Fatalf("pedido da Ana: %+v %v", p, err)
	}
	if _, err := s.SalvarPedidoComo(ctx, bia, Pedido{ID: "p-bia", ClienteID: "c1", Etapa: "Orçamento"}); err != nil {
		t.Fatal(err)
	}
	// Ana não edita o da Bia: responde como inexistente, sem confirmar que existe.
	if _, err := s.SalvarPedidoComo(ctx, ana, Pedido{ID: "p-bia", ClienteID: "c1", Etapa: "Cancelado"}); !errors.Is(err, ErrNaoEncontrado) {
		t.Fatalf("Ana editou pedido da Bia: %v", err)
	}

	e, _ := s.EstadoPara(ctx, ana)
	if len(e.Pedidos) != 1 || e.Pedidos[0].ID != "p-ana" {
		t.Fatalf("Ana deveria ver só o próprio pedido: %+v", e.Pedidos)
	}
	if len(e.Vendedores) != 1 || e.Vendedores[0].ID != "ana" {
		t.Fatalf("Ana deveria ver só o próprio cadastro: %+v", e.Vendedores)
	}
	if len(e.Clientes) != 1 {
		t.Fatalf("clientes são de todos: %+v", e.Clientes)
	}
	gestor, _ := s.EstadoPara(ctx, Usuario{ID: "g", Papel: PapelGerente})
	if len(gestor.Pedidos) != 2 || len(gestor.Vendedores) != 2 {
		t.Fatalf("gerente vê tudo: %d pedidos, %d pessoas", len(gestor.Pedidos), len(gestor.Vendedores))
	}
	// O pedido da Bia continua como estava.
	for _, ped := range gestor.Pedidos {
		if ped.ID == "p-bia" && ped.Etapa != "Orçamento" {
			t.Fatalf("pedido da Bia mudou: %+v", ped)
		}
	}
}
