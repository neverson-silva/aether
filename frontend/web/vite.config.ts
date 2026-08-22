import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import path from "path";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "VITE_");
  const apiTarget = env.VITE_API_TARGET || "";
  return {
    plugins: [
      tanstackRouter({
        routesDirectory: "./src/routes",
        generatedRouteTree: "./src/routeTree.gen.ts",
      }),
      react(),
      tailwindcss(),
    ],
    build: { outDir: "dist", emptyOutDir: true },
    server: {
      port: 5173,
      allowedHosts: true,
      proxy: { "/api": { target: apiTarget || "http://127.0.0.1:8080", ws: true } },
      fs: { allow: [path.resolve(__dirname, "..")] },
    },
    resolve: {
      alias: {
        "@": path.resolve(__dirname, "./src"),
        react: path.resolve(__dirname, "node_modules/react"),
        "react-dom": path.resolve(__dirname, "node_modules/react-dom"),
        "@aether/design-system/styles.css": path.resolve(__dirname, "../aether_ds/src/styles.css"),
        "@aether/design-system": path.resolve(__dirname, "../aether_ds/src"),
      },
      dedupe: ["react", "react-dom"],
    },
  };
});
