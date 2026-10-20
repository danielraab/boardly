import { LayoutGrid } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { User } from "#/features/auth/session";

export function Home({ user }: { user: User }) {
	const { t } = useTranslation();

	return (
		<section className="pt-12">
			<h1 className="text-3xl font-semibold tracking-tight break-words sm:text-4xl">
				{t("home.greeting", { name: user.name ?? user.email })}
			</h1>
			<div className="mt-10 rounded-xl border border-dashed border-line bg-surface px-6 py-12 text-center">
				<LayoutGrid
					className="mx-auto size-8 text-muted"
					strokeWidth={1.75}
					aria-hidden
				/>
				<h2 className="mt-4 text-lg font-semibold">
					{t("home.placeholderTitle")}
				</h2>
				<p className="mx-auto mt-1 max-w-[45ch] text-muted">
					{t("home.placeholderText")}
				</p>
			</div>
		</section>
	);
}
