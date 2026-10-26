import { useMutation, useQueryClient } from "@tanstack/react-query";
import { FileUp } from "lucide-react";
import { type FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import { primaryButton } from "#/components/buttonStyles";
import { textInput } from "#/components/dialogStyles";
import { favouritesQueryKey } from "#/features/favourites/api";
import { FavouritePicker } from "#/features/favourites/FavouritePicker";
import { ApiError } from "#/lib/api";
import { AppLogo } from "./appTokens/AppLogo";
import {
	type ImportErrorCode,
	type ImportFormat,
	type ImportReport,
	importBookmarks,
	importFormats,
} from "./import/api";

const errorKeys = {
	invalid_import_file: "settings.import.errors.invalidFile",
	file_too_large: "settings.import.errors.tooLarge",
	invalid_folder: "settings.import.errors.invalidFolder",
} as const satisfies Record<ImportErrorCode, string>;

/** "Import / Export": the import form. Export comes later. */
export function ImportExportTab() {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const [format, setFormat] = useState<ImportFormat>("karakeep");
	const [file, setFile] = useState<File | null>(null);
	const [folderId, setFolderId] = useState("");

	const run = useMutation({
		mutationFn: () => importBookmarks(format, folderId, file as File),
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: favouritesQueryKey });
			void queryClient.invalidateQueries({ queryKey: ["tags"] });
		},
	});

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		run.mutate();
	}

	const code = run.error instanceof ApiError ? run.error.code : null;
	const error = !run.error
		? null
		: code && code in errorKeys
			? t(errorKeys[code as ImportErrorCode], {
					format: t(`settings.import.formats.${format}`),
				})
			: t("common.genericError");

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
						accept=".json,.html,.htm,application/json,text/html"
						onChange={(e) => {
							setFile(e.target.files?.[0] ?? null);
							run.reset();
						}}
						className={textInput}
					/>
				</div>
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
				{error && (
					<p role="alert" className="text-sm text-danger">
						{error}
					</p>
				)}
				<button
					type="submit"
					className={primaryButton}
					disabled={run.isPending || !file || folderId === ""}
				>
					{run.isPending
						? t("settings.import.importing")
						: t("settings.import.submit")}
				</button>
			</form>
			{run.data && <Report report={run.data} />}
		</section>
	);
}

function Report({ report }: { report: ImportReport }) {
	const { t } = useTranslation();
	const skipped = [
		["duplicates", report.skipped.duplicates],
		["invalidUrls", report.skipped.invalidUrls],
		["notLinks", report.skipped.notLinks],
		["smartLists", report.skipped.smartLists],
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
