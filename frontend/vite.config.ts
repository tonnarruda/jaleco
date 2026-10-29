import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    // Em desenvolvimento, a API Go roda à parte (backend/, porta 8080).
    // API_URL troca o destino (ex.: uma segunda API de teste em outra porta).
    proxy: { "/api": process.env.API_URL ?? "http://localhost:8080" },
  },
});
