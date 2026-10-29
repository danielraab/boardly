import { Archive, Pin } from "lucide-react";
import { useTranslation } from "react-i18next";

/**
 * The marks of a pinned and of an archived note: a pin icon, and an archive
 * icon with the label "Archived", or only the icon with the label as its
 * tooltip where space is short (compact). Renders nothing for a note that is
 * neither.
 */
export function NoteMarks({
	pinned,
	archived,
	compact = false,
	className = "",
}: {
	pinned: boolean;
	archived: boolean;
	compact?: boolean;
	className?: string;
}) {
	const { t } = useTranslation();
	if (!pinned && !archived) return null;
	return (
		<span
			className={`inline-flex shrink-0 items-center gap-1.5 text-muted ${className}`}
		>
			{pinned && (
				<Pin
					className="size-4"
					role="img"
					aria-label={t("notes.pinned")}
					aria-hidden={false}
				>
					<title>{t("notes.pinned")}</title>
				</Pin>
			)}
			{archived &&
				(compact ? (
					<Archive
						className="size-4"
						role="img"
						aria-label={t("notes.archived")}
						aria-hidden={false}
					>
						<title>{t("notes.archived")}</title>
					</Archive>
				) : (
					<span className="inline-flex items-center gap-1 text-xs font-medium">
						<Archive className="size-3.5" aria-hidden />
						{t("notes.archived")}
					</span>
				))}
		</span>
	);
}
