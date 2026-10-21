import { AlertDialog } from "@base-ui/react/alert-dialog";
import { Dialog } from "@base-ui/react/dialog";
import { WebAuthnError } from "@simplewebauthn/browser";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { KeyRound, Plus, Trash2 } from "lucide-react";
import { type FormEvent, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import {
	dangerButton,
	iconButton,
	primaryButton,
	secondaryButton,
} from "#/components/buttonStyles";
import {
	dialogBackdrop as backdrop,
	dialogPopup as popup,
	textInput,
} from "#/components/dialogStyles";
import {
	deletePasskey,
	isCancelled,
	type Passkey,
	passkeysQueryKey,
	passkeysQueryOptions,
	registerPasskey,
	supportsPasskeys,
} from "#/features/auth/passkey";
import { meQueryKey, type User } from "#/features/auth/session";
import { ApiError, api } from "#/lib/api";
import { useThemeMode } from "#/theme";

const MAX_LABEL = 64;

/** Seconds until the RFC 3339 time `until`; ticks every second while above zero. */
function useRemainingSeconds(until: string | null) {
	const end = until ? Date.parse(until) : 0;
	const [remaining, setRemaining] = useState(() => secondsUntil(end));

	useEffect(() => {
		setRemaining(secondsUntil(end));
		if (secondsUntil(end) <= 0) return;
		const id = setInterval(() => {
			const left = secondsUntil(end);
			setRemaining(left);
			if (left <= 0) clearInterval(id);
		}, 1000);
		return () => clearInterval(id);
	}, [end]);

	return remaining;
}

function secondsUntil(end: number) {
	return Math.max(0, Math.ceil((end - Date.now()) / 1000));
}

function formatCountdown(seconds: number) {
	const m = Math.floor(seconds / 60);
	const s = seconds % 60;
	return `${m}:${String(s).padStart(2, "0")}`;
}

export function PasskeysTab({ user }: { user: User }) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const passkeys = useQuery(passkeysQueryOptions);
	const remaining = useRemainingSeconds(user.passkeyEnrollmentUntil);
	const windowOpen = remaining > 0;

	// The server ends the window too; refresh the session data when it closes.
	useEffect(() => {
		if (user.passkeyEnrollmentUntil && !windowOpen) {
			void queryClient.invalidateQueries({ queryKey: meQueryKey });
		}
	}, [user.passkeyEnrollmentUntil, windowOpen, queryClient]);

	return (
		<div>
			<p className="text-muted">{t("passkeys.intro")}</p>
			{windowOpen ? (
				<EnrollmentOpen remaining={remaining} />
			) : (
				<EnrollmentClosed user={user} />
			)}
			<div className="mt-8">
				{passkeys.isPending ? (
					<p className="text-muted">{t("common.loading")}</p>
				) : passkeys.isError ? (
					<p role="alert" className="text-sm text-danger">
						{t("common.genericError")}
					</p>
				) : passkeys.data.length === 0 ? (
					<div className="rounded-xl border border-dashed border-line bg-surface px-6 py-8 text-center">
						<KeyRound
							className="mx-auto size-8 text-muted"
							strokeWidth={1.75}
							aria-hidden
						/>
						<h2 className="mt-3 font-semibold">{t("passkeys.emptyTitle")}</h2>
						<p className="mx-auto mt-1 max-w-[45ch] text-sm text-muted">
							{t("passkeys.emptyText")}
						</p>
					</div>
				) : (
					<ul className="divide-y divide-line rounded-xl border border-line bg-surface">
						{passkeys.data.map((p) => (
							<PasskeyItem key={p.id} passkey={p} />
						))}
					</ul>
				)}
			</div>
		</div>
	);
}

function EnrollmentOpen({ remaining }: { remaining: number }) {
	const { t } = useTranslation();
	const [supported] = useState(supportsPasskeys);
	return (
		<div className="mt-4 flex flex-col gap-3 rounded-xl border border-line bg-surface p-4 sm:flex-row sm:items-center sm:justify-between">
			<p className="text-sm">
				{supported ? (
					<>
						{t("passkeys.windowOpen")}{" "}
						<span
							className="font-semibold tabular-nums"
							role="timer"
							aria-label={t("passkeys.remaining")}
						>
							{formatCountdown(remaining)}
						</span>
					</>
				) : (
					t("passkeys.unsupported")
				)}
			</p>
			<AddPasskeyDialog disabled={!supported} />
		</div>
	);
}

function EnrollmentClosed({ user }: { user: User }) {
	const { t, i18n } = useTranslation();
	const send = useMutation({
		mutationFn: () =>
			api("/api/auth/magic-link", {
				method: "POST",
				json: { email: user.email, locale: i18n.resolvedLanguage },
			}),
	});

	return (
		<div className="mt-4 rounded-xl border border-line bg-surface p-4">
			<p id="passkeys-window-closed" className="text-sm">
				{t("passkeys.windowClosed")}
			</p>
			<div className="mt-3 flex flex-wrap items-center gap-3">
				<button
					type="button"
					className={secondaryButton}
					disabled
					aria-describedby="passkeys-window-closed"
				>
					<Plus className="size-4" aria-hidden />
					{t("passkeys.add")}
				</button>
				<button
					type="button"
					className={primaryButton}
					disabled={send.isPending || send.isSuccess}
					onClick={() => send.mutate()}
				>
					{send.isPending ? t("login.sending") : t("passkeys.sendLink")}
				</button>
			</div>
			{send.isSuccess && (
				<output className="mt-3 block text-sm text-muted">
					{t("passkeys.linkSent", { email: user.email })}
				</output>
			)}
			{send.isError && (
				<p role="alert" className="mt-3 text-sm text-danger">
					{send.error instanceof ApiError && send.error.code === "rate_limited"
						? t("passkeys.linkRateLimited")
						: t("common.genericError")}
				</p>
			)}
		</div>
	);
}

function AddPasskeyDialog({ disabled }: { disabled: boolean }) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const [open, setOpen] = useState(false);
	const [label, setLabel] = useState("");

	const add = useMutation({
		mutationFn: () => registerPasskey(label),
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: passkeysQueryKey });
			setOpen(false);
		},
		onError: (e) => {
			if (e instanceof ApiError && e.code === "enrollment_window_closed") {
				void queryClient.invalidateQueries({ queryKey: meQueryKey });
			}
		},
	});

	function onOpenChange(next: boolean) {
		setOpen(next);
		if (next) {
			setLabel("");
			add.reset();
		}
	}

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		add.mutate();
	}

	const error = add.error;
	const message =
		!error || isCancelled(error)
			? null
			: error instanceof WebAuthnError &&
					error.code === "ERROR_AUTHENTICATOR_PREVIOUSLY_REGISTERED"
				? t("passkeys.alreadyRegistered")
				: error instanceof ApiError && error.code === "enrollment_window_closed"
					? t("passkeys.windowClosed")
					: error instanceof ApiError && error.code === "invalid_label"
						? t("passkeys.labelTooLong")
						: t("common.genericError");

	return (
		<Dialog.Root open={open} onOpenChange={onOpenChange}>
			<Dialog.Trigger
				className={`${primaryButton} shrink-0`}
				disabled={disabled}
			>
				<Plus className="size-4" aria-hidden />
				{t("passkeys.add")}
			</Dialog.Trigger>
			<Dialog.Portal>
				<Dialog.Backdrop className={backdrop} />
				<Dialog.Popup className={popup}>
					<Dialog.Title className="text-lg font-semibold">
						{t("passkeys.addTitle")}
					</Dialog.Title>
					<Dialog.Description className="mt-1 text-sm text-muted">
						{t("passkeys.addText")}
					</Dialog.Description>
					<form className="mt-5" onSubmit={onSubmit}>
						<label
							htmlFor="passkey-label"
							className="block text-sm font-medium"
						>
							{t("passkeys.labelLabel")}
						</label>
						<input
							id="passkey-label"
							name="label"
							autoComplete="off"
							maxLength={MAX_LABEL}
							placeholder={t("passkeys.labelPlaceholder")}
							value={label}
							onChange={(e) => setLabel(e.target.value)}
							className={textInput}
						/>
						{message && (
							<p role="alert" className="mt-3 text-sm text-danger">
								{message}
							</p>
						)}
						<div className="mt-6 flex justify-end gap-3">
							<Dialog.Close className={secondaryButton}>
								{t("passkeys.cancel")}
							</Dialog.Close>
							<button
								type="submit"
								className={primaryButton}
								disabled={add.isPending}
							>
								{add.isPending ? t("passkeys.adding") : t("passkeys.continue")}
							</button>
						</div>
					</form>
				</Dialog.Popup>
			</Dialog.Portal>
		</Dialog.Root>
	);
}

function ProviderIcon({ passkey }: { passkey: Passkey }) {
	const mode = useThemeMode();
	const { iconLight, iconDark } = passkey.provider ?? {};
	const src =
		mode === "dark" ? (iconDark ?? iconLight) : (iconLight ?? iconDark);
	if (!src) {
		return (
			<KeyRound
				className="size-6 shrink-0 text-muted"
				strokeWidth={1.75}
				aria-hidden
			/>
		);
	}
	// An <img> never runs scripts inside an SVG data URI.
	return <img src={src} alt="" className="size-6 shrink-0" />;
}

function PasskeyItem({ passkey }: { passkey: Passkey }) {
	const { t, i18n } = useTranslation();
	const queryClient = useQueryClient();
	const name =
		passkey.label ?? passkey.provider?.name ?? t("passkeys.fallbackName");
	const date = new Intl.DateTimeFormat(i18n.language, { dateStyle: "medium" });

	const remove = useMutation({
		mutationFn: () => deletePasskey(passkey.id),
		onSuccess: () =>
			queryClient.invalidateQueries({ queryKey: passkeysQueryKey }),
	});

	return (
		<li className="flex items-center gap-4 px-4 py-3">
			<ProviderIcon passkey={passkey} />
			<div className="min-w-0 flex-1">
				<p className="truncate font-medium">{name}</p>
				<p className="text-sm text-muted">
					{t("passkeys.created", {
						date: date.format(new Date(passkey.createdAt)),
					})}
					{" · "}
					{passkey.lastUsedAt
						? t("passkeys.lastUsed", {
								date: date.format(new Date(passkey.lastUsedAt)),
							})
						: t("passkeys.neverUsed")}
				</p>
			</div>
			<AlertDialog.Root onOpenChange={(open) => open && remove.reset()}>
				<AlertDialog.Trigger
					className={`${iconButton} size-8 hover:text-danger`}
					aria-label={t("passkeys.delete", { name })}
				>
					<Trash2 className="size-4" aria-hidden />
				</AlertDialog.Trigger>
				<AlertDialog.Portal>
					<AlertDialog.Backdrop className={backdrop} />
					<AlertDialog.Popup className={popup}>
						<AlertDialog.Title className="text-lg font-semibold">
							{t("passkeys.deleteTitle")}
						</AlertDialog.Title>
						<AlertDialog.Description className="mt-1 text-sm text-muted">
							{t("passkeys.deleteText", { name })}
						</AlertDialog.Description>
						{remove.isError && (
							<p role="alert" className="mt-3 text-sm text-danger">
								{t("common.genericError")}
							</p>
						)}
						<div className="mt-6 flex justify-end gap-3">
							<AlertDialog.Close className={secondaryButton}>
								{t("passkeys.cancel")}
							</AlertDialog.Close>
							<button
								type="button"
								className={dangerButton}
								disabled={remove.isPending}
								onClick={() => remove.mutate()}
							>
								{t("passkeys.deleteConfirm")}
							</button>
						</div>
					</AlertDialog.Popup>
				</AlertDialog.Portal>
			</AlertDialog.Root>
		</li>
	);
}
