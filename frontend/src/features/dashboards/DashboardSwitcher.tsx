import { Menu } from "@base-ui/react/menu";
import { Link } from "@tanstack/react-router";
import { Check, ChevronDown } from "lucide-react";
import { useTranslation } from "react-i18next";
import { menuItem, menuPopup } from "#/components/menuStyles";
import { useCurrentDashboard } from "./currentDashboard";

/**
 * The shown dashboard's title next to the product name, as a menu of the
 * other dashboards the viewer can open. Hidden with fewer than two of them.
 */
export function DashboardSwitcher() {
	const { t } = useTranslation();
	const view = useCurrentDashboard();
	if (!view || view.dashboards.length < 2) return null;

	return (
		<>
			<span className="text-lg text-muted" aria-hidden>
				/
			</span>
			<Menu.Root>
				<Menu.Trigger
					className="inline-flex min-w-0 items-center gap-1 rounded-md px-1.5 py-1 text-lg font-medium hover:bg-surface data-[popup-open]:bg-surface"
					aria-label={`${t("dashboards.switcher.label")}: ${view.dashboard.title}`}
				>
					<span className="truncate">{view.dashboard.title}</span>
					<ChevronDown className="size-4 shrink-0 text-muted" aria-hidden />
				</Menu.Trigger>
				<Menu.Portal>
					<Menu.Positioner align="start" sideOffset={6} className="z-50">
						<Menu.Popup className={menuPopup}>
							{view.dashboards.map((d) => {
								const current = d.slug === view.dashboard.slug;
								return (
									<Menu.LinkItem
										key={d.slug}
										closeOnClick
										className={menuItem}
										aria-current={current ? "page" : undefined}
										render={
											<Link
												to="/$userSlug/$dashboardSlug"
												params={{
													userSlug: view.ownerSlug,
													dashboardSlug: d.slug,
												}}
											/>
										}
									>
										<Check
											className={`size-4 ${current ? "text-ink" : "invisible"}`}
											aria-hidden
										/>
										<span className="truncate">{d.title}</span>
									</Menu.LinkItem>
								);
							})}
						</Menu.Popup>
					</Menu.Positioner>
				</Menu.Portal>
			</Menu.Root>
		</>
	);
}
