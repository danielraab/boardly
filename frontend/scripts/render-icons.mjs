// Renders the home-screen icons in public/icons from the source SVGs in icons/
// with headless Chromium. Run `pnpm icons` after changing the drawing; the PNGs
// are committed, so builds don't need Chromium. CHROMIUM overrides the binary.
import { execFileSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("..", import.meta.url));
const chromium = process.env.CHROMIUM ?? "chromium";

const icons = [
	{ source: "app-icon.svg", out: "icon-192.png", size: 192 },
	{ source: "app-icon.svg", out: "icon-512.png", size: 512 },
	{ source: "app-icon-maskable.svg", out: "icon-maskable-512.png", size: 512 },
	{ source: "app-icon.svg", out: "apple-touch-icon.png", size: 180 },
];

const work = mkdtempSync(join(tmpdir(), "famodash-icons-"));
try {
	for (const { source, out, size } of icons) {
		const svg = readFileSync(join(root, "icons", source), "utf8");
		const page = join(work, "icon.html");
		writeFileSync(
			page,
			`<!doctype html><style>html,body{margin:0}svg{display:block;width:${size}px;height:${size}px}</style>${svg}`,
		);
		execFileSync(
			chromium,
			[
				"--headless",
				"--disable-gpu",
				"--hide-scrollbars",
				"--force-device-scale-factor=1",
				`--window-size=${size},${size}`,
				`--screenshot=${join(root, "public", "icons", out)}`,
				`file://${page}`,
			],
			// Chromium logs GPU and driver noise even when it succeeds.
			{ stdio: "ignore" },
		);
		console.log(`public/icons/${out} (${size}×${size})`);
	}
} finally {
	rmSync(work, { recursive: true, force: true });
}
