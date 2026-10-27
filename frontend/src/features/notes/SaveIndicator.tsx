import { useTranslation } from "react-i18next";
import { secondaryButton } from "#/components/buttonStyles";
import type { SaveStatus } from "./useAutosave";

/** "Saving…", "Saved", or "Not saved" with Retry, for a note's autosave. */
export function SaveIndicator({
	status,
	onRetry,
}: {
	status: SaveStatus;
	onRetry: () => void;
}) {
	const { t } = useTranslation();
	if (status === "idle" || status === "conflict") return null;
	if (status === "failed") {
		return (
			<span
				role="status"
				className="flex items-center gap-2 text-sm text-danger"
			>
				{t("notes.editor.status.failed")}
				<button
					type="button"
					className={`${secondaryButton} px-3 py-1`}
					onClick={onRetry}
				>
					{t("notes.editor.status.retry")}
				</button>
			</span>
		);
	}
	return (
		<span role="status" className="text-sm text-muted">
			{status === "saving"
				? t("notes.editor.status.saving")
				: t("notes.editor.status.saved")}
		</span>
	);
}
