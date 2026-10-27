import { Menu } from "@base-ui/react/menu";
import { Link } from "@tanstack/react-router";
import {
	CircleUserRound,
	LayoutGrid,
	LogIn,
	LogOut,
	NotebookPen,
	Settings,
	Star,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { useMe, useSignOut } from "#/features/auth/session";
import { headerButton } from "./buttonStyles";
import { menuItem, menuPopup } from "./menuStyles";

export function UserMenu() {
	const { t } = useTranslation();
	const { data: me, isPending } = useMe();
	const signOut = useSignOut();

	return (
		<Menu.Root>
			<Menu.Trigger className={headerButton} aria-label={t("shell.userMenu")}>
				<CircleUserRound className="size-6" strokeWidth={1.75} aria-hidden />
			</Menu.Trigger>
			<Menu.Portal>
				<Menu.Positioner align="end" sideOffset={6}>
					<Menu.Popup className={menuPopup}>
						{isPending ? (
							<output className="flex items-center gap-2 px-3 py-2 text-muted">
								<span className="size-4 animate-spin rounded-full border-2 border-line border-t-muted motion-reduce:animate-none" />
								{t("shell.loading")}
							</output>
						) : me ? (
							<>
								<Menu.Group>
									<Menu.GroupLabel className="max-w-64 px-3 pt-2 pb-2.5">
										{me.name && (
											<span className="block truncate font-medium text-ink">
												{me.name}
											</span>
										)}
										<span
											className={`block truncate ${me.name ? "text-muted" : "font-medium text-ink"}`}
										>
											{me.email}
										</span>
									</Menu.GroupLabel>
								</Menu.Group>
								<div className="mx-1 mb-1 border-t border-line" />
								<Menu.LinkItem
									closeOnClick
									className={menuItem}
									render={<Link to="/dashboards" />}
								>
									<LayoutGrid className="size-4 text-muted" aria-hidden />
									{t("shell.dashboards")}
								</Menu.LinkItem>
								<Menu.LinkItem
									closeOnClick
									className={menuItem}
									render={<Link to="/favourites" />}
								>
									<Star className="size-4 text-muted" aria-hidden />
									{t("shell.favourites")}
								</Menu.LinkItem>
								<Menu.LinkItem
									closeOnClick
									className={menuItem}
									render={<Link to="/notes" />}
								>
									<NotebookPen className="size-4 text-muted" aria-hidden />
									{t("shell.notes")}
								</Menu.LinkItem>
								<Menu.LinkItem
									closeOnClick
									className={menuItem}
									render={<Link to="/settings" />}
								>
									<Settings className="size-4 text-muted" aria-hidden />
									{t("shell.settings")}
								</Menu.LinkItem>
								<Menu.Item className={menuItem} onClick={() => void signOut()}>
									<LogOut className="size-4 text-muted" aria-hidden />
									{t("shell.signOut")}
								</Menu.Item>
							</>
						) : (
							<Menu.LinkItem
								closeOnClick
								className={menuItem}
								render={<Link to="/login" />}
							>
								<LogIn className="size-4 text-muted" aria-hidden />
								{t("shell.signIn")}
							</Menu.LinkItem>
						)}
					</Menu.Popup>
				</Menu.Positioner>
			</Menu.Portal>
		</Menu.Root>
	);
}
