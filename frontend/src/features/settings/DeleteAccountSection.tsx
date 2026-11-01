import { Dialog } from "@base-ui/react/dialog";
import { Link } from "@tanstack/react-router";
import { type FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import { dangerButton, secondaryButton } from "#/components/buttonStyles";
import {
	dialogBackdrop as backdrop,
	dialogPopup as popup,
	textInput,
} from "#/components/dialogStyles";
import type { User } from "#/features/auth/session";
import { ApiError } from "#/lib/api";
import { formatCountdown, useRemainingSeconds } from "#/lib/countdown";
import {
	formatDeletionDate,
	useCancelDeletion,
	useScheduleDeletion,
} from "./accountDeletion";

const linkClass = "font-medium text-ink underline underline-offset-2";

/** The last section of the Profile tab: delete the account or cancel its deletion. */
export function DeleteAccountSection({ user }: { user: User }) {
	const { t } = useTranslation();
	const remaining = useRemainingSeconds(user.deletionWindowUntil);

	return (
		<section
			aria-labelledby="delete-account-title"
			className="mt-8 rounded-xl border border-danger/40 bg-surface p-6"
		>
			<h2 id="delete-account-title" className="text-lg font-semibold">
				{t("deleteAccount.title")}
			</h2>
			{user.deletionScheduledAt ? (
				<Scheduled at={user.deletionScheduledAt} />
			) : (
				<>
					<p className="mt-2 text-sm text-muted">{t("deleteAccount.text")}</p>
					<ExportHint />
					{remaining > 0 ? (
						<div className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
							<p className="text-sm">
								{t("deleteAccount.windowOpen")}{" "}
								<span
									className="font-semibold tabular-nums"
									role="timer"
									aria-label={t("deleteAccount.remaining")}
								>
									{formatCountdown(remaining)}
								</span>
							</p>
							<DeleteAccountDialog user={user} />
						</div>
					) : (
						<p className="mt-4 text-sm">{t("deleteAccount.windowClosed")}</p>
					)}
				</>
			)}
		</section>
	);
}

function ExportHint() {
	const { t } = useTranslation();
	return (
		<p className="mt-2 text-sm text-muted">
			{t("deleteAccount.exportHint")}{" "}
			<Link to="/settings" search={{ tab: "data" }} className={linkClass}>
				{t("deleteAccount.exportLink")}
			</Link>
		</p>
	);
}

function Scheduled({ at }: { at: string }) {
	const { t, i18n } = useTranslation();
	const cancel = useCancelDeletion();
	return (
		<>
			<p className="mt-2 text-sm">
				{t("deleteAccount.scheduled", {
					date: formatDeletionDate(at, i18n.language),
				})}
			</p>
			<button
				type="button"
				className={`${secondaryButton} mt-4`}
				disabled={cancel.isPending}
				onClick={() => cancel.mutate()}
			>
				{t("deleteAccount.cancel")}
			</button>
			{cancel.isError && (
				<p role="alert" className="mt-2 text-sm text-danger">
					{t("common.genericError")}
				</p>
			)}
		</>
	);
}

function DeleteAccountDialog({ user }: { user: User }) {
	const { t, i18n } = useTranslation();
	const graceDays = user.accountDeletionDays;
	const [open, setOpen] = useState(false);
	const [email, setEmail] = useState("");
	const schedule = useScheduleDeletion();

	const matches =
		email.trim().toLowerCase() === user.email.trim().toLowerCase();
	const deletionDate = formatDeletionDate(
		new Date(Date.now() + graceDays * 24 * 60 * 60 * 1000).toISOString(),
		i18n.language,
	);

	function onOpenChange(next: boolean) {
		setOpen(next);
		if (next) {
			setEmail("");
			schedule.reset();
		}
	}

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		if (!matches) return;
		schedule.mutate(email, { onSuccess: () => setOpen(false) });
	}

	const code = schedule.error instanceof ApiError ? schedule.error.code : null;
	const error =
		code === "deletion_window_closed"
			? t("deleteAccount.windowClosed")
			: code === "email_mismatch"
				? t("deleteAccount.emailMismatch")
				: schedule.error
					? t("common.genericError")
					: null;

	return (
		<Dialog.Root open={open} onOpenChange={onOpenChange}>
			<Dialog.Trigger className={`${dangerButton} shrink-0`}>
				{t("deleteAccount.open")}
			</Dialog.Trigger>
			<Dialog.Portal>
				<Dialog.Backdrop className={backdrop} />
				<Dialog.Popup
					className={`${popup} max-h-[calc(100vh-2rem)] overflow-y-auto`}
				>
					<Dialog.Title className="text-lg font-semibold">
						{t("deleteAccount.dialogTitle")}
					</Dialog.Title>
					<Dialog.Description className="mt-1 text-sm text-muted">
						{graceDays > 0
							? t("deleteAccount.dialogDate", {
									date: deletionDate,
									count: graceDays,
								})
							: t("deleteAccount.dialogNow")}
					</Dialog.Description>
					<ul className="mt-4 list-disc space-y-1.5 pl-5 text-sm">
						<li>{t("deleteAccount.whatIsDeleted")}</li>
						<li>{t("deleteAccount.otherDevices")}</li>
						<li>{t("deleteAccount.apps")}</li>
						<li>{t("deleteAccount.publicDashboards")}</li>
						<li>
							{t("deleteAccount.slugFree", { address: `/${user.slug}/…` })}
						</li>
					</ul>
					<ExportHint />
					<form className="mt-5" onSubmit={onSubmit}>
						<label
							htmlFor="delete-account-email"
							className="block text-sm font-medium"
						>
							{t("deleteAccount.emailLabel", { email: user.email })}
						</label>
						<input
							id="delete-account-email"
							type="email"
							autoComplete="off"
							value={email}
							onChange={(e) => {
								setEmail(e.target.value);
								schedule.reset();
							}}
							className={textInput}
						/>
						{error && (
							<p role="alert" className="mt-3 text-sm text-danger">
								{error}
							</p>
						)}
						<div className="mt-6 flex justify-end gap-3">
							<Dialog.Close className={secondaryButton}>
								{t("deleteAccount.keep")}
							</Dialog.Close>
							<button
								type="submit"
								className={dangerButton}
								disabled={!matches || schedule.isPending}
							>
								{t("deleteAccount.confirm")}
							</button>
						</div>
					</form>
				</Dialog.Popup>
			</Dialog.Portal>
		</Dialog.Root>
	);
}
