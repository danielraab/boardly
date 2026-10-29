import { useThemeMode } from "#/theme";
import type { AppTokenApp } from "./api";
import karakeep from "./logos/karakeep.svg";
import karakeepDark from "./logos/karakeep-dark.svg";
import linkwarden from "./logos/linkwarden.png";
import memos from "./logos/memos.svg";
import memosDark from "./logos/memos-dark.svg";

/** The apps' official logos, see logos/README.md for their sources. */
export const appLogos: Record<AppTokenApp, { light: string; dark?: string }> = {
	karakeep: { light: karakeep, dark: karakeepDark },
	linkwarden: { light: linkwarden },
	memos: { light: memos, dark: memosDark },
};

/** Decorative: the app name is always shown as text next to it. */
export function AppLogo({ app }: { app: AppTokenApp }) {
	const mode = useThemeMode();
	const { light, dark } = appLogos[app];
	const src = mode === "dark" ? (dark ?? light) : light;
	return <img src={src} alt="" className="size-8 shrink-0" />;
}
