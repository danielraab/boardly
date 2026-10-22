import {
	ChevronRight,
	ChevronsDownUp,
	ChevronsUpDown,
	Folder,
} from "lucide-react";
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
 * The children of an explorer's folder as a file tree. In edit mode, links
 * don't navigate.
 */
export function ExplorerTree({
	widgetId,
	size,
	data,
	editing,
}: {
	widgetId: string;
	size: WidgetSize;
	data: ExplorerData;
	editing: boolean;
}) {
	const { t } = useTranslation();
	const [expanded, setExpanded] = useExpanded(widgetId);
	const style = sizeStyles[size];

	if ("missing" in data) {
		return (
			<p className={`${style.text} text-muted`}>
				{t("dashboards.widgets.favouriteExplorer.missing")}
			</p>
		);
	}
	if (data.children.length === 0) {
		return (
			<p className={`${style.text} text-muted`}>
				{t("dashboards.widgets.favouriteExplorer.empty")}
			</p>
		);
	}

	const folders = folderIds(data.children);
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

	return (
		<div className={`flex items-start gap-1 ${style.text}`}>
			<div className="min-w-0 flex-1">{renderNodes(data.children, 0)}</div>
			{folders.length > 0 && (
				<button
					type="button"
					onClick={() =>
						setExpanded(anyExpanded ? new Set() : new Set(folders))
					}
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
			)}
		</div>
	);
}
