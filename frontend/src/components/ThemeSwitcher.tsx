import { Menu } from "@base-ui/react/menu";
import { Check, Palette } from "lucide-react";
import { useTranslation } from "react-i18next";
import {
	setTheme,
	type ThemeChoice,
	type ThemeId,
	themes,
	useTheme,
} from "#/theme";
import { headerButton, menuItem, menuPopup } from "./menuStyles";

const swatch =
	"flex size-5 shrink-0 overflow-hidden rounded-full border border-line";

/** A theme's paper color with an accent dot, rendered with that theme's own tokens. */
function ThemeSwatch({ id }: { id: ThemeId }) {
	return (
		<span
			data-theme={id}
			className={`${swatch} items-center justify-center bg-paper`}
			aria-hidden
		>
			<span className="size-2 rounded-full bg-accent" />
		</span>
	);
}

/** Half White, half Black: System switches between the two. */
function SystemSwatch() {
	return (
		<span className={swatch} aria-hidden>
			<span data-theme="white" className="w-1/2 bg-paper" />
			<span data-theme="black" className="w-1/2 bg-paper" />
		</span>
	);
}

function Indicator() {
	return (
		<span className="flex size-4 items-center">
			<Menu.RadioItemIndicator>
				<Check className="size-4" aria-hidden />
			</Menu.RadioItemIndicator>
		</span>
	);
}

export function ThemeSwitcher() {
	const { t } = useTranslation();
	const choice = useTheme();

	return (
		<Menu.Root>
			<Menu.Trigger className={headerButton} aria-label={t("shell.theme")}>
				<Palette className="size-5" aria-hidden />
			</Menu.Trigger>
			<Menu.Portal>
				<Menu.Positioner align="end" sideOffset={6}>
					<Menu.Popup className={menuPopup}>
						<Menu.RadioGroup
							value={choice}
							onValueChange={(value: ThemeChoice) => setTheme(value)}
						>
							<Menu.RadioItem value="system" closeOnClick className={menuItem}>
								<Indicator />
								<SystemSwatch />
								{t("theme.system")}
							</Menu.RadioItem>
							<Menu.Separator className="my-1 h-px bg-line" />
							{themes.map(({ id }) => (
								<Menu.RadioItem
									key={id}
									value={id}
									closeOnClick
									className={menuItem}
								>
									<Indicator />
									<ThemeSwatch id={id} />
									{t(`theme.${id}`)}
								</Menu.RadioItem>
							))}
						</Menu.RadioGroup>
					</Menu.Popup>
				</Menu.Positioner>
			</Menu.Portal>
		</Menu.Root>
	);
}
