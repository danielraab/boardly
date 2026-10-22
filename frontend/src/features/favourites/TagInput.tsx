import { Combobox } from "@base-ui/react/combobox";
import { useQuery } from "@tanstack/react-query";
import { X } from "lucide-react";
import { useId, useState } from "react";
import { useTranslation } from "react-i18next";
import { menuItem, menuPopup } from "#/components/menuStyles";
import { useDebounced } from "#/lib/useDebounced";
import { suggestTags } from "./api";
import { MAX_TAG } from "./validate";

const sameTag = (a: string, b: string) =>
	a.trim().toLowerCase() === b.trim().toLowerCase();

/**
 * Tags as chips with suggestions from the user's existing tags. Typed text
 * that matches no tag can be added with Enter or from the last list entry.
 */
export function TagInput({
	value,
	onChange,
}: {
	value: string[];
	onChange: (tags: string[]) => void;
}) {
	const { t } = useTranslation();
	const id = useId();
	const [input, setInput] = useState("");
	const [highlighted, setHighlighted] = useState<string | undefined>();
	const query = useDebounced(input.trim(), 200);

	const suggestions = useQuery({
		queryKey: ["tags", query],
		queryFn: () => suggestTags(query),
		staleTime: 30_000,
	});

	const typed = input.trim();
	// Filtering again here keeps older results consistent with the text while
	// the debounced request for it is pending.
	const matches = (suggestions.data ?? [])
		.map((tag) => tag.name)
		.filter(
			(name) =>
				name.toLowerCase().startsWith(typed.toLowerCase()) &&
				!value.some((v) => sameTag(v, name)),
		);
	const canCreate =
		typed !== "" &&
		![...value, ...matches].some((name) => sameTag(name, typed));
	const items = canCreate ? [...matches, typed] : matches;

	function add(name: string) {
		const trimmed = name.trim();
		if (trimmed === "" || value.some((v) => sameTag(v, trimmed))) return;
		onChange([...value, trimmed]);
		setInput("");
	}

	return (
		<Combobox.Root
			multiple
			items={items}
			filter={null}
			value={value}
			onValueChange={(next) => {
				onChange(next);
				setInput("");
			}}
			inputValue={input}
			onInputValueChange={setInput}
			onItemHighlighted={setHighlighted}
		>
			<label htmlFor={id} className="block text-sm font-medium">
				{t("favourites.dialog.tagsLabel")}
			</label>
			<Combobox.Chips className="mt-2 flex flex-wrap items-center gap-1.5 rounded-lg border border-line bg-paper/40 px-2 py-1.5 focus-within:border-ink">
				<Combobox.Value>
					{(tags: string[]) => (
						<>
							{tags.map((tag) => (
								<Combobox.Chip
									key={tag}
									className="inline-flex items-center gap-1 rounded-full bg-paper py-0.5 pr-1 pl-2.5 text-sm"
								>
									{tag}
									<Combobox.ChipRemove
										className="inline-flex size-5 items-center justify-center rounded-full text-muted hover:bg-line/50 hover:text-ink"
										aria-label={t("favourites.dialog.removeTag", { name: tag })}
									>
										<X className="size-3.5" aria-hidden />
									</Combobox.ChipRemove>
								</Combobox.Chip>
							))}
							<Combobox.Input
								id={id}
								maxLength={MAX_TAG}
								placeholder={t("favourites.dialog.tagsPlaceholder")}
								className="min-w-24 flex-1 bg-transparent px-1.5 py-1 text-ink outline-none"
								onKeyDown={(event) => {
									// Enter without a highlighted entry adds the typed text.
									if (event.key === "Enter" && highlighted === undefined) {
										event.preventDefault();
										if (canCreate) add(typed);
									}
								}}
							/>
						</>
					)}
				</Combobox.Value>
			</Combobox.Chips>
			<Combobox.Portal>
				<Combobox.Positioner sideOffset={4} className="z-50">
					<Combobox.Popup
						className={`${menuPopup} max-h-64 w-[var(--anchor-width)] overflow-y-auto empty:hidden`}
					>
						<Combobox.List>
							{(name: string) => (
								<Combobox.Item key={name} value={name} className={menuItem}>
									{canCreate && name === typed
										? t("favourites.dialog.createTag", { name })
										: name}
								</Combobox.Item>
							)}
						</Combobox.List>
					</Combobox.Popup>
				</Combobox.Positioner>
			</Combobox.Portal>
		</Combobox.Root>
	);
}
