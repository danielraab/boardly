import { Check, LoaderCircle } from "lucide-react";
import { useTranslation } from "react-i18next";
import { secondaryButton } from "#/components/buttonStyles";
import type { SaveStatus } from "./useAutosave";

const statusAction = `${secondaryButton} px-3 py-1`;

/**
 * "Saving…", "Saved", or "Not saved" with Retry, for a note's autosave. With
 * compact, saving and saved are icons on phones, and the text stays for screen
 * readers. With onReload, a version conflict is shown too, with a Reload button.
 */
export function SaveIndicator({
	status,
	onRetry,
	onReload,
	compact = false,
}: {
	status: SaveStatus;
	onRetry: () => void;
	onReload?: () => void;
	compact?: boolean;
}) {
	const { t } = useTranslation();
	if (status === "idle") return null;
	if (status === "conflict") {
		if (!onReload) return null;
		return (
			<span
				role="status"
				className="flex items-center gap-2 text-sm text-danger"
			>
				{t("notes.editor.status.conflict")}
				<button type="button" className={statusAction} onClick={onReload}>
					{t("notes.editor.reload")}
				</button>
			</span>
		);
	}
	if (status === "failed") {
		return (
			<span
				role="status"
				className="flex items-center gap-2 text-sm text-danger"
			>
				{t("notes.editor.status.failed")}
				<button type="button" className={statusAction} onClick={onRetry}>
					{t("notes.editor.status.retry")}
				</button>
			</span>
		);
	}
	const text =
		status === "saving"
			? t("notes.editor.status.saving")
			: t("notes.editor.status.saved");
	const colour = status === "saved" ? "text-success" : "text-muted";
	if (!compact) {
		return (
			<span role="status" className={`text-sm ${colour}`}>
				{text}
			</span>
		);
	}
	const Icon = status === "saving" ? LoaderCircle : Check;
	return (
		<span role="status" className={`flex items-center text-sm ${colour}`}>
			<Icon
				className={`size-4 sm:hidden ${status === "saving" ? "motion-safe:animate-spin" : ""}`}
				aria-hidden
			/>
			<span className="sr-only sm:not-sr-only">{text}</span>
		</span>
	);
}
