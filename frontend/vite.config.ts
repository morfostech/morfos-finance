import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Dev server proxies API and uploads to the Go backend so the SPA stays
// same-origin in development.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/api": "http://localhost:8080",
      "/uploads": "http://localhost:8080",
    },
  },
});
