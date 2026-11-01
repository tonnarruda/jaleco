import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { defineConfig, type Plugin } from "vite";
import react from "@vitejs/plugin-react";

const openapi = fileURLToPath(new URL("../backend/openapi.yaml", import.meta.url));

const paginaSwagger = `<!doctype html>
<html lang="pt-BR">
  <head>
    <meta charset="UTF-8" />
    <title>API · I Love Jalecos</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.17.14/swagger-ui.css" />
  </head>
  <body>
    <div id="swagger"></div>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.17.14/swagger-ui-bundle.js"></script>
    <script>
      SwaggerUIBundle({ url: "/docs/openapi.yaml", dom_id: "#swagger", withCredentials: true });
    </script>
  </body>
</html>`;

// Swagger da API em /docs, só no `npm run dev`: não entra no build nem vai para o
// Vercel. No mesmo endereço do app, o "Try it out" passa pelo proxy de /api e usa o
// cookie da sessão (entre pelo POST /api/login ou pelo próprio app antes).
const swagger: Plugin = {
  name: "swagger-dev",
  apply: "serve",
  configureServer(server) {
    server.middlewares.use((req, res, next) => {
      if (req.url === "/docs/openapi.yaml") {
        res.setHeader("Content-Type", "text/yaml; charset=utf-8");
        res.end(readFileSync(openapi)); // lido a cada pedido: editou o yaml, recarregou a página
        return;
      }
      if (req.url === "/docs" || req.url === "/docs/") {
        res.setHeader("Content-Type", "text/html; charset=utf-8");
        res.end(paginaSwagger);
        return;
      }
      next();
    });
  },
};

export default defineConfig({
  plugins: [react(), swagger],
  server: {
    port: 5173,
    // Em desenvolvimento, a API Go roda à parte (backend/, porta 8080).
    // API_URL troca o destino (ex.: uma segunda API de teste em outra porta).
    proxy: { "/api": process.env.API_URL ?? "http://localhost:8080" },
  },
});
