import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { LayoutGrid } from "lucide-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { textInput } from "#/components/dialogStyles";
import { primaryButton } from "#/features/auth/LoginPage";
import type { User } from "#/features/auth/session";
import {
	type Dashboard,
	dashboardsQueryKey,
	dashboardsQueryOptions,
	dashboardViewQueryOptions,
	updateDashboard,
} from "#/features/dashboards/api";
import { DashboardView } from "#/features/dashboards/DashboardView";

/** The signed-in start page: the default dashboard, or how to get one. */
export function Home({ user }: { user: User }) {
	const { t } = useTranslation();
	const list = useQuery(dashboardsQueryOptions);

	if (list.isError) return <GenericError />;
	if (!list.data) return null;

	const preferred = list.data.find((d) => d.isDefault);
	if (preferred) return <DefaultDashboard user={user} dashboard={preferred} />;

	return (
		<Greeting user={user}>
			{list.data.length === 0 ? (
				<>
					<h2 className="mt-4 text-lg font-semibold">
						{t("home.createFirst.title")}
					</h2>
					<p className="mx-auto mt-1 max-w-[45ch] text-muted">
						{t("home.createFirst.text")}
					</p>
					<Link
						to="/dashboards"
						search={{ create: true }}
						className={`${primaryButton} mt-6 py-2.5`}
					>
						{t("home.createFirst.action")}
					</Link>
				</>
			) : (
				<SelectDefault dashboards={list.data} />
			)}
		</Greeting>
	);
}

function DefaultDashboard({
	user,
	dashboard,
}: {
	user: User;
	dashboard: Dashboard;
}) {
	// The same lookup as at /<user-slug>/<dashboard-slug>, so both pages match.
	const view = useQuery(
		dashboardViewQueryOptions(user.slug, dashboard.slug, user.id),
	);
	if (view.isError) return <GenericError />;
	if (!view.data) return null;
	return <DashboardView view={view.data} />;
}

function SelectDefault({ dashboards }: { dashboards: Dashboard[] }) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const choose = useMutation({
		mutationFn: (id: string) => updateDashboard(id, { isDefault: true }),
		onSuccess: () =>
			queryClient.invalidateQueries({ queryKey: dashboardsQueryKey }),
	});

	return (
		<>
			<h2 className="mt-4 text-lg font-semibold">
				{t("home.selectDefault.title")}
			</h2>
			<p className="mx-auto mt-1 max-w-[45ch] text-muted">
				{t("home.selectDefault.text")}
			</p>
			<div className="mx-auto mt-6 max-w-xs text-left">
				<label htmlFor="default-dashboard" className="sr-only">
					{t("home.selectDefault.label")}
				</label>
				<select
					id="default-dashboard"
					value=""
					disabled={choose.isPending}
					onChange={(e) => choose.mutate(e.target.value)}
					className={textInput}
				>
					<option value="" disabled>
						{t("home.selectDefault.placeholder")}
					</option>
					{dashboards.map((d) => (
						<option key={d.id} value={d.id}>
							{d.title}
						</option>
					))}
				</select>
				{choose.isError && (
					<p role="alert" className="mt-2 text-sm text-red-700">
						{t("common.genericError")}
					</p>
				)}
			</div>
		</>
	);
}

function Greeting({ user, children }: { user: User; children: ReactNode }) {
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
				{children}
			</div>
		</section>
	);
}

function GenericError() {
	const { t } = useTranslation();
	return (
		<p role="alert" className="pt-12 text-sm text-red-700">
			{t("common.genericError")}
		</p>
	);
}
