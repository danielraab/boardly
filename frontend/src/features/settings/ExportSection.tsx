import { useMutation } from "@tanstack/react-query";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { primaryButton } from "#/components/buttonStyles";
import { ApiError } from "#/lib/api";
import { exportData, saveFile } from "./export/api";

/**
 * "Export": downloads all data of the user as a JSON file, or with files as a
 * ZIP archive (openspec change export-all-data).
 */
export function ExportSection() {
	const { t, i18n } = useTranslation();
	const [withFiles, setWithFiles] = useState(false);

	const run = useMutation({
		mutationFn: () => exportData(withFiles),
		onSuccess: ({ blob, fileName }) => saveFile(blob, fileName),
	});

	let error: string | null = null;
	if (run.error instanceof ApiError && run.error.status === 429) {
		const next = new Date(Date.now() + (run.error.retryAfter ?? 0) * 1000);
		error = t("settings.export.rateLimited", {
			time: new Intl.DateTimeFormat(i18n.language, {
				timeStyle: "short",
			}).format(next),
		});
	} else if (run.error) {
		error = t("common.genericError");
	}

	return (
		<section className="rounded-xl border border-line bg-surface p-6">
			<h2 className="text-lg font-semibold">{t("settings.export.title")}</h2>
			<p className="mt-1 max-w-[60ch] text-sm text-muted">
				{t("settings.export.intro")}
			</p>
			<div className="mt-5 max-w-xl space-y-5">
				<div>
					<div className="flex items-center gap-3">
						<input
							id="export-with-files"
							type="checkbox"
							checked={withFiles}
							onChange={(e) => {
								setWithFiles(e.target.checked);
								run.reset();
							}}
							className="size-4 accent-ink"
						/>
						<label htmlFor="export-with-files" className="text-sm font-medium">
							{t("settings.export.withFiles")}
						</label>
					</div>
					<p className="mt-1 text-sm text-muted">
						{t("settings.export.withFilesHint")}
					</p>
				</div>
				{error && (
					<p role="alert" className="text-sm text-danger">
						{error}
					</p>
				)}
				<button
					type="button"
					className={primaryButton}
					disabled={run.isPending}
					onClick={() => run.mutate()}
				>
					{run.isPending
						? t("settings.export.exporting")
						: t("settings.export.submit")}
				</button>
			</div>
		</section>
	);
}
