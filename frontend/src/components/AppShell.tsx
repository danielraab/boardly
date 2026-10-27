import { Link } from "@tanstack/react-router";
import { Plus } from "lucide-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { useMe } from "#/features/auth/session";
import { CurrentDashboardProvider } from "#/features/dashboards/currentDashboard";
import { DashboardSwitcher } from "#/features/dashboards/DashboardSwitcher";
import { appTitle } from "#/lib/appTitle";
import { AppIcon } from "./AppIcon";
import { headerButton } from "./buttonStyles";
import { LanguageSwitcher } from "./LanguageSwitcher";
import { ThemeSwitcher } from "./ThemeSwitcher";
import { UserMenu } from "./UserMenu";

/** Shared by header and main so the logo lines up with page content. */
const container = "mx-auto w-full max-w-screen-2xl px-4 sm:px-6";

export function AppShell({ children }: { children: ReactNode }) {
	return (
		<CurrentDashboardProvider>
			<div className="flex min-h-screen flex-col">
				<header
					className={`${container} flex items-center justify-between py-3`}
				>
					<div className="flex min-w-0 items-center gap-1.5">
						<Link
							to="/"
							className="flex min-w-0 items-center gap-2 rounded-md text-lg font-semibold tracking-tight"
						>
							<AppIcon className="size-6 shrink-0" />
							<span className="truncate">{appTitle()}</span>
						</Link>
						<DashboardSwitcher />
					</div>
					<div className="flex items-center gap-1">
						<AddLinkButton />
						<ThemeSwitcher />
						<LanguageSwitcher />
						<UserMenu />
					</div>
				</header>
				<main className={`${container} flex-1 pb-16`}>{children}</main>
			</div>
		</CurrentDashboardProvider>
	);
}

/** Opens the "Save link" page; only for signed-in users. */
function AddLinkButton() {
	const { t } = useTranslation();
	const { data: me } = useMe();
	if (!me) return null;
	return (
		<Link
			to="/share"
			className={headerButton}
			aria-label={t("shell.addLink")}
			title={t("shell.addLink")}
		>
			<Plus className="size-5" aria-hidden />
		</Link>
	);
}
