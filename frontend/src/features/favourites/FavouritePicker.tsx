import { useQuery } from "@tanstack/react-query";
import { ChevronRight, Folder, Search } from "lucide-react";
import { type ReactNode, useState } from "react";
import { useTranslation } from "react-i18next";
import { ImageIcon } from "#/components/ImageIcon";
import { type FavouriteNode, favouritesQueryOptions } from "./api";
import { label, pathsOf } from "./tree";

/** What can be chosen: a folder (including the root) or a link. */
export type PickerMode = "folder" | "link";

const INDENT_REM = 1.25;

/** The IDs of the folders above id, or an empty set when it isn't in the tree. */
function ancestorsOf(root: FavouriteNode, id: string): Set<string> {
	const walk = (node: FavouriteNode, above: string[]): string[] | null => {
		if (node.id === id) return above;
		for (const child of node.children ?? []) {
			const found = walk(child, [...above, node.id]);
			if (found) return found;
		}
		return null;
	};
	return new Set(walk(root, []) ?? []);
}

function hasLink(node: FavouriteNode): boolean {
	return (node.children ?? []).some((c) => c.kind === "link" || hasLink(c));
}

/** The children a folder shows in the tree for the mode. */
function visibleChildren(node: FavouriteNode, mode: PickerMode) {
	return (node.children ?? []).filter((c) =>
		c.kind === "link" ? mode === "link" : mode === "folder" || hasLink(c),
	);
}

function matches(node: FavouriteNode, query: string) {
	return [node.title ?? "", node.url ?? ""].some((s) =>
		s.toLowerCase().includes(query),
	);
}

/**
 * Chooses a folder or a link from the signed-in user's favourites: a tree
 * that can be expanded and collapsed, and searched by name and URL. A value
 * that isn't in the tree selects nothing.
 */
export function FavouritePicker({
	id,
	label: fieldLabel,
	mode,
	value,
	onChange,
}: {
	id: string;
	label: string;
	mode: PickerMode;
	value: string;
	onChange: (id: string) => void;
}) {
	const { t } = useTranslation();
	const favourites = useQuery(favouritesQueryOptions);
	const [query, setQuery] = useState("");
	// Until the viewer toggles a folder, the folders above the value are open.
	const [expanded, setExpanded] = useState<Set<string> | null>(null);
	const root = favourites.data;

	const labelId = `${id}-label`;
	const heading = (
		<span id={labelId} className="block text-sm font-medium">
			{fieldLabel}
		</span>
	);
	if (!root) {
		return (
			<div>
				{heading}
				<p className="mt-2 text-sm text-muted">
					{t("favourites.picker.loading")}
				</p>
			</div>
		);
	}

	const paths = pathsOf(root, t("favourites.title"));
	const open = expanded ?? ancestorsOf(root, value);
	const toggle = (folderId: string) => {
		const next = new Set(open);
		if (!next.delete(folderId)) next.add(folderId);
		setExpanded(next);
	};

	// The row is highlighted when its radio is chosen or focused. The path is
	// the radio's description, outside the label so it isn't part of the name.
	const option = (
		node: FavouriteNode,
		depth: number,
		detail?: string,
	): ReactNode => (
		<div
			key={node.id}
			className="flex min-w-0 flex-1 items-center rounded-md hover:bg-paper has-[:checked]:bg-accent/15 has-[:checked]:font-medium has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-accent"
			style={depth ? { marginLeft: `${depth * INDENT_REM}rem` } : undefined}
		>
			<label className="flex min-w-0 flex-1 cursor-pointer items-center gap-2 px-2 py-1">
				<input
					type="radio"
					name={id}
					value={node.id}
					checked={node.id === value}
					onChange={() => onChange(node.id)}
					aria-describedby={detail ? `${id}-${node.id}-path` : undefined}
					className="sr-only"
				/>
				{node.kind === "folder" ? (
					<Folder className="size-4 shrink-0 text-muted" aria-hidden />
				) : (
					<ImageIcon imageId={node.iconImageId} className="size-4" />
				)}
				<span className="truncate">
					{node.id === root.id ? t("favourites.title") : label(node)}
				</span>
			</label>
			{detail && (
				<span
					id={`${id}-${node.id}-path`}
					className="min-w-0 shrink truncate pr-2 text-xs font-normal text-muted"
				>
					{detail}
				</span>
			)}
		</div>
	);

	const toggleFor = (node: FavouriteNode) => {
		if (visibleChildren(node, mode).length === 0) {
			return <span className="size-6 shrink-0" />;
		}
		const isOpen = open.has(node.id);
		return (
			<button
				type="button"
				aria-expanded={isOpen}
				aria-label={t(isOpen ? "favourites.collapse" : "favourites.expand", {
					name: label(node),
				})}
				onClick={() => toggle(node.id)}
				className="inline-flex size-6 shrink-0 items-center justify-center rounded-md text-muted hover:bg-paper hover:text-ink"
			>
				<ChevronRight
					className={`size-4 transition-transform motion-reduce:transition-none ${isOpen ? "rotate-90" : ""}`}
					aria-hidden
				/>
			</button>
		);
	};

	// In link mode a folder is only a heading that opens and closes.
	const folderRow = (node: FavouriteNode) =>
		mode === "folder" ? (
			option(node, 0)
		) : (
			<span className="flex min-w-0 flex-1 items-center gap-2 px-2 py-1">
				<Folder className="size-4 shrink-0 text-muted" aria-hidden />
				<span className="truncate">{label(node)}</span>
			</span>
		);

	const tree = (folder: FavouriteNode, depth: number): ReactNode[] =>
		visibleChildren(folder, mode).flatMap((node) => {
			const indent = depth
				? { marginLeft: `${depth * INDENT_REM}rem` }
				: undefined;
			if (node.kind === "link") {
				return [
					<div key={node.id} className="flex">
						<span className="size-6 shrink-0" />
						{option(node, depth)}
					</div>,
				];
			}
			return [
				<div key={node.id} className="flex items-center" style={indent}>
					{toggleFor(node)}
					{folderRow(node)}
				</div>,
				...(open.has(node.id) ? tree(node, depth + 1) : []),
			];
		});

	let content: ReactNode[];
	const q = query.trim().toLowerCase();
	if (q) {
		const found: ReactNode[] = [];
		const visit = (node: FavouriteNode) => {
			for (const child of node.children ?? []) {
				if (child.kind === mode && matches(child, q)) {
					found.push(option(child, 0, paths.get(node.id)));
				}
				if (child.kind === "folder") visit(child);
			}
		};
		visit(root);
		content = found;
	} else if (mode === "folder") {
		content = [option(root, 0), ...tree(root, 1)];
	} else {
		content = tree(root, 0);
	}

	const selected = paths.get(value);
	const empty = mode === "link" && !q && !hasLink(root);

	return (
		<div>
			{heading}
			<div className="mt-2 rounded-lg border border-line bg-paper/40">
				<div className="flex items-center gap-2 border-b border-line px-3">
					<Search className="size-4 shrink-0 text-muted" aria-hidden />
					<input
						type="search"
						value={query}
						onChange={(e) => setQuery(e.target.value)}
						placeholder={t("favourites.picker.search")}
						aria-label={t("favourites.picker.search")}
						aria-controls={`${id}-options`}
						className="min-w-0 flex-1 bg-transparent py-2 text-sm outline-none"
					/>
				</div>
				<div
					id={`${id}-options`}
					role="radiogroup"
					aria-labelledby={labelId}
					className="max-h-60 overflow-auto p-1 text-sm"
				>
					{content.length > 0 ? (
						content
					) : (
						<p className="px-2 py-1 text-muted">
							{t(
								empty
									? "favourites.picker.noLinks"
									: "favourites.picker.noMatches",
							)}
						</p>
					)}
				</div>
			</div>
			{selected && (
				<p className="mt-1 truncate text-sm text-muted">
					{t("favourites.picker.selected", { path: selected })}
				</p>
			)}
		</div>
	);
}
