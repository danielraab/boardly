import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";
import { languages } from "#/i18n";

type Icon = { src: string; sizes: string; type: string; purpose: string };

const manifests = Object.fromEntries(
	languages.map(({ code }) => [
		code,
		JSON.parse(readFileSync(`public/manifest.${code}.webmanifest`, "utf8")),
	]),
);

const html = new DOMParser().parseFromString(
	readFileSync("index.html", "utf8"),
	"text/html",
);

/** Width and height of a PNG served from public/, read from its IHDR chunk. */
function pngSize(src: string): string {
	const data = readFileSync(`public${src}`);
	expect(data.subarray(1, 4).toString("latin1"), src).toBe("PNG");
	return `${data.readUInt32BE(16)}x${data.readUInt32BE(20)}`;
}

describe.each(languages)("manifest.$code.webmanifest", ({ code }) => {
	const manifest = manifests[code];

	it("describes a standalone app that starts at /", () => {
		expect(manifest).toMatchObject({
			id: "/",
			start_url: "/",
			scope: "/",
			display: "standalone",
			lang: code,
		});
		expect(manifest.name).toBe("Famodash");
		expect(manifest.description).toEqual(expect.any(String));
	});

	it("lists PNG icons of the declared sizes", () => {
		const purposes = manifest.icons.map(
			(icon: Icon) => `${icon.purpose} ${icon.sizes}`,
		);
		expect(purposes).toEqual([
			"any 192x192",
			"any 512x512",
			"maskable 512x512",
		]);
		for (const icon of manifest.icons as Icon[]) {
			expect(icon.type).toBe("image/png");
			expect(pngSize(icon.src)).toBe(icon.sizes);
		}
	});
});

it("only translates the texts between the manifests", () => {
	const translated = ["name", "short_name", "description", "lang"];
	const untranslated = (manifest: Record<string, unknown>) =>
		Object.fromEntries(
			Object.entries(manifest)
				.filter(([key]) => !translated.includes(key))
				.map(([key, value]) =>
					key === "shortcuts"
						? [key, (value as { url: string }[]).map((s) => s.url)]
						: [key, value],
				),
		);
	expect(untranslated(manifests.de)).toEqual(untranslated(manifests.en));
	expect(manifests.de.description).not.toBe(manifests.en.description);
});

it("shares links into the Save link page", () => {
	expect(manifests.en.share_target).toEqual({
		action: "/share",
		method: "GET",
		enctype: "application/x-www-form-urlencoded",
		params: { title: "title", text: "text", url: "url" },
	});
});

it.each([
	["en", ["Add link", "Favourites", "Dashboards"]],
	["de", ["Link hinzufügen", "Favoriten", "Dashboards"]],
])("names the shortcuts in %s", (code, names) => {
	expect(manifests[code].shortcuts).toEqual([
		{ name: names[0], url: "/share" },
		{ name: names[1], url: "/favourites" },
		{ name: names[2], url: "/dashboards" },
	]);
});

it("declares an Apple touch icon of 180×180", () => {
	const href = html
		.querySelector('link[rel="apple-touch-icon"]')
		?.getAttribute("href");
	expect(pngSize(href ?? "")).toBe("180x180");
});
