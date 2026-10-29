// Package loja guarda os dados da I Love Jalecos e aplica as regras que não
// podem depender do navegador: número do pedido, baixa de estoque e histórico
// de metas.
package loja

import (
	"errors"
	"strings"
)

// Os nomes JSON acompanham os tipos do frontend (frontend/src/types.ts).

type Cliente struct {
	ID              string `json:"id"`
	Nome            string `json:"nome"`
	Whatsapp        string `json:"whatsapp"`
	Email           string `json:"email"`
	Profissao       string `json:"profissao"`
	Faculdade       string `json:"faculdade"`
	Instagram       string `json:"instagram"`
	Cidade          string `json:"cidade"`
	Aniversario     string `json:"aniversario"`
	TamanhoHabitual string `json:"tamanhoHabitual"`
	ComoConheceu    string `json:"comoConheceu"`
	Endereco        string `json:"endereco"`
	Observacoes     string `json:"observacoes"`
}

type Peca struct {
	ID               string  `json:"id"`
	Modelo           string  `json:"modelo"`
	Cor              string  `json:"cor"`
	Tamanho          string  `json:"tamanho"`
	Quantidade       int     `json:"quantidade"`
	NomeBordado      string  `json:"nomeBordado"`
	ProfissaoBordado string  `json:"profissaoBordado"`
	CorLinha         string  `json:"corLinha"`
	Adicional        string  `json:"adicional"`
	PrecoUnitario    float64 `json:"precoUnitario"`
}

type Pedido struct {
	ID                string  `json:"id"`
	Numero            int     `json:"numero"`
	ClienteID         string  `json:"clienteId"`
	VendedorID        string  `json:"vendedorId"`
	Canal             string  `json:"canal"`
	DataPedido        string  `json:"dataPedido"`
	Pecas             []Peca  `json:"pecas"`
	Etapa             string  `json:"etapa"`
	PrazoEntrega      string  `json:"prazoEntrega"`
	CodigoRastreio    string  `json:"codigoRastreio"`
	Observacoes       string  `json:"observacoes"`
	FormaPagamento    string  `json:"formaPagamento"`
	SituacaoPagamento string  `json:"situacaoPagamento"`
	Desconto          float64 `json:"desconto"`
	Frete             float64 `json:"frete"`
	ValorPago         float64 `json:"valorPago"`
}

type ItemEstoque struct {
	ID         string  `json:"id"`
	Modelo     string  `json:"modelo"`
	Cor        string  `json:"cor"`
	Tamanho    string  `json:"tamanho"`
	Quantidade int     `json:"quantidade"`
	Minimo     int     `json:"minimo"`
	PrecoVenda float64 `json:"precoVenda"` // 0 = sem preço cadastrado
}

// Meta vale a partir de Desde (AAAA-MM) até a próxima vigência.
type Meta struct {
	Desde string  `json:"desde"`
	Valor float64 `json:"valor"`
}

type Vendedor struct {
	ID       string  `json:"id"`
	Nome     string  `json:"nome"`
	Whatsapp string  `json:"whatsapp"`
	Papel    string  `json:"papel"`
	Acesso   string  `json:"acesso"`
	Comissao float64 `json:"comissao"`
	Metas    []Meta  `json:"metas"`
	// Email é o login. PossuiLogin só sai do servidor: diz se a pessoa já criou a senha.
	Email       string `json:"email"`
	PossuiLogin bool   `json:"possuiLogin"`
}

type Estado struct {
	Clientes   []Cliente     `json:"clientes"`
	Pedidos    []Pedido      `json:"pedidos"`
	Estoque    []ItemEstoque `json:"estoque"`
	Vendedores []Vendedor    `json:"vendedores"`
	Modelos    []string      `json:"modelos"` // na ordem de cadastro: os padrão primeiro
}

// ModelosPadrao entram no banco toda vez que o servidor sobe (sem duplicar).
var ModelosPadrao = []string{
	"Jaleco Feminino Acinturado",
	"Jaleco Masculino Clássico",
	"Jaleco Manga Curta",
	"Jaleco Estudante",
	"Conjunto Scrub",
	"Jaleco Vet com Bolso Patinha",
}

var ErrNaoEncontrado = errors.New("registro não encontrado")

// ErrValidacao carrega a mensagem que pode ir direto para a tela.
type ErrValidacao struct{ Msg string }

func (e ErrValidacao) Error() string { return e.Msg }

// Orçamento ainda não é venda e cancelado deixou de ser: nenhum dos dois segura peça.
func consomeEstoque(etapa string) bool {
	return etapa != "Cancelado" && etapa != "Orçamento"
}

// chaveVariacao espelha chaveVariacao de frontend/src/lib/pedido.ts.
func chaveVariacao(modelo, cor, tamanho string) string {
	partes := []string{modelo, cor, tamanho}
	for i, p := range partes {
		partes[i] = strings.ToLower(strings.TrimSpace(p))
	}
	return strings.Join(partes, "|")
}
