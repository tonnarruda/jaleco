package api

import (
	"bufio"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestOpenAPICobreTodasAsRotas compara as rotas registradas em New (api.go) com as
// documentadas em backend/openapi.yaml, nos dois sentidos: rota nova sem
// documentação, ou documentação de rota que saiu, faz o teste falhar.
func TestOpenAPICobreTodasAsRotas(t *testing.T) {
	codigo := rotasDoCodigo(t, "api.go")
	doc := rotasDoOpenAPI(t, "../../openapi.yaml")
	if len(codigo) == 0 || len(doc) == 0 {
		t.Fatalf("nenhuma rota lida: código=%d, openapi=%d", len(codigo), len(doc))
	}
	for r := range codigo {
		if !doc[r] {
			t.Errorf("rota sem documentação no openapi.yaml: %s", r)
		}
	}
	for r := range doc {
		if !codigo[r] {
			t.Errorf("openapi.yaml documenta rota que não existe em api.go: %s", r)
		}
	}
	if t.Failed() {
		t.Logf("rotas no código: %v", ordenadas(codigo))
	}
}

var rotaRegistrada = regexp.MustCompile(`mux\.Handle(?:Func)?\("([A-Z]+) (/api/[^"]*)"`)

func rotasDoCodigo(t *testing.T, arquivo string) map[string]bool {
	t.Helper()
	b, err := os.ReadFile(arquivo)
	if err != nil {
		t.Fatal(err)
	}
	rotas := map[string]bool{}
	for _, m := range rotaRegistrada.FindAllStringSubmatch(string(b), -1) {
		rotas[m[1]+" "+m[2]] = true
	}
	return rotas
}

// rotasDoOpenAPI lê só a estrutura de paths/métodos, pela indentação do arquivo
// (caminho com 2 espaços, método com 4): sem depender de biblioteca de YAML.
func rotasDoOpenAPI(t *testing.T, arquivo string) map[string]bool {
	t.Helper()
	f, err := os.Open(arquivo)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	metodos := map[string]bool{"get": true, "post": true, "put": true, "patch": true, "delete": true}
	rotas := map[string]bool{}
	emPaths, caminho := false, ""
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		linha := strings.TrimRight(sc.Text(), " ")
		switch {
		case linha == "paths:":
			emPaths = true
		case !emPaths || linha == "":
		case !strings.HasPrefix(linha, " "):
			emPaths = false // próxima seção do topo (components)
		case strings.HasPrefix(linha, "  /") && strings.HasSuffix(linha, ":"):
			caminho = strings.TrimSuffix(strings.TrimSpace(linha), ":")
		case strings.HasPrefix(linha, "    ") && !strings.HasPrefix(linha, "     "):
			if m := strings.TrimSuffix(strings.TrimSpace(linha), ":"); metodos[m] {
				rotas[strings.ToUpper(m)+" "+caminho] = true
			}
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return rotas
}

func ordenadas(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
