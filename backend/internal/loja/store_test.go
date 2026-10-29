package loja_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	. "github.com/ilovejalecos/backend/internal/loja"
	"github.com/ilovejalecos/backend/internal/testdb"
)

var novaLoja = testdb.Open

func qtd(t *testing.T, s *Store, id string) int {
	t.Helper()
	e, err := s.Estado(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range e.Estoque {
		if i.ID == id {
			return i.Quantidade
		}
	}
	t.Fatalf("item %s não está no estoque", id)
	return 0
}

func pedido(id, etapa string, quantidade int) Pedido {
	return Pedido{ID: id, Etapa: etapa, Pecas: []Peca{{ID: "p", Modelo: "Jaleco", Cor: "Branco", Tamanho: "M", Quantidade: quantidade}}}
}

func TestPedidoBaixaDevolveEstoque(t *testing.T) {
	ctx := context.Background()
	s := novaLoja(t)
	if err := s.EntradaEstoque(ctx, ItemEstoque{ID: "e1", Modelo: "Jaleco", Cor: "Branco", Tamanho: "M", Quantidade: 10}); err != nil {
		t.Fatal(err)
	}

	// Orçamento não segura peça.
	if _, err := s.SalvarPedido(ctx, pedido("a", "Orçamento", 3)); err != nil {
		t.Fatal(err)
	}
	if got := qtd(t, s, "e1"); got != 10 {
		t.Fatalf("orçamento baixou estoque: %d", got)
	}

	// Virou venda: baixa 3.
	if _, err := s.SalvarPedido(ctx, pedido("a", "Em produção", 3)); err != nil {
		t.Fatal(err)
	}
	if got := qtd(t, s, "e1"); got != 7 {
		t.Fatalf("esperava 7, veio %d", got)
	}

	// Editou para 5 peças: devolve 3 e baixa 5, sem contar duas vezes.
	if _, err := s.SalvarPedido(ctx, pedido("a", "Em produção", 5)); err != nil {
		t.Fatal(err)
	}
	if got := qtd(t, s, "e1"); got != 5 {
		t.Fatalf("esperava 5, veio %d", got)
	}

	// Cancelado devolve tudo.
	if _, err := s.SalvarPedido(ctx, pedido("a", "Cancelado", 5)); err != nil {
		t.Fatal(err)
	}
	if got := qtd(t, s, "e1"); got != 10 {
		t.Fatalf("cancelado não devolveu: %d", got)
	}

	// Excluir pedido vendido devolve as peças.
	if _, err := s.SalvarPedido(ctx, pedido("b", "Pronto", 4)); err != nil {
		t.Fatal(err)
	}
	if err := s.ExcluirPedido(ctx, "b"); err != nil {
		t.Fatal(err)
	}
	if got := qtd(t, s, "e1"); got != 10 {
		t.Fatalf("exclusão não devolveu: %d", got)
	}
}

func TestNumeroDoPedidoEhDoServidor(t *testing.T) {
	ctx := context.Background()
	s := novaLoja(t)
	p1 := pedido("a", "Orçamento", 1)
	p1.Numero = 99 // o cliente não escolhe o número
	salvo, err := s.SalvarPedido(ctx, p1)
	if err != nil {
		t.Fatal(err)
	}
	if salvo.Numero != 1 {
		t.Fatalf("primeiro pedido deveria ser 1, veio %d", salvo.Numero)
	}
	p2, _ := s.SalvarPedido(ctx, pedido("b", "Orçamento", 1))
	if p2.Numero != 2 {
		t.Fatalf("segundo pedido deveria ser 2, veio %d", p2.Numero)
	}
	editado, _ := s.SalvarPedido(ctx, pedido("a", "Pronto", 1))
	if editado.Numero != 1 {
		t.Fatalf("edição mudou o número para %d", editado.Numero)
	}
}

func TestEntradaSomaNaMesmaVariacao(t *testing.T) {
	ctx := context.Background()
	s := novaLoja(t)
	s.EntradaEstoque(ctx, ItemEstoque{ID: "e1", Modelo: "Jaleco", Cor: "Branco", Tamanho: "M", Quantidade: 2, Minimo: 1})
	s.EntradaEstoque(ctx, ItemEstoque{ID: "e2", Modelo: " jaleco ", Cor: "BRANCO", Tamanho: "m", Quantidade: 3, Minimo: 4})
	e, _ := s.Estado(ctx)
	if len(e.Estoque) != 1 || e.Estoque[0].Quantidade != 5 || e.Estoque[0].Minimo != 4 {
		t.Fatalf("esperava uma variação com 5 peças e mínimo 4: %+v", e.Estoque)
	}
}

func TestHistoricoDeMetasNaoSePerde(t *testing.T) {
	ctx := context.Background()
	s := novaLoja(t)
	v := Vendedor{ID: "v1", Nome: "Ana", Papel: "Vendedor(a)", Acesso: "Ativo", Comissao: 5,
		Metas: []Meta{{Desde: "2026-08", Valor: 2000}, {Desde: "2026-10", Valor: 3000}}}
	if err := s.SalvarVendedor(ctx, v); err != nil {
		t.Fatal(err)
	}
	// A tela manda só a vigência nova: as antigas continuam.
	v.Metas = []Meta{{Desde: "2026-11", Valor: 4000}}
	if err := s.SalvarVendedor(ctx, v); err != nil {
		t.Fatal(err)
	}
	e, _ := s.Estado(ctx)
	got := e.Vendedores[0].Metas
	want := []Meta{{"2026-08", 2000}, {"2026-10", 3000}, {"2026-11", 4000}}
	if len(got) != len(want) {
		t.Fatalf("metas: %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("meta %d: esperava %+v, veio %+v", i, want[i], got[i])
		}
	}
}

func TestVendedorInvalido(t *testing.T) {
	s := novaLoja(t)
	casos := []Vendedor{
		{ID: "v", Nome: "", Papel: "Gerente", Acesso: "Ativo"},
		{ID: "v", Nome: "X", Papel: "Dono", Acesso: "Ativo"},
		{ID: "v", Nome: "X", Papel: "Gerente", Acesso: "Ativo", Comissao: 150},
		{ID: "v", Nome: "X", Papel: "Gerente", Acesso: "Ativo", Metas: []Meta{{Desde: "nov", Valor: 1}}},
		{ID: "admin", Nome: "X", Papel: "Gerente", Acesso: "Ativo"},
		{ID: "v", Nome: "X", Papel: "Gerente", Acesso: "Ativo", Email: "sem-arroba"},
	}
	for _, v := range casos {
		var ev ErrValidacao
		if err := s.SalvarVendedor(context.Background(), v); !errors.As(err, &ev) {
			t.Errorf("%+v: esperava erro de validação, veio %v", v, err)
		}
	}
}

func TestPedidosSimultaneosTemNumerosDiferentes(t *testing.T) {
	ctx := context.Background()
	s := novaLoja(t)
	if err := s.EntradaEstoque(ctx, ItemEstoque{ID: "e1", Modelo: "Jaleco", Cor: "Branco", Tamanho: "M", Quantidade: 100}); err != nil {
		t.Fatal(err)
	}
	const n = 20
	numeros := make([]int, n)
	erros := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, err := s.SalvarPedido(ctx, pedido(string(rune('a'+i)), "Pronto", 1))
			numeros[i], erros[i] = p.Numero, err
		}()
	}
	wg.Wait()
	vistos := map[int]bool{}
	for i := range n {
		if erros[i] != nil {
			t.Fatalf("pedido %d falhou: %v", i, erros[i])
		}
		if vistos[numeros[i]] {
			t.Fatalf("número %d repetido", numeros[i])
		}
		vistos[numeros[i]] = true
	}
	if got := qtd(t, s, "e1"); got != 100-n {
		t.Fatalf("estoque deveria ser %d, veio %d", 100-n, got)
	}
}

func TestCentavosNaoSePerdem(t *testing.T) {
	ctx := context.Background()
	s := novaLoja(t)
	p := pedido("a", "Pronto", 1)
	p.Pecas[0].PrecoUnitario = 189.9
	p.Desconto, p.Frete, p.ValorPago = 9.49, 22.35, 100.01
	if _, err := s.SalvarPedido(ctx, p); err != nil {
		t.Fatal(err)
	}
	e, _ := s.Estado(ctx)
	got := e.Pedidos[0]
	if got.Desconto != 9.49 || got.Frete != 22.35 || got.ValorPago != 100.01 || got.Pecas[0].PrecoUnitario != 189.9 {
		t.Fatalf("valores mudaram ao gravar: %+v", got)
	}
}

func TestPrecoDeVendaNaEntrada(t *testing.T) {
	ctx := context.Background()
	s := novaLoja(t)
	item := ItemEstoque{ID: "e1", Modelo: "Jaleco", Cor: "Branco", Tamanho: "M", Quantidade: 2, PrecoVenda: 189.9}
	if err := s.EntradaEstoque(ctx, item); err != nil {
		t.Fatal(err)
	}
	preco := func() float64 {
		e, _ := s.Estado(ctx)
		return e.Estoque[0].PrecoVenda
	}
	if got := preco(); got != 189.9 {
		t.Fatalf("preço gravado: %v", got)
	}
	// Entrada sem preço não apaga o que já existia.
	item.ID, item.PrecoVenda = "e2", 0
	s.EntradaEstoque(ctx, item)
	if got := preco(); got != 189.9 {
		t.Fatalf("entrada sem preço mudou o preço para %v", got)
	}
	// Entrada com preço novo substitui.
	item.ID, item.PrecoVenda = "e3", 199
	s.EntradaEstoque(ctx, item)
	if got := preco(); got != 199 {
		t.Fatalf("preço novo não entrou: %v", got)
	}
	// Ajuste corrige o preço.
	if err := s.AjustarEstoque(ctx, Ajuste{ID: "e1", Quantidade: 6, Minimo: 1, PrecoVenda: 179.5}); err != nil {
		t.Fatal(err)
	}
	if got := preco(); got != 179.5 {
		t.Fatalf("ajuste não mudou o preço: %v", got)
	}
	var ev ErrValidacao
	if err := s.EntradaEstoque(ctx, ItemEstoque{ID: "x", Modelo: "J", Cor: "B", Tamanho: "P", PrecoVenda: -1}); !errors.As(err, &ev) {
		t.Fatalf("preço negativo deveria ser recusado, veio %v", err)
	}
}

func TestRemoverProdutoInteiro(t *testing.T) {
	ctx := context.Background()
	s := novaLoja(t)
	for _, i := range []ItemEstoque{
		{ID: "pp", Modelo: "Slim", Cor: "Branco", Tamanho: "PP", Quantidade: 1},
		{ID: "m", Modelo: "Slim", Cor: "Branco", Tamanho: "M", Quantidade: 2},
		{ID: "outro", Modelo: "Slim", Cor: "Preto", Tamanho: "M", Quantidade: 3},
	} {
		if err := s.EntradaEstoque(ctx, i); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.RemoverEstoque(ctx, "pp", "m"); err != nil {
		t.Fatal(err)
	}
	e, _ := s.Estado(ctx)
	if len(e.Estoque) != 1 || e.Estoque[0].ID != "outro" {
		t.Fatalf("esperava só a variação preta: %+v", e.Estoque)
	}
	var ev ErrValidacao
	if err := s.RemoverEstoque(ctx); !errors.As(err, &ev) {
		t.Fatalf("lista vazia deveria ser recusada, veio %v", err)
	}
}

func TestAjusteEmLoteTudoOuNada(t *testing.T) {
	ctx := context.Background()
	s := novaLoja(t)
	s.EntradaEstoque(ctx, ItemEstoque{ID: "p", Modelo: "Slim", Cor: "Branco", Tamanho: "P", Quantidade: 1})
	s.EntradaEstoque(ctx, ItemEstoque{ID: "m", Modelo: "Slim", Cor: "Branco", Tamanho: "M", Quantidade: 1})

	if err := s.AjustarEstoque(ctx,
		Ajuste{ID: "p", Quantidade: 4, Minimo: 2, PrecoVenda: 150},
		Ajuste{ID: "m", Quantidade: 6, Minimo: 3, PrecoVenda: 160},
	); err != nil {
		t.Fatal(err)
	}
	// Um item inexistente no lote desfaz os outros.
	err := s.AjustarEstoque(ctx,
		Ajuste{ID: "p", Quantidade: 99},
		Ajuste{ID: "nao-existe", Quantidade: 1},
	)
	if !errors.Is(err, ErrNaoEncontrado) {
		t.Fatalf("esperava ErrNaoEncontrado, veio %v", err)
	}
	e, _ := s.Estado(ctx)
	got := map[string]ItemEstoque{}
	for _, i := range e.Estoque {
		got[i.ID] = i
	}
	if got["p"].Quantidade != 4 || got["p"].PrecoVenda != 150 || got["m"].Quantidade != 6 || got["m"].Minimo != 3 {
		t.Fatalf("lote com erro gravou pela metade ou o primeiro lote se perdeu: %+v", got)
	}
}

func TestNaoExcluiOQueEstaEmPedido(t *testing.T) {
	ctx := context.Background()
	s := novaLoja(t)
	for _, i := range []ItemEstoque{
		{ID: "m", Modelo: "Slim", Cor: "Branco", Tamanho: "M", Quantidade: 5},
		{ID: "g", Modelo: "Slim", Cor: "Branco", Tamanho: "G", Quantidade: 5},
	} {
		s.EntradaEstoque(ctx, i)
	}
	s.SalvarCliente(ctx, Cliente{ID: "c1", Nome: "Maria"})
	s.SalvarVendedor(ctx, Vendedor{ID: "v1", Nome: "Ana", Papel: "Vendedor(a)", Acesso: "Ativo",
		Metas: []Meta{{Desde: "2026-10", Valor: 1000}}})
	s.SalvarVendedor(ctx, Vendedor{ID: "v2", Nome: "Bia", Papel: "Vendedor(a)", Acesso: "Ativo",
		Metas: []Meta{{Desde: "2026-10", Valor: 500}}})
	// Pedido cancelado também conta: continua no histórico. Grafia diferente na peça também.
	if _, err := s.SalvarPedido(ctx, Pedido{ID: "p1", ClienteID: "c1", VendedorID: "v1", Etapa: "Cancelado",
		Pecas: []Peca{{Modelo: " slim", Cor: "BRANCO", Tamanho: "M", Quantidade: 1}}}); err != nil {
		t.Fatal(err)
	}

	var ev ErrValidacao
	// Lote com um tamanho em pedido: nada sai, nem o tamanho livre.
	if err := s.RemoverEstoque(ctx, "m", "g"); !errors.As(err, &ev) || !strings.Contains(ev.Msg, "M (pedido #1)") {
		t.Fatalf("esperava recusa citando M e o pedido #1, veio %v", err)
	}
	if e, _ := s.Estado(ctx); len(e.Estoque) != 2 {
		t.Fatalf("recusa apagou parte do lote: %+v", e.Estoque)
	}
	if err := s.RemoverEstoque(ctx, "g"); err != nil {
		t.Fatalf("tamanho sem pedido deveria sair: %v", err)
	}

	if err := s.ExcluirVendedor(ctx, "v1"); !errors.As(err, &ev) || !strings.Contains(ev.Msg, "Inativo") {
		t.Fatalf("vendedor com pedido deveria ser recusado, veio %v", err)
	}
	if err := s.ExcluirCliente(ctx, "c1"); !errors.As(err, &ev) {
		t.Fatalf("cliente com pedido deveria ser recusado, veio %v", err)
	}

	// Sem pedido: sai, e as metas saem junto.
	if err := s.ExcluirVendedor(ctx, "v2"); err != nil {
		t.Fatal(err)
	}
	e, _ := s.Estado(ctx)
	for _, v := range e.Vendedores {
		if v.ID == "v2" {
			t.Fatal("v2 continua cadastrado")
		}
	}
	if len(e.Vendedores) != 1 || len(e.Clientes) != 1 {
		t.Fatalf("exclusão recusada apagou algo: %+v %+v", e.Vendedores, e.Clientes)
	}
}

func TestModelosPadraoEModeloNovo(t *testing.T) {
	ctx := context.Background()
	s := novaLoja(t)
	e, _ := s.Estado(ctx)
	if strings.Join(e.Modelos, "|") != strings.Join(ModelosPadrao, "|") {
		t.Fatalf("modelos padrão fora de ordem ou faltando: %v", e.Modelos)
	}

	// Modelo novo na entrada entra no fim da lista; grafia diferente do mesmo nome não duplica.
	s.EntradaEstoque(ctx, ItemEstoque{ID: "a", Modelo: "Jaleco Slim", Cor: "Branco", Tamanho: "M", Quantidade: 1})
	s.EntradaEstoque(ctx, ItemEstoque{ID: "b", Modelo: " jaleco slim", Cor: "Branco", Tamanho: "G", Quantidade: 1})
	s.EntradaEstoque(ctx, ItemEstoque{ID: "c", Modelo: "conjunto scrub", Cor: "Verde", Tamanho: "M", Quantidade: 1})
	e, _ = s.Estado(ctx)
	if len(e.Modelos) != len(ModelosPadrao)+1 || e.Modelos[len(e.Modelos)-1] != "Jaleco Slim" {
		t.Fatalf("esperava só \"Jaleco Slim\" a mais, no fim: %v", e.Modelos)
	}

	// Sem estoque, o modelo continua cadastrado.
	if err := s.RemoverEstoque(ctx, "a", "b"); err != nil {
		t.Fatal(err)
	}
	e, _ = s.Estado(ctx)
	if e.Modelos[len(e.Modelos)-1] != "Jaleco Slim" {
		t.Fatalf("modelo sumiu junto com o estoque: %v", e.Modelos)
	}
}
