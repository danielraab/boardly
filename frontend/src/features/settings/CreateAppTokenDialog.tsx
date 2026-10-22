import { Dialog } from "@base-ui/react/dialog";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Check, Copy, Plus, TriangleAlert } from "lucide-react";
import { type FormEvent, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import {
	primaryButton,
	secondaryButton,
	secondaryIconButton,
} from "#/components/buttonStyles";
import {
	dialogBackdrop,
	dialogPopup,
	textInput,
} from "#/components/dialogStyles";
import { FavouritePicker } from "#/features/favourites/FavouritePicker";
import { ApiError } from "#/lib/api";
import {
	type AppTokenApp,
	appTokensQueryKey,
	type CreatedAppToken,
	createAppToken,
	serverAddress,
} from "./appTokens/api";

const MAX_NAME = 100;

const COPIED_MS = 3000;

type Expiry = "30d" | "90d" | "1y" | "never" | "custom";

const expiryOptions: readonly Expiry[] = [
	"30d",
	"90d",
	"1y",
	"never",
	"custom",
];

/** A local date as YYYY-MM-DD, the value format of a date input. */
function isoDate(d: Date) {
	const pad = (n: number) => String(n).padStart(2, "0");
	return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

/**
 * The expiry time for the choice, or null for no expiry. A custom date expires
 * at the end of that day; undefined means the custom date is missing or past.
 */
function expiresAt(expiry: Expiry, customDate: string, now = new Date()) {
	const days = { "30d": 30, "90d": 90 } as const;
	switch (expiry) {
		case "never":
			return null;
		case "30d":
		case "90d":
			return new Date(now.getTime() + days[expiry] * 86_400_000);
		case "1y": {
			const d = new Date(now);
			d.setFullYear(d.getFullYear() + 1);
			return d;
		}
		case "custom": {
			if (!customDate) return undefined;
			const d = new Date(`${customDate}T23:59:59`);
			return Number.isNaN(d.getTime()) || d <= now ? undefined : d;
		}
	}
}

/**
 * "Connect app": asks for the token's name, app, root folder and expiry, then
 * shows the new secret once. Closing the dialog forgets the secret.
 */
export function CreateAppTokenDialog() {
	const { t } = useTranslation();
	const [open, setOpen] = useState(false);
	const [created, setCreated] = useState<CreatedAppToken | null>(null);

	// The secret is forgotten once the dialog has finished closing.
	return (
		<Dialog.Root
			open={open}
			onOpenChange={setOpen}
			onOpenChangeComplete={(isOpen) => !isOpen && setCreated(null)}
		>
			<Dialog.Trigger className={`${primaryButton} shrink-0`}>
				<Plus className="size-4" aria-hidden />
				{t("settings.apps.connect")}
			</Dialog.Trigger>
			<Dialog.Portal>
				<Dialog.Backdrop className={dialogBackdrop} />
				<Dialog.Popup
					className={`${dialogPopup} max-h-[calc(100dvh-2rem)] overflow-y-auto`}
				>
					{created ? (
						<SecretStep token={created} />
					) : (
						<FormStep onCreated={setCreated} />
					)}
				</Dialog.Popup>
			</Dialog.Portal>
		</Dialog.Root>
	);
}

function FormStep({
	onCreated,
}: {
	onCreated: (token: CreatedAppToken) => void;
}) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const [name, setName] = useState("");
	const [app, setApp] = useState<AppTokenApp>("karakeep");
	const [rootId, setRootId] = useState("");
	const [expiry, setExpiry] = useState<Expiry>("90d");
	const [customDate, setCustomDate] = useState("");

	const expires = expiresAt(expiry, customDate);
	const create = useMutation({
		mutationFn: () =>
			createAppToken({
				name,
				app,
				rootId,
				expiresAt: expires ? expires.toISOString() : null,
			}),
		onSuccess: (token) => {
			void queryClient.invalidateQueries({ queryKey: appTokensQueryKey });
			onCreated(token);
		},
	});

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		create.mutate();
	}

	const code = create.error instanceof ApiError ? create.error.code : null;
	const error = !create.error
		? null
		: code === "invalid_expiry"
			? t("settings.apps.form.expiryInvalid")
			: code === "invalid_root"
				? t("settings.apps.form.rootInvalid")
				: t("common.genericError");
	const customInvalid = expiry === "custom" && customDate !== "" && !expires;

	return (
		<>
			<Dialog.Title className="text-lg font-semibold">
				{t("settings.apps.form.title")}
			</Dialog.Title>
			<Dialog.Description className="mt-1 text-sm text-muted">
				{t("settings.apps.form.text")}
			</Dialog.Description>
			<form className="mt-5 space-y-5" onSubmit={onSubmit}>
				<div>
					<label htmlFor="app-token-name" className="block text-sm font-medium">
						{t("settings.apps.form.name")}
					</label>
					<input
						id="app-token-name"
						name="name"
						autoComplete="off"
						required
						maxLength={MAX_NAME}
						placeholder={t("settings.apps.form.namePlaceholder")}
						value={name}
						onChange={(e) => setName(e.target.value)}
						className={textInput}
					/>
				</div>
				<div>
					<label htmlFor="app-token-app" className="block text-sm font-medium">
						{t("settings.apps.form.app")}
					</label>
					<select
						id="app-token-app"
						value={app}
						onChange={(e) => setApp(e.target.value as AppTokenApp)}
						className={textInput}
					>
						<option value="karakeep">Karakeep</option>
					</select>
				</div>
				<div>
					<FavouritePicker
						id="app-token-root"
						label={t("settings.apps.form.root")}
						mode="folder"
						value={rootId}
						onChange={setRootId}
					/>
					<p className="mt-1 text-sm text-muted">
						{t("settings.apps.form.rootHint")}
					</p>
				</div>
				<fieldset>
					<legend className="text-sm font-medium">
						{t("settings.apps.form.expiry")}
					</legend>
					<div className="mt-2 flex flex-wrap gap-2">
						{expiryOptions.map((option) => (
							<label
								key={option}
								className="flex cursor-pointer items-center gap-2 rounded-lg border border-line px-3 py-1.5 text-sm has-[:checked]:border-ink"
							>
								<input
									type="radio"
									name="app-token-expiry"
									value={option}
									checked={expiry === option}
									onChange={() => setExpiry(option)}
									className="accent-ink"
								/>
								{t(`settings.apps.form.expiryOptions.${option}`)}
							</label>
						))}
					</div>
					{expiry === "custom" && (
						<div className="mt-3">
							<label
								htmlFor="app-token-expiry-date"
								className="block text-sm font-medium"
							>
								{t("settings.apps.form.expiryDate")}
							</label>
							<input
								id="app-token-expiry-date"
								type="date"
								required
								min={isoDate(new Date())}
								value={customDate}
								onChange={(e) => setCustomDate(e.target.value)}
								aria-invalid={customInvalid || undefined}
								className={textInput}
							/>
							{customInvalid && (
								<p role="alert" className="mt-2 text-sm text-danger">
									{t("settings.apps.form.expiryInvalid")}
								</p>
							)}
						</div>
					)}
				</fieldset>
				{error && (
					<p role="alert" className="text-sm text-danger">
						{error}
					</p>
				)}
				<div className="flex justify-end gap-3">
					<Dialog.Close className={secondaryButton}>
						{t("settings.apps.cancel")}
					</Dialog.Close>
					<button
						type="submit"
						className={primaryButton}
						disabled={
							create.isPending ||
							name.trim() === "" ||
							rootId === "" ||
							expires === undefined
						}
					>
						{create.isPending
							? t("settings.apps.form.creating")
							: t("settings.apps.form.create")}
					</button>
				</div>
			</form>
		</>
	);
}

function SecretStep({ token }: { token: CreatedAppToken }) {
	const { t } = useTranslation();
	return (
		<>
			<Dialog.Title className="text-lg font-semibold">
				{t("settings.apps.secret.title")}
			</Dialog.Title>
			<Dialog.Description className="mt-1 text-sm text-muted">
				{t("settings.apps.secret.text", { name: token.name })}
			</Dialog.Description>
			<div className="mt-5 space-y-4">
				<CopyField
					id="app-token-server"
					label={t("settings.apps.secret.server")}
					value={serverAddress(token.app)}
				/>
				<CopyField
					id="app-token-secret"
					label={t("settings.apps.secret.key")}
					value={token.secret}
				/>
			</div>
			<p className="mt-4 flex items-start gap-2 rounded-lg border border-danger/40 bg-danger/10 px-3 py-2 text-sm font-medium">
				<TriangleAlert
					className="mt-0.5 size-4 shrink-0 text-danger"
					aria-hidden
				/>
				{t("settings.apps.secret.once")}
			</p>
			<div className="mt-6 flex justify-end">
				<Dialog.Close className={primaryButton}>
					{t("settings.apps.secret.done")}
				</Dialog.Close>
			</div>
		</>
	);
}

function CopyField({
	id,
	label,
	value,
}: {
	id: string;
	label: string;
	value: string;
}) {
	const { t } = useTranslation();
	const [copied, setCopied] = useState(false);

	// The button shows the check mark for COPIED_MS after a copy.
	useEffect(() => {
		if (!copied) return;
		const id = setTimeout(() => setCopied(false), COPIED_MS);
		return () => clearTimeout(id);
	}, [copied]);

	async function copy() {
		try {
			await navigator.clipboard.writeText(value);
			setCopied(true);
		} catch {
			// The value stays visible and selectable; there is nothing else to do.
		}
	}

	return (
		<div>
			<label htmlFor={id} className="block text-sm font-medium">
				{label}
			</label>
			<div className="mt-2 flex gap-2">
				<input
					id={id}
					readOnly
					value={value}
					onFocus={(e) => e.currentTarget.select()}
					className="min-w-0 flex-1 rounded-lg border border-line bg-paper/40 px-4 py-2.5 font-mono text-sm text-ink outline-none focus-visible:border-ink"
				/>
				<button
					type="button"
					className={secondaryIconButton}
					onClick={copy}
					aria-label={
						copied
							? t("settings.apps.secret.copied")
							: t("settings.apps.secret.copy", { label })
					}
				>
					{copied ? (
						<Check className="size-4" aria-hidden />
					) : (
						<Copy className="size-4" aria-hidden />
					)}
				</button>
			</div>
		</div>
	);
}
