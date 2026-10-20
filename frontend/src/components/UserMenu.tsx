import { Menu } from "@base-ui/react/menu";
import { Link } from "@tanstack/react-router";
import { CircleUserRound, LogIn } from "lucide-react";
import { useTranslation } from "react-i18next";
import { headerButton, menuItem, menuPopup } from "./menuStyles";

export function UserMenu() {
	const { t } = useTranslation();

	return (
		<Menu.Root>
			<Menu.Trigger className={headerButton} aria-label={t("shell.userMenu")}>
				<CircleUserRound className="size-6" strokeWidth={1.75} aria-hidden />
			</Menu.Trigger>
			<Menu.Portal>
				<Menu.Positioner align="end" sideOffset={6}>
					<Menu.Popup className={menuPopup}>
						<Menu.LinkItem
							closeOnClick
							className={menuItem}
							render={<Link to="/login" />}
						>
							<LogIn className="size-4 text-muted" aria-hidden />
							{t("shell.signIn")}
						</Menu.LinkItem>
					</Menu.Popup>
				</Menu.Positioner>
			</Menu.Portal>
		</Menu.Root>
	);
}
