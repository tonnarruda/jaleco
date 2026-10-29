# I Love Jalecos — Gestão

- `frontend/`: React + Vite
- `backend/`: API em Go (`/api`) com Postgres

## Rodar em desenvolvimento

```sh
docker compose up -d                    # Postgres na porta 5433
cd backend && go run ./cmd/server       # API em :8080, cria as tabelas sozinha
cd frontend && npm run dev              # app em :5173, repassa /api para :8080
```

A API lê `DATABASE_URL` (padrão `postgres://jaleco:jaleco@localhost:5433/jaleco?sslmode=disable`).

## Testes

```sh
cd backend && go test ./...
```

Usam o mesmo Postgres, cada teste num schema temporário que é apagado no fim.

## Acesso

- **Primeiro administrador:** enquanto não houver nenhum, a tela de entrada mostra o botão
  "Gerar usuário Administrador", que leva ao cadastro (nome, e-mail e senha). Com o admin
  criado, o botão some e a rota passa a recusar. **Quem chegar primeiro vira admin:** ao
  publicar, crie o admin antes de divulgar o endereço.
- **Equipe:** o admin cadastra a pessoa com e-mail e clica no ícone de link na Equipe; o
  link vai pelo WhatsApp e a pessoa cria a própria senha. Esqueceu a senha: o mesmo ícone
  gera um link de nova senha.
- **Papéis:** Vendedor(a) vê e edita só os próprios pedidos e o próprio cadastro (clientes,
  estoque e modelos são de todos); Gerente vê tudo, menos gerir a equipe; Administrador vê tudo.
  A regra vale no servidor, não só na tela.
- **Sessão:** cookie HttpOnly de 30 dias, guardado no banco. Marcar a pessoa como Inativa
  derruba o acesso na hora.
- **Ninguém consegue gerar o link pela tela** (ex.: a única administradora esqueceu a senha):
  quem tem acesso ao servidor roda `cd backend && go run ./cmd/server link-acesso <e-mail>`
  e manda o link impresso para a pessoa. `APP_URL` define o endereço do link
  (padrão `http://localhost:5173`).

Variáveis da API: `DATABASE_URL` e `COOKIE_SEGURO=1` (obrigatório em produção, com HTTPS).

## Produção (pendente)

- Servir por **HTTPS** e ligar `COOKIE_SEGURO=1`; sem isso o cookie de sessão trafega aberto.
- O limite de tentativas de login usa o IP da conexão: atrás de um proxy reverso, todos
  chegam com o mesmo IP. Ajustar quando houver proxy.
