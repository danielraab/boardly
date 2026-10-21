import { useNavigate, useSearch } from "@tanstack/react-router";
import { LayoutGrid, Pencil } from "lucide-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { secondaryButton } from "#/features/auth/LoginPage";
import type { DashboardViewData } from "./api";
import { useShowDashboard } from "./currentDashboard";
import { LayoutEditor } from "./layout/LayoutEditor";
import { DashboardGrid } from "./layout/SectionGrid";
import { useColumns } from "./layout/useColumns";

/** Search params of pages that show a dashboard; `edit` opens edit mode. */
export type DashboardSearch = { edit?: true };

export function validateDashboardSearch(
	search: Record<string, unknown>,
): DashboardSearch {
	return { edit: search.edit === true ? true : undefined };
}

/** A dashboard page: its title and sections, and edit mode for its owner. */
export function DashboardView({ view }: { view: DashboardViewData }) {
	const { t } = useTranslation();
	useShowDashboard(view);
	const columns = useColumns(view.columns);
	const navigate = useNavigate();
	const search: DashboardSearch = useSearch({ strict: false });

	// Only the owner can edit; for anyone else the parameter is ignored.
	const editing = search.edit === true && view.isOwner && view.id !== undefined;
	const setEditing = (edit: boolean) =>
		void navigate({
			to: ".",
			search: (prev: DashboardSearch) => ({ ...prev, edit: edit || undefined }),
			replace: !edit,
		});

	if (editing) {
		return (
			<LayoutEditor
				view={view}
				columns={columns}
				onExit={() => setEditing(false)}
			/>
		);
	}

	return (
		<DashboardPage
			title={view.dashboard.title}
			actions={
				view.isOwner && (
					<button
						type="button"
						onClick={() => setEditing(true)}
						className={`${secondaryButton} px-4 py-2 text-sm`}
					>
						<Pencil className="size-4" aria-hidden />
						{t("dashboards.editor.edit")}
					</button>
				)
			}
		>
			{view.sections.length === 0 ? (
				<EmptyDashboard isOwner={view.isOwner} />
			) : (
				<DashboardGrid sections={view.sections} columns={columns} />
			)}
		</DashboardPage>
	);
}

/** The page frame with the title and, next to it, the edit actions. */
export function DashboardPage({
	title,
	actions,
	children,
}: {
	title: string;
	actions?: ReactNode;
	children: ReactNode;
}) {
	return (
		<section className="pt-12">
			<div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-3">
				<h1 className="min-w-0 text-3xl font-semibold tracking-tight break-words sm:text-4xl">
					{title}
				</h1>
				{actions && <div className="flex items-center gap-2">{actions}</div>}
			</div>
			<div className="mt-10">{children}</div>
		</section>
	);
}

function EmptyDashboard({ isOwner }: { isOwner: boolean }) {
	const { t } = useTranslation();
	return (
		<div className="rounded-xl border border-dashed border-line bg-surface px-6 py-12 text-center">
			<LayoutGrid
				className="mx-auto size-8 text-muted"
				strokeWidth={1.75}
				aria-hidden
			/>
			<h2 className="mt-4 text-lg font-semibold">
				{t("dashboards.view.emptyTitle")}
			</h2>
			<p className="mx-auto mt-1 max-w-[45ch] text-muted">
				{isOwner
					? t("dashboards.view.emptyOwnerText")
					: t("dashboards.view.emptyText")}
			</p>
		</div>
	);
}
