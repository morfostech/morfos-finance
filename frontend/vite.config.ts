import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

// Dev server proxies API and uploads to the Go backend so the SPA stays
// same-origin in development.
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, ".", "VITE_");
  const apiTarget = env.VITE_API_PROXY_TARGET || "http://localhost:8080";

  return {
    base: "/finance/",
    plugins: [react()],
    server: {
      port: 5173,
      proxy: {
        "/finance/api": {
          target: apiTarget,
          rewrite: (path) => path.replace(/^\/finance/, ""),
        },
        "/finance/uploads": {
          target: apiTarget,
          rewrite: (path) => path.replace(/^\/finance/, ""),
        },
      },
    },
  };
});
