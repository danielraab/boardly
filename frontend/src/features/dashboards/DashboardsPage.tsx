import { useQuery } from "@tanstack/react-query";
import { Link, Navigate } from "@tanstack/react-router";
import { LayoutGrid, Pencil, Plus, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { iconButton, secondaryButton } from "#/components/buttonStyles";
import { widePage } from "#/components/pageWidths";
import { useMe } from "#/features/auth/session";
import { usePageTitle } from "#/lib/usePageTitle";
import { type Dashboard, dashboardsQueryOptions } from "./api";
import { DashboardDialog, type DashboardDialogTarget } from "./DashboardDialog";
import { DeleteDashboardDialog } from "./DeleteDashboardDialog";

const badge =
	"rounded-full border border-line px-2 py-0.5 text-xs font-medium text-muted";

export function DashboardsPage({
	openCreate,
	onCreateOpened,
}: {
	/** Open the create dialog on mount (from "Create your first dashboard"). */
	openCreate: boolean;
	onCreateOpened: () => void;
}) {
	const { t } = useTranslation();
	usePageTitle(t("dashboards.title"));
	const { data: me } = useMe();
	const list = useQuery({ ...dashboardsQueryOptions, enabled: !!me });
	const [editing, setEditing] = useState<DashboardDialogTarget | null>(null);
	const [deleting, setDeleting] = useState<Dashboard | null>(null);

	useEffect(() => {
		if (openCreate) {
			setEditing({ dashboard: null });
			onCreateOpened();
		}
	}, [openCreate, onCreateOpened]);

	// The session can end while the page is open (401 from any request).
	if (me === null) return <Navigate to="/login" />;
	if (!me) return null;

	const createButton = (
		<button
			type="button"
			className={secondaryButton}
			onClick={() => setEditing({ dashboard: null })}
		>
			<Plus className="size-4" aria-hidden />
			{t("dashboards.create")}
		</button>
	);

	return (
		<section className={`${widePage} pt-12`}>
			<div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
				<div>
					<h1 className="text-3xl font-semibold tracking-tight">
						{t("dashboards.title")}
					</h1>
					<p className="mt-2 max-w-[60ch] text-muted">
						{t("dashboards.intro")}
					</p>
				</div>
				{list.data && list.data.length > 0 && (
					<div className="shrink-0">{createButton}</div>
				)}
			</div>
			<div className="mt-8">
				{list.isPending ? (
					<p className="text-muted">{t("common.loading")}</p>
				) : list.isError ? (
					<p role="alert" className="text-sm text-red-700">
						{t("common.genericError")}
					</p>
				) : list.data.length === 0 ? (
					<div className="rounded-xl border border-dashed border-line bg-surface px-6 py-12 text-center">
						<LayoutGrid
							className="mx-auto size-8 text-muted"
							strokeWidth={1.75}
							aria-hidden
						/>
						<h2 className="mt-4 text-lg font-semibold">
							{t("dashboards.empty.title")}
						</h2>
						<p className="mx-auto mt-1 max-w-[45ch] text-muted">
							{t("dashboards.empty.text")}
						</p>
						<div className="mt-6">{createButton}</div>
					</div>
				) : (
					<ul className="divide-y divide-line rounded-xl border border-line bg-surface">
						{list.data.map((d) => (
							<li key={d.id} className="flex items-center gap-3 px-4 py-3">
								<div className="min-w-0 flex-1">
									<div className="flex flex-wrap items-center gap-2">
										<span className="truncate font-medium">{d.title}</span>
										{d.isDefault && (
											<span className={badge}>{t("dashboards.default")}</span>
										)}
										{d.isPublic && (
											<span className={badge}>{t("dashboards.public")}</span>
										)}
									</div>
									<Link
										to="/$userSlug/$dashboardSlug"
										params={{ userSlug: me.slug, dashboardSlug: d.slug }}
										className="block truncate text-sm text-muted underline-offset-4 hover:underline"
									>
										/{me.slug}/{d.slug}
									</Link>
								</div>
								<button
									type="button"
									className={iconButton}
									aria-label={t("dashboards.edit", { title: d.title })}
									onClick={() => setEditing({ dashboard: d })}
								>
									<Pencil className="size-4" aria-hidden />
								</button>
								<button
									type="button"
									className={iconButton}
									aria-label={t("dashboards.delete", { title: d.title })}
									onClick={() => setDeleting(d)}
								>
									<Trash2 className="size-4" aria-hidden />
								</button>
							</li>
						))}
					</ul>
				)}
			</div>
			<DashboardDialog
				target={editing}
				userSlug={me.slug}
				onClose={() => setEditing(null)}
			/>
			<DeleteDashboardDialog
				dashboard={deleting}
				onClose={() => setDeleting(null)}
			/>
		</section>
	);
}
