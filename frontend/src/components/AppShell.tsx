import { Link } from "@tanstack/react-router";
import type { ReactNode } from "react";
import { LanguageSwitcher } from "./LanguageSwitcher";
import { UserMenu } from "./UserMenu";

export function AppShell({ children }: { children: ReactNode }) {
	return (
		<div className="flex min-h-screen flex-col">
			<header className="mx-auto flex w-full max-w-6xl items-center justify-between px-4 py-3 sm:px-6">
				<Link
					to="/"
					className="rounded-md text-lg font-semibold tracking-tight"
				>
					Famodash
				</Link>
				<div className="flex items-center gap-1">
					<LanguageSwitcher />
					<UserMenu />
				</div>
			</header>
			<main className="mx-auto w-full max-w-6xl flex-1 px-4 pb-16 sm:px-6">
				{children}
			</main>
		</div>
	);
}
