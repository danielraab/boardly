import { Menu } from "@base-ui/react/menu";
import { useNavigate, useSearch } from "@tanstack/react-router";
import { Ellipsis, LayoutGrid, Settings } from "lucide-react";
import { type ReactNode, useState } from "react";
import { useTranslation } from "react-i18next";
import { secondaryIconButton } from "#/components/buttonStyles";
import { menuItem, menuPopup } from "#/components/menuStyles";
import type { Dashboard, DashboardViewData } from "./api";
import { useShowDashboard } from "./currentDashboard";
import { DashboardDialog, type DashboardDialogTarget } from "./DashboardDialog";
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

/**
 * A dashboard page: its title and sections, and edit mode and settings for
 * its owner. `atHome` is set when `/` shows it as the default dashboard.
 */
export function DashboardView({
	view,
	atHome = false,
}: {
	view: DashboardViewData;
	atHome?: boolean;
}) {
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

	// After a save, follow the dashboard when its current address stops showing it.
	const onSaved = (saved: Dashboard) => {
		const moved = atHome
			? !saved.isDefault
			: saved.slug !== view.dashboard.slug;
		if (moved) {
			void navigate({
				to: "/$userSlug/$dashboardSlug",
				params: { userSlug: view.ownerSlug, dashboardSlug: saved.slug },
				replace: true,
			});
		}
	};

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
					<DashboardActions
						view={view}
						onEditLayout={() => setEditing(true)}
						onSaved={onSaved}
					/>
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

/** The owner's menu next to the title: edit the layout or the settings. */
function DashboardActions({
	view,
	onEditLayout,
	onSaved,
}: {
	view: DashboardViewData;
	onEditLayout: () => void;
	onSaved: (saved: Dashboard) => void;
}) {
	const { t } = useTranslation();
	const [settings, setSettings] = useState<DashboardDialogTarget | null>(null);

	function openSettings() {
		if (view.id === undefined) return;
		setSettings({
			dashboard: {
				id: view.id,
				slug: view.dashboard.slug,
				title: view.dashboard.title,
				isDefault: view.isDefault ?? false,
				isPublic: view.isPublic ?? false,
				...view.columns,
			},
		});
	}

	return (
		<>
			<Menu.Root>
				<Menu.Trigger
					className={secondaryIconButton}
					aria-label={t("dashboards.view.actions")}
					title={t("dashboards.view.actions")}
				>
					<Ellipsis className="size-4" aria-hidden />
				</Menu.Trigger>
				<Menu.Portal>
					<Menu.Positioner align="end" sideOffset={6}>
						<Menu.Popup className={menuPopup}>
							<Menu.Item className={menuItem} onClick={onEditLayout}>
								<LayoutGrid className="size-4 text-muted" aria-hidden />
								{t("dashboards.editor.editLayout")}
							</Menu.Item>
							<Menu.Item className={menuItem} onClick={openSettings}>
								<Settings className="size-4 text-muted" aria-hidden />
								{t("dashboards.view.settings")}
							</Menu.Item>
						</Menu.Popup>
					</Menu.Positioner>
				</Menu.Portal>
			</Menu.Root>
			<DashboardDialog
				target={settings}
				userSlug={view.ownerSlug}
				onClose={() => setSettings(null)}
				onSaved={onSaved}
			/>
		</>
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
