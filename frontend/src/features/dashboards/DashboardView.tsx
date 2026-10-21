import { LayoutGrid } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { DashboardViewData } from "./api";
import { useShowDashboard } from "./currentDashboard";

/** A dashboard page; it has no content yet, only its title and a placeholder. */
export function DashboardView({ view }: { view: DashboardViewData }) {
	const { t } = useTranslation();
	useShowDashboard(view);

	return (
		<section className="pt-12">
			<h1 className="text-3xl font-semibold tracking-tight break-words sm:text-4xl">
				{view.dashboard.title}
			</h1>
			<div className="mt-10 rounded-xl border border-dashed border-line bg-surface px-6 py-12 text-center">
				<LayoutGrid
					className="mx-auto size-8 text-muted"
					strokeWidth={1.75}
					aria-hidden
				/>
				<h2 className="mt-4 text-lg font-semibold">
					{t("dashboards.view.emptyTitle")}
				</h2>
				<p className="mx-auto mt-1 max-w-[45ch] text-muted">
					{t("dashboards.view.emptyText")}
				</p>
			</div>
		</section>
	);
}
