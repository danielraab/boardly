import { Dialog } from "@base-ui/react/dialog";
import { Menu } from "@base-ui/react/menu";
import {
	ChevronRight,
	Ellipsis,
	FolderPlus,
	Inbox,
	Notebook,
	Pencil,
	Plus,
	Tag,
	Trash2,
	X,
} from "lucide-react";
import { type ReactNode, useState } from "react";
import { useTranslation } from "react-i18next";
import { iconButton, secondaryButton } from "#/components/buttonStyles";
import { ChosenIcon } from "#/components/icons/ChosenIcon";
import { menuItem, menuPopup } from "#/components/menuStyles";
import { useStoredIdSet } from "#/lib/useStoredIdSet";
import type { Note, NoteCategory } from "../api";
import { iconProps } from "../noteIcon";
import { CategoryDialog, type CategoryDialogTarget } from "./CategoryDialog";
import { DeleteCategoryDialog } from "./DeleteCategoryDialog";
import {
	buildTree,
	type CategoryFilter,
	type CategoryTreeNode,
	filterNotes,
} from "./tree";

const sameFilter = (a: CategoryFilter, b: CategoryFilter) =>
	typeof a === "object" && typeof b === "object" ? a.id === b.id : a === b;

const entry =
	"flex min-w-0 flex-1 items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm hover:bg-surface aria-[current=true]:bg-surface aria-[current=true]:font-medium";

/**
 * The categories of /notes: "All notes", "Uncategorized" and the category
 * tree with note counts, to filter the grid and to manage categories. At
 * phone width it is a drawer that opens from a button.
 */
export function SidePanel(props: {
	notes: Note[];
	categories: NoteCategory[];
	filter: CategoryFilter;
	onFilterChange: (filter: CategoryFilter) => void;
}) {
	const { t } = useTranslation();
	const [drawerOpen, setDrawerOpen] = useState(false);
	return (
		<>
			<aside
				aria-label={t("notes.categories.heading")}
				className="hidden w-64 shrink-0 md:block"
			>
				<Panel {...props} />
			</aside>
			<Dialog.Root open={drawerOpen} onOpenChange={setDrawerOpen}>
				<Dialog.Trigger className={`${secondaryButton} md:hidden`}>
					<Tag className="size-4" aria-hidden />
					{t("notes.categories.open")}
				</Dialog.Trigger>
				<Dialog.Portal>
					<Dialog.Backdrop className="fixed inset-0 z-40 bg-ink/30 md:hidden" />
					<Dialog.Popup className="fixed inset-y-0 left-0 z-40 w-[min(20rem,85vw)] overflow-auto border-r border-line bg-paper p-4 text-ink shadow-[0_8px_24px_-12px_var(--color-shadow)] md:hidden">
						<div className="mb-2 flex items-center justify-between">
							<Dialog.Title className="text-base font-semibold">
								{t("notes.categories.heading")}
							</Dialog.Title>
							<Dialog.Close
								className={iconButton}
								aria-label={t("notes.categories.close")}
							>
								<X className="size-4" aria-hidden />
							</Dialog.Close>
						</div>
						<Panel
							{...props}
							onFilterChange={(f) => {
								props.onFilterChange(f);
								setDrawerOpen(false);
							}}
						/>
					</Dialog.Popup>
				</Dialog.Portal>
			</Dialog.Root>
		</>
	);
}

function Panel({
	notes,
	categories,
	filter,
	onFilterChange,
}: {
	notes: Note[];
	categories: NoteCategory[];
	filter: CategoryFilter;
	onFilterChange: (filter: CategoryFilter) => void;
}) {
	const { t, i18n } = useTranslation();
	const [collapsed, setCollapsed] = useStoredIdSet("boardly.notes.collapsed");
	const [dialog, setDialog] = useState<CategoryDialogTarget | null>(null);
	const [deleting, setDeleting] = useState<NoteCategory | null>(null);
	const tree = buildTree(categories, i18n.language);
	const count = (f: CategoryFilter) => filterNotes(notes, categories, f).length;

	const item = (
		f: CategoryFilter,
		icon: ReactNode,
		label: string,
		depth = 0,
	) => {
		const n = count(f);
		return (
			<button
				type="button"
				className={entry}
				aria-current={sameFilter(filter, f)}
				aria-label={`${label}, ${t("notes.categories.count", { count: n })}`}
				onClick={() => onFilterChange(f)}
				style={{ paddingLeft: `${0.5 + depth}rem` }}
			>
				{icon}
				<span className="min-w-0 flex-1 truncate">{label}</span>
				<span className="text-xs text-muted tabular-nums" aria-hidden>
					{n}
				</span>
			</button>
		);
	};

	const node = ({ category, children, depth }: CategoryTreeNode): ReactNode => {
		const open = !collapsed.has(category.id);
		const toggle = () => {
			const next = new Set(collapsed);
			if (open) next.add(category.id);
			else next.delete(category.id);
			setCollapsed(next);
		};
		return (
			<li key={category.id}>
				<div className="group flex items-center">
					<span
						className="flex w-6 shrink-0 justify-center"
						style={{ marginLeft: `${depth}rem` }}
					>
						{children.length > 0 && (
							<button
								type="button"
								className={iconButton}
								aria-expanded={open}
								aria-label={t(
									open
										? "notes.categories.collapse"
										: "notes.categories.expand",
									{ title: category.title },
								)}
								onClick={toggle}
							>
								<ChevronRight
									className={`size-4 transition-transform ${open ? "rotate-90" : ""}`}
									aria-hidden
								/>
							</button>
						)}
					</span>
					{item(
						{ id: category.id },
						<ChosenIcon
							{...iconProps(category.icon)}
							className="size-4"
							fallback={Tag}
						/>,
						category.title,
					)}
					<Menu.Root>
						<Menu.Trigger
							className={`${iconButton} opacity-100 md:opacity-0 md:group-hover:opacity-100 md:focus-visible:opacity-100 data-[popup-open]:opacity-100`}
							aria-label={t("notes.categories.actions", {
								title: category.title,
							})}
						>
							<Ellipsis className="size-4" aria-hidden />
						</Menu.Trigger>
						<Menu.Portal>
							<Menu.Positioner align="end" sideOffset={4} className="z-50">
								<Menu.Popup className={menuPopup}>
									<Menu.Item
										className={menuItem}
										onClick={() => setDialog({ category })}
									>
										<Pencil className="size-4" aria-hidden />
										{t("notes.categories.edit")}
									</Menu.Item>
									<Menu.Item
										className={menuItem}
										onClick={() => setDialog({ parentId: category.id })}
									>
										<FolderPlus className="size-4" aria-hidden />
										{t("notes.categories.newSub")}
									</Menu.Item>
									<Menu.Item
										className={menuItem}
										onClick={() => setDeleting(category)}
									>
										<Trash2 className="size-4" aria-hidden />
										{t("notes.categories.delete")}
									</Menu.Item>
								</Menu.Popup>
							</Menu.Positioner>
						</Menu.Portal>
					</Menu.Root>
				</div>
				{open && children.length > 0 && <ul>{children.map(node)}</ul>}
			</li>
		);
	};

	return (
		<nav aria-label={t("notes.categories.heading")} className="space-y-1">
			<h2 className="hidden px-2 pb-1 text-xs font-medium tracking-wide text-muted uppercase md:block">
				{t("notes.categories.heading")}
			</h2>
			<div className="flex">
				<span className="w-6 shrink-0" />
				{item(
					"all",
					<Notebook className="size-4 text-muted" aria-hidden />,
					t("notes.categories.all"),
				)}
				<span className="w-7 shrink-0" />
			</div>
			<div className="flex">
				<span className="w-6 shrink-0" />
				{item(
					"none",
					<Inbox className="size-4 text-muted" aria-hidden />,
					t("notes.categories.none"),
				)}
				<span className="w-7 shrink-0" />
			</div>
			{tree.length > 0 && <ul className="space-y-0.5">{tree.map(node)}</ul>}
			<button
				type="button"
				className="mt-2 flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm text-muted hover:bg-surface hover:text-ink"
				onClick={() => setDialog({ parentId: null })}
			>
				<Plus className="size-4" aria-hidden />
				{t("notes.categories.new")}
			</button>
			<CategoryDialog
				target={dialog}
				categories={categories}
				onClose={() => setDialog(null)}
			/>
			<DeleteCategoryDialog
				category={deleting}
				categories={categories}
				notes={notes}
				onClose={() => setDeleting(null)}
				onDeleted={(c) => {
					if (typeof filter === "object" && filter.id === c.id)
						onFilterChange("all");
				}}
			/>
		</nav>
	);
}
