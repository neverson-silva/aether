import { defineConfig, loadEnv, type Plugin } from "vite";
import fs from "fs";
import type { IncomingMessage, ServerResponse } from "http";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import path from "path";

function monacoAssets(): Plugin {
  let outputDirectory = "";
  const sourceDirectory = path.resolve(import.meta.dirname, "node_modules/monaco-editor/min/vs");

  const serve = (request: IncomingMessage, response: ServerResponse, next: (error?: unknown) => void) => {
    const requestPath = decodeURIComponent((request.url ?? "/").split("?")[0]).replace(/^\/+/, "");
    if (!requestPath || requestPath.includes("..")) {
      next();
      return;
    }
    const filePath = path.join(sourceDirectory, requestPath);
    if (!fs.existsSync(filePath) || !fs.statSync(filePath).isFile()) {
      next();
      return;
    }
    const extension = path.extname(filePath);
    const contentType = extension === ".js" ? "application/javascript" : extension === ".json" ? "application/json" : "application/octet-stream";
    response.statusCode = 200;
    response.setHeader("Content-Type", contentType);
    fs.createReadStream(filePath).pipe(response);
  };

  return {
    name: "aether-monaco-assets",
    configResolved(config) {
      outputDirectory = config.build.outDir;
    },
    configureServer(server) {
      server.middlewares.use("/monaco/vs", serve);
    },
    configurePreviewServer(server) {
      server.middlewares.use("/monaco/vs", serve);
    },
    closeBundle() {
      fs.cpSync(sourceDirectory, path.resolve(outputDirectory, "monaco/vs"), { recursive: true });
    },
  };
}

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "VITE_");
  const apiTarget = env.VITE_API_TARGET || "";
  return {
    plugins: [
      monacoAssets(),
      tanstackRouter({
        autoCodeSplitting: true,
      }),
      react(),
      tailwindcss(),
    ],
    build: { outDir: "dist", emptyOutDir: true, },
    server: {
      port: 5173,
      allowedHosts: true,
      proxy: { "/api/": { target: apiTarget || "http://127.0.0.1:8080", ws: true } },
      fs: { allow: [path.resolve(import.meta.dirname, "..")] },
    },
    resolve: {
      alias: {
        "@": path.resolve(import.meta.dirname, "./src"),
        react: path.resolve(import.meta.dirname, "node_modules/react"),
        "react-dom": path.resolve(import.meta.dirname, "node_modules/react-dom"),
        "@aether/design-system/styles.css": path.resolve(import.meta.dirname, "../aether_ds/src/styles.css"),
        "@aether/design-system": path.resolve(import.meta.dirname, "../aether_ds/src"),
      },
      dedupe: ["react", "react-dom"],
    },
  };
});
