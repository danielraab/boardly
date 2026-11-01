import { Link } from "@tanstack/react-router";
import { TriangleAlert, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { secondaryButton } from "#/components/buttonStyles";
import { useMe } from "#/features/auth/session";
import {
	formatDeletionDate,
	useCancelDeletion,
	useCancelledNotice,
} from "./accountDeletion";

/**
 * Shown under the header while the signed-in user's account is scheduled for
 * deletion, and, after cancelling, the notice that connected apps stay revoked.
 */
export function DeletionBanner() {
	const { t, i18n } = useTranslation();
	const { data: me } = useMe();
	const cancel = useCancelDeletion();
	const notice = useCancelledNotice();

	if (me?.deletionScheduledAt) {
		return (
			<div
				role="status"
				className="mb-6 flex flex-col gap-3 rounded-xl border border-danger/40 bg-danger/10 px-4 py-3 sm:flex-row sm:items-center sm:justify-between"
			>
				<p className="flex items-start gap-2 text-sm">
					<TriangleAlert
						className="mt-0.5 size-4 shrink-0 text-danger"
						aria-hidden
					/>
					<span>
						{t("deleteAccount.banner", {
							date: formatDeletionDate(me.deletionScheduledAt, i18n.language),
						})}
						{cancel.isError && (
							<span role="alert" className="block text-danger">
								{t("common.genericError")}
							</span>
						)}
					</span>
				</p>
				<button
					type="button"
					className={`${secondaryButton} shrink-0 bg-surface`}
					disabled={cancel.isPending}
					onClick={() => cancel.mutate()}
				>
					{t("deleteAccount.cancel")}
				</button>
			</div>
		);
	}

	if (me && notice.shown) {
		return (
			<div
				role="status"
				className="mb-6 flex items-start gap-3 rounded-xl border border-line bg-surface px-4 py-3"
			>
				<p className="min-w-0 flex-1 text-sm">
					{t("deleteAccount.cancelled")}{" "}
					<Link
						to="/settings"
						search={{ tab: "apps" }}
						className="font-medium underline underline-offset-2"
					>
						{t("deleteAccount.cancelledLink")}
					</Link>
				</p>
				<button
					type="button"
					className="inline-flex size-7 shrink-0 items-center justify-center rounded-md text-muted hover:bg-paper hover:text-ink"
					aria-label={t("common.close")}
					onClick={notice.close}
				>
					<X className="size-4" aria-hidden />
				</button>
			</div>
		);
	}

	return null;
}
