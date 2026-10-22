import { writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import tailwindcss from "@tailwindcss/vite";
import { devtools } from "@tanstack/devtools-vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import viteReact from "@vitejs/plugin-react";
import { defineConfig, type Plugin } from "vite";

// The Go server embeds this directory (backend/internal/web), see design.md.
const outDir = fileURLToPath(
	new URL("../backend/internal/web/dist", import.meta.url),
);

// emptyOutDir wipes the committed .gitkeep; put it back so `go build` works on a clean checkout.
function keepGitkeep(): Plugin {
	return {
		name: "boardly-keep-gitkeep",
		apply: "build",
		closeBundle() {
			writeFileSync(`${outDir}/.gitkeep`, "");
		},
	};
}

const config = defineConfig({
	resolve: { tsconfigPaths: true },
	plugins: [
		devtools(),
		tailwindcss(),
		tanstackRouter({ target: "react", autoCodeSplitting: true }),
		viteReact(),
		keepGitkeep(),
	],
	build: {
		outDir,
		emptyOutDir: true,
	},
	server: {
		proxy: {
			"/api": "http://localhost:8080",
		},
	},
});

export default config;
