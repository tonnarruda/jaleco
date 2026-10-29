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

## Deploy (Render + Vercel)

- **Render:** banco Postgres e API (`render.yaml` + `backend/Dockerfile`).
- **Vercel:** frontend (`frontend/vercel.json`). O Vercel repassa `/api/*` para a API, então
  o navegador vê um endereço só e o cookie de sessão funciona.

1. **Render:** New → Blueprint → este repositório. Ele cria `jaleco-db` e `jaleco-api`.
   - O plano gratuito do Postgres **expira e apaga os dados**: troque para pago antes de
     cadastrar dados reais.
2. Copie o endereço da API no Render (ex.: `https://jaleco-api.onrender.com`). Se for
   diferente do que está em `frontend/vercel.json`, ajuste o `destination` e faça commit.
3. **Vercel:** Add New → Project → este repositório, com **Root Directory = `frontend`**. O
   resto o Vercel detecta (Vite, `npm run build`, saída `dist`).
4. Abra o endereço do Vercel e crie o administrador **na hora** ("Gerar usuário
   Administrador"): até isso acontecer, quem abrir o site primeiro vira admin.

**Link de acesso pelo servidor em produção:** rode o comando na sua máquina apontando para o
banco do Render (use a *External Database URL* do painel, com `?sslmode=require`):

```sh
cd backend
DATABASE_URL='postgres://…?sslmode=require' APP_URL='https://<seu-app>.vercel.app' \
  go run ./cmd/server link-acesso dona@loja.com
```

## Produção: limitações conhecidas

- O limite de tentativas de login é **por conta (e-mail)**, não por IP: atrás do Vercel e do
  Render o IP da conexão é do proxy. Quem errar a senha de uma conta 5 vezes de propósito a
  trava por 15 minutos.
- A contagem de tentativas fica em memória: reiniciar a API (ou ela dormir no plano gratuito)
  zera a contagem.
