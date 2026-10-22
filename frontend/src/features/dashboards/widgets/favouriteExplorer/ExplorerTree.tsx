import {
	ChevronRight,
	ChevronsDownUp,
	ChevronsUpDown,
	Folder,
	Plus,
} from "lucide-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { iconButton } from "#/components/buttonStyles";
import { ImageIcon } from "#/components/ImageIcon";
import type { FavouriteNode } from "#/features/favourites/api";
import { label } from "#/features/favourites/tree";
import { sizeStyles, type WidgetSize } from "../sizes";
import { useExpanded } from "./useExpanded";

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
 * With onAdd, a button left of the expand-all button calls it.
 */
export function ExplorerTree({
	widgetId,
	size,
	data,
	editing,
	title = null,
	headingLevel = 2,
	onAdd,
}: {
	widgetId: string;
	size: WidgetSize;
	data: ExplorerData;
	editing: boolean;
	title?: string | null;
	headingLevel?: number;
	onAdd?: () => void;
}) {
	const { t } = useTranslation();
	const [expanded, setExpanded] = useExpanded(widgetId);
	const style = sizeStyles[size];

	const children = "missing" in data ? [] : data.children;
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
				const row = `flex w-full min-w-0 items-center gap-1.5 rounded ${style.row} hover:bg-paper`;
				if (node.kind === "folder") {
					const open = expanded.has(node.id);
					return (
						<li key={node.id}>
							<button
								type="button"
								aria-expanded={open}
								onClick={() => toggle(node.id)}
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
								<span className="truncate font-medium">{name}</span>
							</button>
							{open && renderNodes(node.children ?? [], depth + 1)}
						</li>
					);
				}
				const content = (
					<>
						<span className={`${style.icon} shrink-0`} />
						<ImageIcon imageId={node.iconImageId} className={style.icon} />
						<span className="truncate">{name}</span>
					</>
				);
				return (
					<li key={node.id}>
						{editing ? (
							<span className={row} style={indent} title={node.url}>
								{content}
							</span>
						) : (
							<a
								href={node.url}
								target="_blank"
								rel="noopener noreferrer"
								className={row}
								style={indent}
								title={node.url}
							>
								{content}
							</a>
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
	} else {
		body = renderNodes(children, 0);
	}

	const toggleAll = folders.length > 0 && (
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
						{title}
					</Heading>
					{buttons}
				</div>
				{body}
			</div>
		);
	}
	return (
		<div className={`flex items-start gap-1 ${style.text}`}>
			<div className="min-w-0 flex-1">{body}</div>
			{buttons}
		</div>
	);
}
