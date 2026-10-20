#!/usr/bin/env node
// Checks that German and English translations are complete:
// - frontend/src/locales/{de,en}.json have the same keys, no empty values and
//   the same {{placeholders}} per key;
// - backend/internal/mail/templates/{de,en}/ contain the same template files.
// Uses only Node built-ins, so it runs without installing dependencies.
// Usage: node .github/scripts/check-i18n.mjs (from the repository root)

import { readdirSync, readFileSync } from "node:fs";
import { join } from "node:path";

const LOCALES = ["en", "de"];
const LOCALES_DIR = "frontend/src/locales";
const MAIL_TEMPLATES_DIR = "backend/internal/mail/templates";

const errors = [];

function flatten(obj, prefix = "") {
	return Object.entries(obj).flatMap(([k, v]) =>
		typeof v === "object" && v !== null
			? flatten(v, `${prefix}${k}.`)
			: [[`${prefix}${k}`, v]],
	);
}

function placeholders(value) {
	return [...String(value).matchAll(/\{\{\s*([^}\s,]+)[^}]*\}\}/g)]
		.map((m) => m[1])
		.sort()
		.join(",");
}

function listFiles(dir, prefix = "") {
	return readdirSync(dir, { withFileTypes: true }).flatMap((e) =>
		e.isDirectory()
			? listFiles(join(dir, e.name), `${prefix}${e.name}/`)
			: [`${prefix}${e.name}`],
	);
}

// Frontend catalogs
const catalogs = Object.fromEntries(
	LOCALES.map((l) => [
		l,
		new Map(
			flatten(JSON.parse(readFileSync(join(LOCALES_DIR, `${l}.json`), "utf8"))),
		),
	]),
);

for (const l of LOCALES) {
	for (const [key, value] of catalogs[l]) {
		if (typeof value !== "string" || value.trim() === "") {
			errors.push(`${LOCALES_DIR}/${l}.json: "${key}" is empty or not a string`);
		}
		for (const other of LOCALES.filter((o) => o !== l)) {
			if (!catalogs[other].has(key)) {
				errors.push(`${LOCALES_DIR}/${other}.json: missing key "${key}" (present in ${l})`);
			} else if (l < other && placeholders(value) !== placeholders(catalogs[other].get(key))) {
				errors.push(
					`"${key}": placeholders differ between ${l} [${placeholders(value)}] and ${other} [${placeholders(catalogs[other].get(key))}]`,
				);
			}
		}
	}
}

// Backend mail templates
const templates = Object.fromEntries(
	LOCALES.map((l) => [l, new Set(listFiles(join(MAIL_TEMPLATES_DIR, l)))]),
);

for (const l of LOCALES) {
	for (const file of templates[l]) {
		for (const other of LOCALES.filter((o) => o !== l)) {
			if (!templates[other].has(file)) {
				errors.push(`${MAIL_TEMPLATES_DIR}/${other}/: missing ${file} (present in ${l})`);
			}
		}
	}
}

if (errors.length > 0) {
	console.error(`i18n check failed with ${errors.length} problem(s):`);
	for (const e of errors) console.error(`  - ${e}`);
	process.exit(1);
}

console.log(
	`i18n check passed: ${catalogs.en.size} frontend keys, ${templates.en.size} mail templates in ${LOCALES.join(", ")}.`,
);
