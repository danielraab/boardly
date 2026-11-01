import { useQuery } from "@tanstack/react-query";
import { Plus } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { secondaryButton } from "#/components/buttonStyles";
import { textInput } from "#/components/dialogStyles";
import {
	type IconChoice,
	IconChooser,
	type IconSource,
} from "#/components/IconChooser";
import {
	type FavouriteNode,
	favouritesQueryOptions,
} from "#/features/favourites/api";
import { DomainHint } from "#/features/favourites/DomainHint";
import { FavouritePicker } from "#/features/favourites/FavouritePicker";
import { NodeDialog } from "#/features/favourites/NodeDialog";
import { findNode } from "#/features/favourites/tree";
import { checkUrl } from "#/features/favourites/validate";
import { useDebounced } from "#/lib/useDebounced";
import type { SectionHeight } from "../../api";
import type { WidgetSettingsProps } from "../registry";
import { type WidgetSize, widgetSizes } from "../sizes";
import { maxWidgetTitle, WidgetSettingsForm } from "../WidgetSettingsForm";
import { useFavicon } from "./useFavicon";

export type LinkConfig = {
	source: "favourite" | "url";
	/** Only for source "favourite", with its last known title. */
	favouriteId?: string;
	favouriteTitle?: string | null;
	/** The URL, or the favourite's last known URL. */
	url: string;
	title: string | null;
	/** Missing in configs saved before icon sources; see iconSourceOf. */
	iconSource?: IconSource;
	/** Only for iconSource "image". */
	imageId: string | null;
	/** Only for iconSource "symbol": a key of linkIcons. */
	icon: string | null;
	target: "_blank" | "_self";
	size: WidgetSize;
};

/** How long the URL has to stay unchanged before its favicon is looked up. */
export const faviconDelayMs = 500;

/** The icon source of a config, also of one saved before sources existed. */
export function iconSourceOf(config: LinkConfig): IconSource {
	if (config.iconSource) return config.iconSource;
	if (config.imageId) return "image";
	return config.icon ? "symbol" : "favicon";
}

export function LinkSettings(props: WidgetSettingsProps) {
	const { t } = useTranslation();
	const config = props.widget.config as LinkConfig;
	const favourites = useQuery(favouritesQueryOptions);

	const [source, setSource] = useState(config.source);
	const [favouriteId, setFavouriteId] = useState(config.favouriteId ?? "");
	const [url, setUrl] = useState(config.source === "url" ? config.url : "");
	// A favourite created from here, until the refetched tree contains it.
	const [created, setCreated] = useState<FavouriteNode | null>(null);
	const [creating, setCreating] = useState(false);
	const [title, setTitle] = useState(config.title ?? "");
	const [icon, setIcon] = useState<IconChoice>({
		source: iconSourceOf(config),
		imageId: config.imageId,
		icon: config.icon,
	});
	const [target, setTarget] = useState(config.target);
	const [size, setSize] = useState(config.size);

	const found = favourites.data && findNode(favourites.data, favouriteId);
	const favourite =
		found?.kind === "link"
			? found
			: created?.id === favouriteId
				? created
				: undefined;

	const urlError = url.trim() ? checkUrl(url) : null;
	let linkUrl: string | null = null;
	if (source === "favourite") linkUrl = favourite?.url ?? null;
	else if (url.trim() && !urlError) linkUrl = url.trim();
	// The website icon: a favourite's known one, or looked up once the URL
	// stops changing (until then, it is still being looked for).
	const known = source === "favourite" ? favourite?.iconImageId : undefined;
	const settledUrl = useDebounced(linkUrl, faviconDelayMs);
	const lookedUp = useFavicon(settledUrl, known);
	const favicon =
		known || settledUrl === linkUrl
			? lookedUp
			: { id: null, loading: linkUrl !== null };
	const host = linkUrl ? new URL(linkUrl).host : null;

	const valid = linkUrl !== null;

	function apply(dims: { width: number; height: SectionHeight }) {
		const common = {
			title: title.trim() || null,
			iconSource: icon.source,
			imageId: icon.source === "image" ? icon.imageId : null,
			icon: icon.source === "symbol" ? icon.icon : null,
			target,
			size,
		};
		const next: LinkConfig =
			source === "favourite" && favourite?.url
				? {
						source: "favourite",
						favouriteId: favourite.id,
						favouriteTitle: favourite.title,
						url: favourite.url,
						...common,
					}
				: { source: "url", url: linkUrl ?? "", ...common };
		props.onApply({ ...props.widget, ...dims, config: next });
	}

	return (
		<WidgetSettingsForm
			title={t(
				props.isNew
					? "dashboards.widgets.link.addTitle"
					: "dashboards.widgets.link.editTitle",
			)}
			widget={props.widget}
			isNew={props.isNew}
			desktopColumns={props.desktopColumns}
			valid={valid}
			fitContent
			onApply={apply}
			onDelete={props.onDelete}
		>
			<fieldset>
				<legend className="text-sm font-medium">
					{t("dashboards.widgets.link.sourceLabel")}
				</legend>
				<div className="mt-2 flex flex-wrap gap-x-5 gap-y-2 text-sm">
					{(["favourite", "url"] as const).map((s) => (
						<label key={s} className="flex items-center gap-2">
							<input
								type="radio"
								name="link-source"
								checked={source === s}
								onChange={() => setSource(s)}
								className="size-4 accent-ink"
							/>
							{t(`dashboards.widgets.link.source.${s}`)}
						</label>
					))}
				</div>
			</fieldset>

			{source === "favourite" && (
				<div>
					<FavouritePicker
						id="link-favourite"
						label={t("dashboards.widgets.link.favouriteLabel")}
						mode="link"
						value={favourite ? favouriteId : ""}
						onChange={setFavouriteId}
					/>
					{favourites.data && favouriteId !== "" && !favourite && (
						<p className="mt-1 text-sm text-muted">
							{t("dashboards.widgets.link.favouriteMissing")}
						</p>
					)}
					<button
						type="button"
						onClick={() => setCreating(true)}
						disabled={!favourites.data}
						className={`${secondaryButton} mt-2`}
					>
						<Plus className="size-4" aria-hidden />
						{t("dashboards.widgets.link.newFavourite")}
					</button>
				</div>
			)}

			{source === "url" && (
				<UrlField
					id="link-url"
					value={url}
					onChange={setUrl}
					error={urlError}
				/>
			)}

			<div>
				<label htmlFor="link-title" className="block text-sm font-medium">
					{t("dashboards.widgets.link.titleLabel")}
				</label>
				<input
					id="link-title"
					value={title}
					maxLength={maxWidgetTitle}
					onChange={(e) => setTitle(e.target.value)}
					aria-describedby="link-title-hint"
					className={textInput}
				/>
				<p id="link-title-hint" className="mt-1 text-sm text-muted">
					{t("dashboards.widgets.link.titleHint")}
				</p>
			</div>

			<IconChooser
				value={icon}
				onChange={setIcon}
				favicon={favicon}
				host={host}
			/>

			<div className="grid gap-4 sm:grid-cols-2">
				<div>
					<label htmlFor="link-target" className="block text-sm font-medium">
						{t("dashboards.widgets.link.targetLabel")}
					</label>
					<select
						id="link-target"
						value={target}
						onChange={(e) => setTarget(e.target.value as LinkConfig["target"])}
						className={textInput}
					>
						<option value="_blank">
							{t("dashboards.widgets.link.target._blank")}
						</option>
						<option value="_self">
							{t("dashboards.widgets.link.target._self")}
						</option>
					</select>
				</div>
				<div>
					<label htmlFor="link-size" className="block text-sm font-medium">
						{t("dashboards.widgets.sizeLabel")}
					</label>
					<select
						id="link-size"
						value={size}
						onChange={(e) => setSize(e.target.value as WidgetSize)}
						className={textInput}
					>
						{widgetSizes.map((s) => (
							<option key={s} value={s}>
								{t(`dashboards.widgets.size.${s}`)}
							</option>
						))}
					</select>
				</div>
			</div>
			<NodeDialog
				target={
					creating && favourites.data
						? {
								kind: "link",
								parent: favourites.data,
								parentLabel: t("favourites.title"),
							}
						: null
				}
				chooseFolder
				onSaved={(node) => {
					setCreated(node);
					setFavouriteId(node.id);
				}}
				onClose={() => setCreating(false)}
			/>
		</WidgetSettingsForm>
	);
}

function UrlField({
	id,
	value,
	onChange,
	error,
}: {
	id: string;
	value: string;
	onChange: (value: string) => void;
	error: ReturnType<typeof checkUrl>;
}) {
	const { t } = useTranslation();
	return (
		<div>
			<label htmlFor={id} className="block text-sm font-medium">
				{t("dashboards.widgets.link.urlLabel")}
			</label>
			<input
				id={id}
				type="url"
				value={value}
				onChange={(e) => onChange(e.target.value)}
				placeholder="https://"
				aria-invalid={error !== null || undefined}
				aria-describedby={`${id}-domain`}
				className={textInput}
			/>
			<DomainHint id={`${id}-domain`} url={value} />
			{error && (
				<p role="alert" className="mt-1 text-sm text-red-700">
					{t(`favourites.errors.${error}`)}
				</p>
			)}
		</div>
	);
}
