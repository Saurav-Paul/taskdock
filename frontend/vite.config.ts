import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 8861,
    proxy: {
      "/api": "http://localhost:8860",
      "/mcp": "http://localhost:8860",
    },
  },
  build: {
    outDir: "dist",
  },
});
