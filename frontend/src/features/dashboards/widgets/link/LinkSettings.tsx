import { useQuery, useQueryClient } from "@tanstack/react-query";
import { type ChangeEvent, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { secondaryButton } from "#/components/buttonStyles";
import { textInput } from "#/components/dialogStyles";
import {
	createFavourite,
	favouritesQueryKey,
	favouritesQueryOptions,
} from "#/features/favourites/api";
import {
	checkLinkTitle,
	checkUrl,
	fieldErrorFromCode,
} from "#/features/favourites/validate";
import { ApiError } from "#/lib/api";
import type { SectionHeight } from "../../api";
import { folderOptions, linkOptions } from "../favouriteOptions";
import type { WidgetSettingsProps } from "../registry";
import { type WidgetSize, widgetSizes } from "../sizes";
import { WidgetSettingsForm } from "../WidgetSettingsForm";
import { faviconFor, uploadImage } from "./api";
import { type LinkIconKey, linkIcons } from "./icons";
import { LinkIcon } from "./LinkView";

export type LinkConfig = {
	source: "favourite" | "url";
	/** Only for source "favourite", with its last known title. */
	favouriteId?: string;
	favouriteTitle?: string | null;
	/** The URL, or the favourite's last known URL. */
	url: string;
	title: string | null;
	imageId: string | null;
	icon: string | null;
	target: "_blank" | "_self";
	size: WidgetSize;
};

export const maxLinkWidgetTitle = 200;

type Source = "favourite" | "new" | "url";

/** How long the URL has to stay unchanged before its favicon is looked up. */
export const faviconDelayMs = 500;

const imageTypes = "image/png,image/jpeg,image/webp,image/gif,image/x-icon";

export function LinkSettings(props: WidgetSettingsProps) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const config = props.widget.config as LinkConfig;
	const favourites = useQuery(favouritesQueryOptions);

	const [source, setSource] = useState<Source>(config.source);
	const [favouriteId, setFavouriteId] = useState(config.favouriteId ?? "");
	const [url, setUrl] = useState(config.source === "url" ? config.url : "");
	const [newUrl, setNewUrl] = useState("");
	const [newTitle, setNewTitle] = useState("");
	const [folderId, setFolderId] = useState("");
	const [title, setTitle] = useState(config.title ?? "");
	// auto: filled with the URL's favicon, so a new URL may replace it.
	const [image, setImage] = useState({ id: config.imageId, auto: false });
	const [icon, setIcon] = useState(config.icon ?? "");
	const [target, setTarget] = useState(config.target);
	const [size, setSize] = useState(config.size);
	const [uploadError, setUploadError] = useState<string | null>(null);
	const [createError, setCreateError] = useState<string | null>(null);
	const [creating, setCreating] = useState(false);

	const links = favourites.data ? linkOptions(favourites.data) : [];
	const folders = favourites.data
		? folderOptions(favourites.data, t("favourites.title"))
		: [];
	const favourite = links.find((l) => l.node.id === favouriteId)?.node;
	const folder = folderId || folders[0]?.node.id || "";

	const urlError = url.trim() ? checkUrl(url) : null;
	const newUrlError = newUrl.trim() ? checkUrl(newUrl) : null;
	const newTitleError = checkLinkTitle(newTitle);
	let linkUrl: string | null = null;
	if (source === "favourite") linkUrl = favourite?.url ?? null;
	else if (source === "url" && url.trim() && !urlError) linkUrl = url.trim();
	else if (source === "new" && newUrl.trim() && !newUrlError)
		linkUrl = newUrl.trim();
	const known = source === "favourite" ? favourite?.iconImageId : undefined;

	// Fill an empty or auto-filled image with the favicon of a newly chosen URL.
	const imageRef = useRef(image);
	imageRef.current = image;
	const openedUrl = useRef(props.isNew ? null : config.url);
	useEffect(() => {
		if (!linkUrl || linkUrl === openedUrl.current) return;
		openedUrl.current = null;
		const current = imageRef.current;
		if (current.id && !current.auto) return;
		if (known) {
			setImage({ id: known, auto: true });
			return;
		}
		let cancelled = false;
		const timer = setTimeout(() => {
			faviconFor(linkUrl).then(
				(id) => {
					if (!cancelled) setImage({ id, auto: true });
				},
				() => {
					// No favicon (or a blocked address): an old auto-filled one doesn't fit.
					if (!cancelled && imageRef.current.auto)
						setImage({ id: null, auto: false });
				},
			);
		}, faviconDelayMs);
		return () => {
			cancelled = true;
			clearTimeout(timer);
		};
	}, [linkUrl, known]);

	async function onUpload(event: ChangeEvent<HTMLInputElement>) {
		const file = event.target.files?.[0];
		event.target.value = "";
		if (!file) return;
		setUploadError(null);
		try {
			setImage({ id: await uploadImage(file), auto: false });
		} catch (err) {
			const code = err instanceof ApiError ? err.code : "";
			setUploadError(
				code === "unsupported_image" || code === "image_too_large"
					? t(`dashboards.widgets.link.errors.${code}`)
					: t("dashboards.widgets.link.errors.uploadFailed"),
			);
		}
	}

	let valid = false;
	if (source === "favourite") valid = favourite !== undefined;
	else if (source === "url") valid = linkUrl !== null;
	else valid = linkUrl !== null && !newTitleError && folder !== "";

	async function apply(dims: { width: number; height: SectionHeight }) {
		const common = {
			title: title.trim() || null,
			imageId: image.id,
			icon: icon || null,
			target,
			size,
		};
		let next: LinkConfig;
		if (source === "url" && linkUrl) {
			next = { source: "url", url: linkUrl, ...common };
		} else {
			let node = favourite;
			if (source === "new" && linkUrl) {
				setCreating(true);
				setCreateError(null);
				try {
					node = await createFavourite({
						parentId: folder,
						kind: "link",
						url: linkUrl,
						title: newTitle.trim(),
						tags: [],
					});
				} catch (err) {
					const field =
						err instanceof ApiError
							? fieldErrorFromCode(err.code, "link")
							: null;
					setCreateError(
						field
							? t(`favourites.errors.${field}`)
							: t("dashboards.widgets.link.errors.createFailed"),
					);
					return;
				} finally {
					setCreating(false);
				}
				void queryClient.invalidateQueries({ queryKey: favouritesQueryKey });
			}
			if (!node?.url) return;
			next = {
				source: "favourite",
				favouriteId: node.id,
				favouriteTitle: node.title,
				url: node.url,
				...common,
			};
		}
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
			valid={valid && !creating}
			onApply={(dims) => void apply(dims)}
			onDelete={props.onDelete}
		>
			<fieldset>
				<legend className="text-sm font-medium">
					{t("dashboards.widgets.link.sourceLabel")}
				</legend>
				<div className="mt-2 flex flex-wrap gap-x-5 gap-y-2 text-sm">
					{(["favourite", "new", "url"] as const).map((s) => (
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
					<label htmlFor="link-favourite" className="block text-sm font-medium">
						{t("dashboards.widgets.link.favouriteLabel")}
					</label>
					<select
						id="link-favourite"
						value={favourite ? favouriteId : ""}
						onChange={(e) => setFavouriteId(e.target.value)}
						disabled={!favourites.data}
						className={textInput}
					>
						<option value="" disabled>
							{t("dashboards.widgets.link.favouritePlaceholder")}
						</option>
						{links.map(({ node, path }) => (
							<option key={node.id} value={node.id}>
								{path}
							</option>
						))}
					</select>
					{favourites.data && favouriteId !== "" && !favourite && (
						<p className="mt-1 text-sm text-muted">
							{t("dashboards.widgets.link.favouriteMissing")}
						</p>
					)}
				</div>
			)}

			{source === "new" && (
				<>
					<UrlField
						id="link-new-url"
						value={newUrl}
						onChange={setNewUrl}
						error={newUrlError}
					/>
					<div>
						<label
							htmlFor="link-new-title"
							className="block text-sm font-medium"
						>
							{t("dashboards.widgets.link.newTitleLabel")}
						</label>
						<input
							id="link-new-title"
							value={newTitle}
							onChange={(e) => setNewTitle(e.target.value)}
							className={textInput}
						/>
						{newTitleError && (
							<p role="alert" className="mt-1 text-sm text-red-700">
								{t(`favourites.errors.${newTitleError}`)}
							</p>
						)}
					</div>
					<div>
						<label htmlFor="link-folder" className="block text-sm font-medium">
							{t("dashboards.widgets.link.folderLabel")}
						</label>
						<select
							id="link-folder"
							value={folder}
							onChange={(e) => setFolderId(e.target.value)}
							disabled={!favourites.data}
							className={textInput}
						>
							{folders.map(({ node, path }) => (
								<option key={node.id} value={node.id}>
									{path}
								</option>
							))}
						</select>
					</div>
				</>
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
					maxLength={maxLinkWidgetTitle}
					onChange={(e) => setTitle(e.target.value)}
					aria-describedby="link-title-hint"
					className={textInput}
				/>
				<p id="link-title-hint" className="mt-1 text-sm text-muted">
					{t("dashboards.widgets.link.titleHint")}
				</p>
			</div>

			<div>
				<span className="block text-sm font-medium">
					{t("dashboards.widgets.link.imageLabel")}
				</span>
				<div className="mt-1 flex flex-wrap items-center gap-3">
					<span
						className="inline-flex size-10 items-center justify-center rounded-lg border border-line"
						data-testid="link-image-preview"
					>
						<LinkIcon
							imageId={image.id}
							icon={icon || null}
							className="size-6"
						/>
					</span>
					<label className={`${secondaryButton} cursor-pointer`}>
						{t("dashboards.widgets.link.upload")}
						<input
							type="file"
							accept={imageTypes}
							onChange={(e) => void onUpload(e)}
							className="sr-only"
						/>
					</label>
					{image.id && (
						<button
							type="button"
							onClick={() => setImage({ id: null, auto: false })}
							className={secondaryButton}
						>
							{t("dashboards.widgets.link.removeImage")}
						</button>
					)}
				</div>
				{uploadError && (
					<p role="alert" className="mt-1 text-sm text-red-700">
						{uploadError}
					</p>
				)}
			</div>

			<div className="grid gap-4 sm:grid-cols-3">
				<div>
					<label htmlFor="link-icon" className="block text-sm font-medium">
						{t("dashboards.widgets.link.iconLabel")}
					</label>
					<select
						id="link-icon"
						value={icon}
						onChange={(e) => setIcon(e.target.value)}
						className={textInput}
					>
						<option value="">{t("dashboards.widgets.link.iconNone")}</option>
						{(Object.keys(linkIcons) as LinkIconKey[]).map((key) => (
							<option key={key} value={key}>
								{t(`dashboards.widgets.link.icons.${key}`)}
							</option>
						))}
					</select>
				</div>
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
			{createError && (
				<p role="alert" className="text-sm text-red-700">
					{createError}
				</p>
			)}
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
				className={textInput}
			/>
			{error && (
				<p role="alert" className="mt-1 text-sm text-red-700">
					{t(`favourites.errors.${error}`)}
				</p>
			)}
		</div>
	);
}
