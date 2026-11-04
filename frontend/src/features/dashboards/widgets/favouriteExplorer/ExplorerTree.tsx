import { Link } from "@tanstack/react-router";
import {
	ChevronRight,
	ChevronsDownUp,
	ChevronsUpDown,
	Folder,
	Plus,
	Search,
	X,
} from "lucide-react";
import { type ReactNode, useId, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { iconButton } from "#/components/buttonStyles";
import { ImageIcon } from "#/components/ImageIcon";
import type { FavouriteNode } from "#/features/favourites/api";
import { DetailsButton } from "#/features/favourites/FavouriteDetails";
import type { FavouritesSearch } from "#/features/favourites/FavouritesPage";
import { Highlighted } from "#/features/favourites/Highlighted";
import { filterTree, label, searchTerms } from "#/features/favourites/tree";
import { sizeStyles, type WidgetSize } from "../sizes";
import { useExpanded } from "./useExpanded";

/** The info button's icon, a step smaller than the row's icons. */
const detailsIcon: Record<WidgetSize, string> = {
	S: "size-3",
	M: "size-3.5",
	L: "size-4",
};

/** What an explorer shows: its folder's children, or that the folder is gone. */
export type ExplorerData = { missing: true } | { children: FavouriteNode[] };

function folderIds(nodes: FavouriteNode[]): string[] {
	return nodes.flatMap((n) =>
		n.kind === "folder" ? [n.id, ...folderIds(n.children ?? [])] : [],
	);
}

/**
 * The children of an explorer's folder as a file tree, below its title as a
 * heading of headingLevel when it has one. In edit mode, links don't navigate.
 * With onAdd, a button left of the expand-all button calls it. With onDetails,
 * an info button at the end of each link row calls it with the link's ID.
 * A search button opens a field that filters the tree in the browser; while
 * it has terms, every shown folder is open, without changing the stored ones.
 */
export function ExplorerTree({
	widgetId,
	size,
	data,
	editing,
	title = null,
	headingLevel = 2,
	onAdd,
	onDetails,
	titleLink,
}: {
	widgetId: string;
	size: WidgetSize;
	data: ExplorerData;
	editing: boolean;
	title?: string | null;
	headingLevel?: number;
	onAdd?: () => void;
	onDetails?: (id: string) => void;
	/** Makes the title a link to /favourites with these search params. */
	titleLink?: FavouritesSearch;
}) {
	const { t } = useTranslation();
	const [expanded, setExpanded] = useExpanded(widgetId);
	const style = sizeStyles[size];
	const searchId = useId();
	const [searching, setSearching] = useState(false);
	const [query, setQuery] = useState("");
	const searchButton = useRef<HTMLButtonElement>(null);

	const children = "missing" in data ? [] : data.children;
	const terms = useMemo(() => searchTerms(query), [query]);
	const filtering = terms.length > 0;
	const shown = useMemo(
		() =>
			filtering
				? (filterTree(
						{ id: "", kind: "folder", title: null, children },
						{ terms, tags: [] },
					)?.children ?? [])
				: children,
		[children, terms, filtering],
	);
	const closeSearch = () => {
		setSearching(false);
		setQuery("");
		searchButton.current?.focus();
	};
	const folders = folderIds(children);
	const anyExpanded = folders.some((id) => expanded.has(id));
	const toggleAllLabel = t(
		anyExpanded
			? "dashboards.widgets.favouriteExplorer.collapseAll"
			: "dashboards.widgets.favouriteExplorer.expandAll",
	);
	const toggle = (id: string) => {
		const next = new Set(expanded);
		if (!next.delete(id)) next.add(id);
		setExpanded(next);
	};

	const renderNodes = (nodes: FavouriteNode[], depth: number) => (
		<ul>
			{nodes.map((node) => {
				const name = label(node);
				const indent = { paddingLeft: `${depth * style.indent}px` };
				// At least as wide as the widget, wider when a deep row needs it.
				const row = `flex w-min min-w-[100cqw] items-center gap-1.5 rounded ${style.row} hover:bg-paper`;
				const text = <Highlighted text={name} terms={terms} />;
				if (node.kind === "folder") {
					// While searching, every shown folder is open and the stored ones stay as they are.
					const open = filtering || expanded.has(node.id);
					return (
						<li key={node.id}>
							<button
								type="button"
								aria-expanded={open}
								onClick={filtering ? undefined : () => toggle(node.id)}
								className={`${row} text-left`}
								style={indent}
							>
								<ChevronRight
									className={`${style.icon} shrink-0 text-muted transition-transform motion-reduce:transition-none ${open ? "rotate-90" : ""}`}
									aria-hidden
								/>
								<Folder
									className={`${style.icon} shrink-0 text-muted`}
									aria-hidden
								/>
								<span
									className="min-w-24 max-w-sm flex-1 truncate font-medium contain-inline-size"
									title={name}
								>
									{text}
								</span>
							</button>
							{open && renderNodes(node.children ?? [], depth + 1)}
						</li>
					);
				}
				const content = (
					<>
						<span className={`${style.icon} shrink-0`} />
						<ImageIcon imageId={node.iconImageId} className={style.icon} />
						<span className="min-w-24 max-w-sm flex-1 truncate contain-inline-size">
							{text}
						</span>
					</>
				);
				const linkRow = onDetails
					? `flex min-w-0 flex-1 items-center gap-1.5 rounded ${style.row} hover:bg-paper`
					: row;
				return (
					<li
						key={node.id}
						className={
							onDetails ? "flex w-min min-w-[100cqw] items-center" : undefined
						}
					>
						{editing ? (
							<span className={linkRow} style={indent} title={node.url}>
								{content}
							</span>
						) : (
							<a
								href={node.url}
								target="_blank"
								rel="noopener noreferrer"
								className={linkRow}
								style={indent}
								title={node.url}
							>
								{content}
							</a>
						)}
						{onDetails && (
							<DetailsButton
								link={node}
								onClick={() => onDetails(node.id)}
								className="inline-flex shrink-0 items-center justify-center rounded p-1 text-muted hover:bg-paper hover:text-ink"
								iconClassName={detailsIcon[size]}
							/>
						)}
					</li>
				);
			})}
		</ul>
	);

	let body: ReactNode;
	if ("missing" in data) {
		body = (
			<p className="text-muted">
				{t("dashboards.widgets.favouriteExplorer.missing")}
			</p>
		);
	} else if (children.length === 0) {
		body = (
			<p className="text-muted">
				{t("dashboards.widgets.favouriteExplorer.empty")}
			</p>
		);
	} else if (shown.length === 0) {
		body = (
			<p className="text-muted">
				{t("dashboards.widgets.favouriteExplorer.noMatches")}
			</p>
		);
	} else {
		// Deep rows scroll sideways inside the widget instead of hiding names.
		body = (
			<div className="@container overflow-x-auto" data-testid="explorer-scroll">
				<div className="w-max min-w-full">{renderNodes(shown, 0)}</div>
			</div>
		);
	}

	const searchLabel = t("dashboards.widgets.favouriteExplorer.search");
	const closeLabel = t("dashboards.widgets.favouriteExplorer.closeSearch");
	const searchToggle = children.length > 0 && (
		<button
			ref={searchButton}
			type="button"
			onClick={() => (searching ? closeSearch() : setSearching(true))}
			aria-label={searchLabel}
			title={searchLabel}
			aria-expanded={searching}
			aria-controls={searchId}
			className={iconButton}
		>
			<Search className="size-4" aria-hidden />
		</button>
	);
	const searchBar = searching && children.length > 0 && (
		<div
			id={searchId}
			className="mb-1 flex items-center gap-1.5 rounded-md border border-line bg-paper/40 pl-2"
		>
			<Search className={`${style.icon} shrink-0 text-muted`} aria-hidden />
			<input
				type="search"
				// biome-ignore lint/a11y/noAutofocus: the viewer just asked for the field
				autoFocus
				value={query}
				onChange={(e) => setQuery(e.target.value)}
				onKeyDown={(e) => {
					if (e.key === "Escape") {
						e.preventDefault();
						e.stopPropagation();
						closeSearch();
					}
				}}
				aria-label={t("dashboards.widgets.favouriteExplorer.searchLabel")}
				className={`min-w-0 flex-1 bg-transparent ${style.row} outline-none [&::-webkit-search-cancel-button]:hidden`}
			/>
			<button
				type="button"
				onClick={closeSearch}
				aria-label={closeLabel}
				title={closeLabel}
				className={iconButton}
			>
				<X className="size-4" aria-hidden />
			</button>
		</div>
	);

	const toggleAll = folders.length > 0 && !filtering && (
		<button
			type="button"
			onClick={() => setExpanded(anyExpanded ? new Set() : new Set(folders))}
			aria-label={toggleAllLabel}
			title={toggleAllLabel}
			className={iconButton}
		>
			{anyExpanded ? (
				<ChevronsDownUp className="size-4" aria-hidden />
			) : (
				<ChevronsUpDown className="size-4" aria-hidden />
			)}
		</button>
	);

	const addLabel = t("dashboards.widgets.favouriteExplorer.addFavourite");
	const buttons = (
		<>
			{onAdd && (
				<button
					type="button"
					onClick={onAdd}
					aria-label={addLabel}
					title={addLabel}
					className={iconButton}
				>
					<Plus className="size-4" aria-hidden />
				</button>
			)}
			{searchToggle}
			{toggleAll}
		</>
	);

	// With a title, the buttons are on the title's line; without, beside the tree.
	if (title) {
		const Heading = `h${headingLevel}` as "h2";
		return (
			<div className={style.text}>
				<div className="flex items-center gap-1">
					<Heading className="min-w-0 flex-1 truncate font-medium">
						{titleLink ? (
							<Link
								to="/favourites"
								search={titleLink}
								className="hover:underline"
							>
								{title}
							</Link>
						) : (
							title
						)}
					</Heading>
					{buttons}
				</div>
				{searchBar}
				{body}
			</div>
		);
	}
	return (
		<div className={`flex items-start gap-1 ${style.text}`}>
			<div className="min-w-0 flex-1">
				{searchBar}
				{body}
			</div>
			{buttons}
		</div>
	);
}
