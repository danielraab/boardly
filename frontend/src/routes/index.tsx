import { createFileRoute } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { fullPage } from "#/components/pageWidths";
import { useMe } from "#/features/auth/session";
import { validateDashboardSearch } from "#/features/dashboards/DashboardView";
import { Home } from "#/features/home/Home";
import { Landing } from "#/features/landing/Landing";

// `edit` opens edit mode when / shows the default dashboard.
export const Route = createFileRoute("/")({
	validateSearch: validateDashboardSearch,
	component: Index,
});

/** Home for signed-in users, the landing page otherwise; neither while loading. */
function Index() {
	const { data: me, isPending } = useMe();
	if (isPending) return <PageSkeleton />;
	return me ? <Home user={me} /> : <Landing />;
}

function PageSkeleton() {
	const { t } = useTranslation();
	return (
		<div
			aria-busy="true"
			className={`${fullPage} space-y-4 pt-12 motion-safe:animate-pulse`}
		>
			<span className="sr-only">{t("common.loading")}</span>
			<div className="h-14 w-2/3 max-w-md rounded-md bg-surface" />
			<div className="h-5 w-1/2 max-w-sm rounded-md bg-surface" />
			<div className="mt-10 aspect-video rounded-xl bg-surface" />
		</div>
	);
}
