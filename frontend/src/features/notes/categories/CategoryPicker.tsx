import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { ChosenIcon } from "#/components/icons/ChosenIcon";
import { noteCategoriesQueryOptions } from "../api";
import { iconProps } from "../noteIcon";
import { buildTree, flatten } from "./tree";

const INDENT_REM = 1;

/**
 * A choice of one of the user's categories, shown as a tree, or of the empty
 * choice (null) labelled emptyLabel, such as "No category" or "Top level".
 * Without emptyLabel, null means nothing is chosen. extraChoices are offered
 * between the empty choice and the tree; their IDs must not be category IDs.
 */
export function CategoryPicker({
	id,
	label,
	emptyLabel,
	extraChoices = [],
	value,
	onChange,
}: {
	id: string;
	label: string;
	emptyLabel?: string;
	extraChoices?: { id: string; title: string }[];
	value: string | null;
	onChange: (id: string | null) => void;
}) {
	const { t, i18n } = useTranslation();
	const categories = useQuery(noteCategoriesQueryOptions);
	const labelId = `${id}-label`;

	const options = [
		...(emptyLabel === undefined
			? []
			: [{ id: null, title: emptyLabel, depth: 0, icon: null }]),
		...extraChoices.map((c) => ({ ...c, depth: 0, icon: null })),
		...flatten(buildTree(categories.data ?? [], i18n.language)).map((n) => ({
			id: n.category.id,
			title: n.category.title,
			depth: n.depth,
			icon: n.category.icon,
		})),
	];

	return (
		<div>
			<span id={labelId} className="block text-sm font-medium">
				{label}
			</span>
			{categories.isPending ? (
				<p className="mt-2 text-sm text-muted">
					{t("notes.categories.picker.loading")}
				</p>
			) : (
				<div
					role="radiogroup"
					aria-labelledby={labelId}
					className="mt-2 max-h-60 overflow-auto rounded-lg border border-line bg-paper/40 p-1 text-sm"
				>
					{options.map((option) => (
						<label
							key={option.id ?? ""}
							className="flex min-w-0 cursor-pointer items-center gap-2 rounded-md px-2 py-1 hover:bg-paper has-[:checked]:bg-accent/15 has-[:checked]:font-medium has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-accent"
							style={
								option.depth
									? { marginLeft: `${option.depth * INDENT_REM}rem` }
									: undefined
							}
						>
							<input
								type="radio"
								name={id}
								value={option.id ?? ""}
								checked={option.id === value}
								onChange={() => onChange(option.id)}
								className="sr-only"
							/>
							{option.icon && (
								<ChosenIcon {...iconProps(option.icon)} className="size-4" />
							)}
							<span className="truncate">{option.title}</span>
						</label>
					))}
				</div>
			)}
		</div>
	);
}
