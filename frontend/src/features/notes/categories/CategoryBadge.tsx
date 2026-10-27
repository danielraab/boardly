import { Menu } from "@base-ui/react/menu";
import { Check, FolderPlus } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ChosenIcon } from "#/components/icons/ChosenIcon";
import { menuItem, menuPopup } from "#/components/menuStyles";
import type { NoteCategory } from "../api";
import { noteBackground } from "../colours";
import { iconProps } from "../noteIcon";
import { buildTree, categoryPath, flatten } from "./tree";

/** The value of the picker for "No category". */
const noCategory = "";

/**
 * A note's category as a badge that opens a picker for another category.
 * Without a category it shows nothing, or with showAdd an "Add category"
 * button that opens the same picker.
 */
export function CategoryBadge({
	categoryId,
	categories,
	onChange,
	showAdd = false,
	disabled = false,
}: {
	categoryId: string | null;
	categories: NoteCategory[];
	onChange: (categoryId: string | null) => void;
	showAdd?: boolean;
	disabled?: boolean;
}) {
	const { t, i18n } = useTranslation();
	const category = categories.find((c) => c.id === categoryId) ?? null;
	if (!category && !showAdd) return null;
	const path = category
		? categoryPath(categories, category.id).join(" › ")
		: "";
	const options = flatten(buildTree(categories, i18n.language));

	return (
		<Menu.Root>
			{category ? (
				<Menu.Trigger
					disabled={disabled}
					title={path}
					aria-label={t("notes.categories.badge.change", { path })}
					className={`relative inline-flex max-w-full items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium text-ink hover:ring-1 hover:ring-ink/40 disabled:opacity-60 data-[popup-open]:ring-2 data-[popup-open]:ring-ink ${category.color ? "" : "border border-line bg-surface"}`}
					style={{ background: noteBackground(category.color) }}
				>
					{category.icon && (
						<ChosenIcon {...iconProps(category.icon)} className="size-3.5" />
					)}
					<span className="truncate">{category.title}</span>
				</Menu.Trigger>
			) : (
				<Menu.Trigger
					disabled={disabled}
					className="inline-flex items-center gap-1.5 rounded-full border border-dashed border-line px-2.5 py-0.5 text-xs font-medium text-muted hover:text-ink disabled:opacity-60"
				>
					<FolderPlus className="size-3.5" aria-hidden />
					{t("notes.categories.badge.add")}
				</Menu.Trigger>
			)}
			<Menu.Portal>
				<Menu.Positioner align="start" sideOffset={6}>
					<Menu.Popup
						className={`${menuPopup} max-h-80 max-w-[calc(100vw-2rem)] overflow-auto`}
						aria-label={t("notes.categories.badge.picker")}
					>
						<Menu.RadioGroup
							value={categoryId ?? noCategory}
							onValueChange={(value: string) =>
								onChange(value === noCategory ? null : value)
							}
						>
							{[
								{
									id: noCategory,
									title: t("notes.categories.badge.none"),
									depth: 0,
									icon: null,
								},
								...options.map((o) => ({
									id: o.category.id,
									title: o.category.title,
									depth: o.depth,
									icon: o.category.icon,
								})),
							].map((option) => (
								<Menu.RadioItem
									key={option.id}
									value={option.id}
									closeOnClick
									className={menuItem}
									style={{ paddingLeft: `${0.75 + option.depth}rem` }}
								>
									<span className="flex size-4 shrink-0 items-center">
										<Menu.RadioItemIndicator>
											<Check className="size-4" aria-hidden />
										</Menu.RadioItemIndicator>
									</span>
									{option.icon && (
										<ChosenIcon
											{...iconProps(option.icon)}
											className="size-4"
										/>
									)}
									<span className="truncate">{option.title}</span>
								</Menu.RadioItem>
							))}
						</Menu.RadioGroup>
					</Menu.Popup>
				</Menu.Positioner>
			</Menu.Portal>
		</Menu.Root>
	);
}
