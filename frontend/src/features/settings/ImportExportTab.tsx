import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ArchiveRestore, FileUp } from "lucide-react";
import { type FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import { primaryButton } from "#/components/buttonStyles";
import { textInput } from "#/components/dialogStyles";
import {
	dashboardsQueryKey,
	dashboardViewQueryKey,
} from "#/features/dashboards/api";
import {
	favouritesQueryKey,
	tagUsageQueryKey,
} from "#/features/favourites/api";
import { FavouritePicker } from "#/features/favourites/FavouritePicker";
import { mediaQueryKey } from "#/features/media/api";
import { noteCategoriesQueryKey, notesQueryKey } from "#/features/notes/api";
import { CategoryPicker } from "#/features/notes/categories/CategoryPicker";
import { ApiError } from "#/lib/api";
import { AppLogo } from "./appTokens/AppLogo";
import { ExportSection } from "./ExportSection";
import {
	type BookmarkFormat,
	type DataImportReport,
	formatsWithNotes,
	type ImportErrorCode,
	type ImportFormat,
	type ImportPart,
	type ImportReport,
	importBookmarks,
	importData,
	importFormats,
	importParts,
} from "./import/api";

type ImportResult =
	| { kind: "bookmarks"; report: ImportReport }
	| { kind: "boardly"; report: DataImportReport };

const errorKeys = {
	invalid_import_file: "settings.import.errors.invalidFile",
	file_too_large: "settings.import.errors.tooLarge",
	invalid_folder: "settings.import.errors.invalidFolder",
	invalid_category: "settings.import.errors.invalidCategory",
} as const satisfies Record<ImportErrorCode, string>;

/** "Import / Export": the import form and the export of all data. */
export function ImportExportTab() {
	return (
		<div className="space-y-6">
			<ImportSection />
			<ExportSection />
		</div>
	);
}

/**
 * "Import": imports a Karakeep, Linkwarden or Netscape file, or chosen parts
 * of a Boardly export.
 */
function ImportSection() {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const [format, setFormat] = useState<ImportFormat>("karakeep");
	const [file, setFile] = useState<File | null>(null);
	const [folderId, setFolderId] = useState("");
	const [categoryId, setCategoryId] = useState<string | null>(null);
	const [skipDuplicates, setSkipDuplicates] = useState(true);
	const [parts, setParts] = useState<ReadonlySet<ImportPart>>(
		() => new Set(importParts),
	);
	const boardly = format === "boardly";
	const needsFolder = !boardly || parts.has("favourites");
	const withNotes = boardly
		? parts.has("notes")
		: formatsWithNotes.includes(format);

	const run = useMutation({
		mutationFn: async (): Promise<ImportResult> => {
			if (format === "boardly") {
				const chosen = importParts.filter((p) => parts.has(p));
				return {
					kind: "boardly",
					report: await importData(
						chosen,
						needsFolder ? folderId : null,
						withNotes ? categoryId : null,
						file as File,
						skipDuplicates,
					),
				};
			}
			return {
				kind: "bookmarks",
				report: await importBookmarks(
					format as BookmarkFormat,
					folderId,
					withNotes ? categoryId : null,
					file as File,
					skipDuplicates,
				),
			};
		},
		onSuccess: () => {
			for (const queryKey of [
				favouritesQueryKey,
				["tags"],
				tagUsageQueryKey,
				notesQueryKey,
				noteCategoriesQueryKey,
				dashboardsQueryKey,
				dashboardViewQueryKey,
				mediaQueryKey,
			]) {
				void queryClient.invalidateQueries({ queryKey });
			}
		},
	});

	function togglePart(part: ImportPart, on: boolean) {
		setParts((current) => {
			const next = new Set(current);
			if (on) next.add(part);
			else next.delete(part);
			return next;
		});
		run.reset();
	}

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		run.mutate();
	}

	const code = run.error instanceof ApiError ? run.error.code : null;
	let error = !run.error
		? null
		: code && code in errorKeys
			? t(errorKeys[code as ImportErrorCode], {
					format: t(`settings.import.formats.${format}`),
				})
			: t("common.genericError");
	if (error && boardly && code === "file_too_large") {
		error = `${error} ${t("settings.import.errors.tooLargeHint")}`;
	}

	return (
		<section className="rounded-xl border border-line bg-surface p-6">
			<h2 className="text-lg font-semibold">{t("settings.import.title")}</h2>
			<p className="mt-1 max-w-[60ch] text-sm text-muted">
				{t("settings.import.intro")}
			</p>
			<form className="mt-5 max-w-xl space-y-5" onSubmit={onSubmit}>
				<fieldset>
					<legend className="text-sm font-medium">
						{t("settings.import.format")}
					</legend>
					<div className="mt-2 flex flex-col gap-2">
						{importFormats.map((option) => (
							<label
								key={option}
								className="flex cursor-pointer items-center gap-3 rounded-lg border border-line px-3 py-2 text-sm has-[:checked]:border-ink"
							>
								<input
									type="radio"
									name="import-format"
									value={option}
									checked={format === option}
									onChange={() => {
										setFormat(option);
										run.reset();
									}}
									className="accent-ink"
								/>
								{option === "netscape" ? (
									<FileUp
										className="size-8 shrink-0 p-1 text-muted"
										aria-hidden
									/>
								) : option === "boardly" ? (
									<ArchiveRestore
										className="size-8 shrink-0 p-1 text-muted"
										aria-hidden
									/>
								) : (
									<AppLogo app={option} />
								)}
								{t(`settings.import.formats.${option}`)}
							</label>
						))}
					</div>
					<p className="mt-2 text-sm text-muted">
						{t("settings.import.karakeepHint")}
					</p>
				</fieldset>
				<div>
					<label htmlFor="import-file" className="block text-sm font-medium">
						{t("settings.import.file")}
					</label>
					<input
						id="import-file"
						type="file"
						accept=".json,.html,.htm,.zip,application/json,text/html,application/zip"
						onChange={(e) => {
							setFile(e.target.files?.[0] ?? null);
							run.reset();
						}}
						className={textInput}
					/>
				</div>
				{boardly && (
					<fieldset>
						<legend className="text-sm font-medium">
							{t("settings.import.parts.label")}
						</legend>
						<div className="mt-2 flex flex-col gap-2">
							{importParts.map((part) => (
								<div key={part} className="flex items-center gap-3">
									<input
										id={`import-part-${part}`}
										type="checkbox"
										checked={parts.has(part)}
										onChange={(e) => togglePart(part, e.target.checked)}
										className="size-4 accent-ink"
									/>
									<label htmlFor={`import-part-${part}`} className="text-sm">
										{t(`settings.import.parts.${part}`)}
									</label>
								</div>
							))}
						</div>
					</fieldset>
				)}
				{needsFolder && (
					<div>
						<FavouritePicker
							id="import-folder"
							label={t("settings.import.folder")}
							mode="folder"
							value={folderId}
							onChange={(id) => {
								setFolderId(id);
								run.reset();
							}}
						/>
						<p className="mt-1 text-sm text-muted">
							{t("settings.import.folderHint")}
						</p>
					</div>
				)}
				{withNotes && (
					<div>
						<CategoryPicker
							id="import-category"
							label={t("settings.import.category")}
							emptyLabel={t("settings.import.categoryTopLevel")}
							value={categoryId}
							onChange={(id) => {
								setCategoryId(id);
								run.reset();
							}}
						/>
						<p className="mt-1 text-sm text-muted">
							{t(
								boardly
									? "settings.import.categoryHintBoardly"
									: "settings.import.categoryHint",
							)}
						</p>
					</div>
				)}
				<div className="flex items-center gap-3">
					<input
						id="import-skip-duplicates"
						type="checkbox"
						checked={skipDuplicates}
						onChange={(e) => {
							setSkipDuplicates(e.target.checked);
							run.reset();
						}}
						className="size-4 accent-ink"
					/>
					<label
						htmlFor="import-skip-duplicates"
						className="text-sm font-medium"
					>
						{t("settings.import.skipDuplicates")}
					</label>
				</div>
				{error && (
					<p role="alert" className="text-sm text-danger">
						{error}
					</p>
				)}
				<button
					type="submit"
					className={primaryButton}
					disabled={
						run.isPending ||
						!file ||
						(needsFolder && folderId === "") ||
						(boardly && parts.size === 0)
					}
				>
					{run.isPending
						? t("settings.import.importing")
						: t("settings.import.submit")}
				</button>
			</form>
			{run.data?.kind === "bookmarks" && <Report report={run.data.report} />}
			{run.data?.kind === "boardly" && <DataReport report={run.data.report} />}
		</section>
	);
}

function Report({ report }: { report: ImportReport }) {
	const { t } = useTranslation();
	const skipped = [
		["duplicates", report.skipped.duplicates],
		["duplicateNotes", report.skipped.duplicateNotes],
		["invalidUrls", report.skipped.invalidUrls],
		["notLinks", report.skipped.notLinks],
		["smartLists", report.skipped.smartLists],
		["tooLargeNotes", report.skipped.tooLargeNotes],
		["shortenedDescriptions", report.shortenedDescriptions],
	] as const;
	return (
		<output className="mt-6 block rounded-lg border border-line bg-paper/40 px-4 py-3 text-sm">
			<p className="font-medium">
				{t("settings.import.report.imported", {
					links: t("settings.import.report.links", { count: report.links }),
					folders: t("settings.import.report.folders", {
						count: report.folders,
					}),
				})}
			</p>
			{(report.notes > 0 || report.categories > 0) && (
				<p className="mt-1 font-medium">
					{t("settings.import.report.importedNotes", {
						notes: t("settings.import.report.notes", { count: report.notes }),
						categories: t("settings.import.report.categories", {
							count: report.categories,
						}),
					})}
				</p>
			)}
			{skipped.some(([, n]) => n > 0) && (
				<ul className="mt-2 list-disc space-y-1 pl-5 text-muted">
					{skipped
						.filter(([, n]) => n > 0)
						.map(([key, n]) => (
							<li key={key}>
								{t(`settings.import.report.${key}`, { count: n })}
							</li>
						))}
				</ul>
			)}
		</output>
	);
}

type DataReportKey =
	| "links"
	| "folders"
	| "tags"
	| "duplicates"
	| "invalidUrls"
	| "dashboards"
	| "skippedWidgets"
	| "notes"
	| "categories"
	| "duplicateNotes"
	| "sources"
	| "duplicateSources"
	| "invalidSources"
	| "images"
	| "attachments"
	| "offlineCopies"
	| "skippedFiles";

function DataReport({ report }: { report: DataImportReport }) {
	const { t } = useTranslation();
	const lines: string[] = [];
	const notes: string[] = [];
	const add = (key: DataReportKey, count: number, list = lines) => {
		if (count > 0) list.push(t(`settings.import.dataReport.${key}`, { count }));
	};
	if (report.favourites) {
		add("links", report.favourites.links);
		add("folders", report.favourites.folders);
		add("tags", report.favourites.tags);
		add("duplicates", report.favourites.skipped.duplicates, notes);
		add("invalidUrls", report.favourites.skipped.invalidUrls, notes);
	}
	if (report.dashboards) {
		add("dashboards", report.dashboards.dashboards);
		add("skippedWidgets", report.dashboards.skippedWidgets, notes);
	}
	if (report.notes) {
		add("notes", report.notes.notes);
		add("categories", report.notes.categories);
		add("duplicateNotes", report.notes.duplicateNotes, notes);
	}
	if (report.media) {
		add("sources", report.media.sources);
		add("duplicateSources", report.media.duplicates, notes);
		add("invalidSources", report.media.invalid, notes);
	}
	add("images", report.files.images);
	add("attachments", report.files.attachments);
	add("offlineCopies", report.files.offlineCopies ?? 0);
	add("skippedFiles", report.files.skipped, notes);
	return (
		<output className="mt-6 block rounded-lg border border-line bg-paper/40 px-4 py-3 text-sm">
			<p className="font-medium">
				{lines.length > 0
					? t("settings.import.dataReport.imported", {
							items: lines.join(", "),
						})
					: t("settings.import.dataReport.nothing")}
			</p>
			{report.dashboards && report.dashboards.dashboards > 0 && (
				<p className="mt-1 text-muted">
					{t("settings.import.dataReport.dashboardsPrivate")}
				</p>
			)}
			{notes.length > 0 && (
				<ul className="mt-2 list-disc space-y-1 pl-5 text-muted">
					{notes.map((line) => (
						<li key={line}>{line}</li>
					))}
				</ul>
			)}
		</output>
	);
}
