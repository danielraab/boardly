import { Menu } from "@base-ui/react/menu";
import { Check, Languages } from "lucide-react";
import { useTranslation } from "react-i18next";
import { type LanguageCode, languages, setLanguage } from "#/i18n";
import { headerButton } from "./buttonStyles";
import { menuItem, menuPopup } from "./menuStyles";

export function LanguageSwitcher() {
	const { t, i18n } = useTranslation();

	return (
		<Menu.Root>
			<Menu.Trigger className={headerButton} aria-label={t("shell.language")}>
				<Languages className="size-5" aria-hidden />
			</Menu.Trigger>
			<Menu.Portal>
				<Menu.Positioner align="end" sideOffset={6}>
					<Menu.Popup className={menuPopup}>
						<Menu.RadioGroup
							value={i18n.resolvedLanguage}
							onValueChange={(code: LanguageCode) => setLanguage(code)}
						>
							{languages.map((lang) => (
								<Menu.RadioItem
									key={lang.code}
									value={lang.code}
									closeOnClick
									className={menuItem}
									lang={lang.code}
								>
									<span className="flex size-4 items-center">
										<Menu.RadioItemIndicator>
											<Check className="size-4" aria-hidden />
										</Menu.RadioItemIndicator>
									</span>
									{lang.label}
								</Menu.RadioItem>
							))}
						</Menu.RadioGroup>
					</Menu.Popup>
				</Menu.Positioner>
			</Menu.Portal>
		</Menu.Root>
	);
}
