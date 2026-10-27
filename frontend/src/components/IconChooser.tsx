import { Popover } from "@base-ui/react/popover";
import { Tabs } from "@base-ui/react/tabs";
import { Check, ChevronDown, Search, SmilePlus } from "lucide-react";
import { type ChangeEvent, type ReactNode, useState } from "react";
import { useTranslation } from "react-i18next";
import { primaryButton, secondaryButton } from "#/components/buttonStyles";
import { ApiError } from "#/lib/api";
import { uploadImage } from "#/lib/images";
import { ChosenIcon } from "./icons/ChosenIcon";
import { symbols as allSymbols, type SymbolKey } from "./icons/symbols";

export type IconSource = "favicon" | "image" | "symbol" | "none";

/**
 * An icon while it is being chosen. The image and the symbol are kept while
 * another source is chosen, so switching back restores them.
 */
export type IconChoice = {
	source: IconSource;
	imageId: string | null;
	icon: string | null;
};

/**
 * "none" has no tab: it is chosen with the footer's "Remove icon" button.
 * Without a website icon there is no "favicon" tab either.
 */
const tabsFor = (withFavicon: boolean): IconSource[] =>
	withFavicon ? ["favicon", "image", "symbol"] : ["image", "symbol"];

/** The tab the chooser opens on for a source. */
const tabFor = (source: IconSource, withFavicon: boolean) =>
	source === "none" || (source === "favicon" && !withFavicon)
		? withFavicon
			? "favicon"
			: "symbol"
		: source;

const imageTypes = "image/png,image/jpeg,image/webp,image/gif,image/x-icon";

const tab =
	"min-w-0 truncate whitespace-nowrap rounded-md px-2 py-1.5 text-sm text-muted outline-none hover:text-ink focus-visible:ring-2 focus-visible:ring-accent data-[selected]:bg-paper data-[selected]:font-medium data-[selected]:text-ink";

/** The image ID and symbol that show a choice, given the URL's favicon. */
export function shownIcon(choice: IconChoice, faviconId: string | null) {
	return {
		imageId:
			choice.source === "favicon"
				? faviconId
				: choice.source === "image"
					? choice.imageId
					: null,
		icon: choice.source === "symbol" ? choice.icon : null,
	};
}

/**
 * An icon field: a button with the current icon that opens a chooser with one
 * tab per source. Choices apply right away; the chooser stays open until
 * Done, Escape or a click outside. Without favicon, there is no "Website
 * icon" tab. The "field" variant is a labelled full-width button for forms,
 * "button" a square icon button.
 */
export function IconChooser({
	value,
	onChange,
	favicon,
	host = null,
	variant = "field",
}: {
	value: IconChoice;
	onChange: (value: IconChoice) => void;
	/** The website icon of the link's URL; id null when there is none. */
	favicon?: { id: string | null; loading: boolean };
	/** The host of the link's URL, or null without a URL. */
	host?: string | null;
	variant?: "field" | "button";
}) {
	const { t } = useTranslation();
	const withFavicon = favicon !== undefined;
	const tabs = tabsFor(withFavicon);
	const [open, setOpen] = useState(false);
	const [tabValue, setTabValue] = useState<IconSource>(
		tabFor(value.source, withFavicon),
	);
	const [filter, setFilter] = useState("");
	const [uploadError, setUploadError] = useState<string | null>(null);
	const [uploading, setUploading] = useState(false);

	const symbolName = (key: string) => t(`symbols.${key as SymbolKey}`);
	const kindName =
		value.source === "symbol" && value.icon
			? symbolName(value.icon)
			: t(`iconChooser.tabs.${value.source}`);
	const shown = shownIcon(value, favicon?.id ?? null);
	const choose = (source: IconSource, fields: Partial<IconChoice> = {}) =>
		onChange({ ...value, ...fields, source });

	async function onUpload(event: ChangeEvent<HTMLInputElement>) {
		const file = event.target.files?.[0];
		event.target.value = "";
		if (!file) return;
		setUploadError(null);
		setUploading(true);
		try {
			choose("image", { imageId: await uploadImage(file) });
		} catch (err) {
			const code = err instanceof ApiError ? err.code : "";
			setUploadError(
				code === "unsupported_image" || code === "image_too_large"
					? t(`iconChooser.errors.${code}`)
					: t("iconChooser.errors.uploadFailed"),
			);
		} finally {
			setUploading(false);
		}
	}

	/**
	 * A button that makes source the icon. It stays (pressed) once it is, so
	 * focus doesn't fall out of the chooser.
	 */
	const chooseButton = (
		source: IconSource,
		label: string,
		fields = {},
		inUseLabel = t("iconChooser.inUse"),
	) => {
		const inUse = value.source === source;
		return (
			<button
				type="button"
				aria-pressed={inUse}
				onClick={() => !inUse && choose(source, fields)}
				className={`${secondaryButton} aria-pressed:bg-paper`}
			>
				{inUse && <Check className="size-4" aria-hidden />}
				{inUse ? inUseLabel : label}
			</button>
		);
	};

	const preview = (content: ReactNode) => (
		<span className="inline-flex size-14 shrink-0 items-center justify-center rounded-lg border border-line bg-paper/40">
			{content}
		</span>
	);

	let faviconText: string;
	if (!host) faviconText = t("iconChooser.favicon.noUrl");
	else if (favicon?.loading) faviconText = t("iconChooser.favicon.loading");
	else if (!favicon?.id)
		faviconText = t("iconChooser.favicon.none", {
			host,
		});
	else
		faviconText = t("iconChooser.favicon.found", {
			host,
		});

	const q = filter.trim().toLowerCase();
	const symbols = (Object.keys(allSymbols) as SymbolKey[]).filter(
		(key) => !q || symbolName(key).toLowerCase().includes(q) || key.includes(q),
	);

	return (
		<div>
			{variant === "field" && (
				<span className="block text-sm font-medium">
					{t("iconChooser.label")}
				</span>
			)}
			<Popover.Root
				open={open}
				onOpenChange={(next) => {
					setOpen(next);
					if (next) setTabValue(tabFor(value.source, withFavicon));
				}}
			>
				<Popover.Trigger
					aria-label={t("iconChooser.button", {
						name: kindName,
					})}
					title={variant === "button" ? kindName : undefined}
					className={
						variant === "field"
							? `${secondaryButton} mt-2 w-full justify-start`
							: `${secondaryButton} size-10 justify-center p-0`
					}
				>
					<ChosenIcon
						{...shown}
						className="size-5"
						fallback={withFavicon ? undefined : SmilePlus}
					/>
					{variant === "field" && (
						<>
							<span className="min-w-0 flex-1 truncate text-left">
								{kindName}
							</span>
							<ChevronDown className="size-4 shrink-0 text-muted" aria-hidden />
						</>
					)}
				</Popover.Trigger>
				<Popover.Portal>
					{/* Above dialogs (z-40), which the chooser is often opened from. */}
					<Popover.Positioner sideOffset={6} align="start" className="z-50">
						<Popover.Popup className="w-[22rem] max-w-[calc(100vw-2rem)] rounded-xl border border-line bg-surface p-3 text-ink shadow-[0_8px_24px_-12px_var(--color-shadow)] outline-none origin-[var(--transform-origin)] transition-[opacity,scale] duration-100 data-[starting-style]:scale-95 data-[starting-style]:opacity-0 data-[ending-style]:opacity-0 motion-reduce:transition-none">
							<Popover.Title className="sr-only">
								{t("iconChooser.label")}
							</Popover.Title>
							<Tabs.Root
								value={tabValue}
								onValueChange={(v) => setTabValue(v as IconSource)}
							>
								<Tabs.List className="flex gap-1 border-b border-line pb-2">
									{tabs.map((s) => (
										<Tabs.Tab key={s} value={s} className={tab}>
											{t(`iconChooser.tabs.${s}`)}
										</Tabs.Tab>
									))}
								</Tabs.List>

								{withFavicon && (
									<Tabs.Panel value="favicon" className="pt-3">
										<div className="flex items-center gap-3">
											{preview(
												<ChosenIcon
													imageId={favicon?.id ?? null}
													icon={null}
													className="size-8"
												/>,
											)}
											<p className="text-sm text-muted">{faviconText}</p>
										</div>
										<div className="mt-3">
											{chooseButton("favicon", t("iconChooser.favicon.use"))}
										</div>
									</Tabs.Panel>
								)}

								<Tabs.Panel value="image" className="pt-3">
									<div className="flex items-center gap-3">
										{preview(
											<ChosenIcon
												imageId={value.imageId}
												icon={null}
												className="size-8"
											/>,
										)}
										<p className="text-sm text-muted">
											{t("iconChooser.image.hint")}
										</p>
									</div>
									<div className="mt-3 flex flex-wrap items-center gap-2">
										<label
											className={`${secondaryButton} cursor-pointer ${uploading ? "opacity-50" : ""}`}
										>
											{t(
												value.imageId
													? "iconChooser.image.replace"
													: "iconChooser.image.upload",
											)}
											<input
												type="file"
												accept={imageTypes}
												disabled={uploading}
												onChange={(e) => void onUpload(e)}
												className="sr-only"
											/>
										</label>
										{value.imageId &&
											chooseButton("image", t("iconChooser.image.use"))}
									</div>
									{uploadError && (
										<p role="alert" className="mt-2 text-sm text-red-700">
											{uploadError}
										</p>
									)}
								</Tabs.Panel>

								<Tabs.Panel value="symbol" className="pt-3">
									<div className="flex items-center gap-2 rounded-lg border border-line bg-paper/40 px-3">
										<Search
											className="size-4 shrink-0 text-muted"
											aria-hidden
										/>
										<input
											type="search"
											value={filter}
											onChange={(e) => setFilter(e.target.value)}
											placeholder={t("iconChooser.symbol.filter")}
											aria-label={t("iconChooser.symbol.filter")}
											className="min-w-0 flex-1 bg-transparent py-1.5 text-sm outline-none"
										/>
									</div>
									<div
										role="radiogroup"
										aria-label={t("iconChooser.symbol.group")}
										className="mt-2 grid max-h-52 grid-cols-6 gap-1 overflow-auto p-0.5"
									>
										{symbols.map((key) => {
											const Icon = allSymbols[key];
											return (
												<label
													key={key}
													title={symbolName(key)}
													className="flex aspect-square cursor-pointer items-center justify-center rounded-md text-muted hover:bg-paper hover:text-ink has-[:checked]:bg-accent/15 has-[:checked]:text-ink has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-accent"
												>
													<input
														type="radio"
														name="icon-symbol"
														value={key}
														checked={
															value.source === "symbol" && value.icon === key
														}
														onChange={() => choose("symbol", { icon: key })}
														className="sr-only"
													/>
													<Icon className="size-5" aria-hidden />
													<span className="sr-only">{symbolName(key)}</span>
												</label>
											);
										})}
									</div>
									{symbols.length === 0 && (
										<p className="mt-2 text-sm text-muted">
											{t("iconChooser.symbol.noMatches")}
										</p>
									)}
								</Tabs.Panel>
							</Tabs.Root>
							<div className="mt-3 flex justify-between gap-2 border-t border-line pt-3">
								{chooseButton(
									"none",
									t("iconChooser.remove"),
									{},
									t("iconChooser.removed"),
								)}
								<Popover.Close className={primaryButton}>
									{t("iconChooser.done")}
								</Popover.Close>
							</div>
						</Popover.Popup>
					</Popover.Positioner>
				</Popover.Portal>
			</Popover.Root>
		</div>
	);
}
