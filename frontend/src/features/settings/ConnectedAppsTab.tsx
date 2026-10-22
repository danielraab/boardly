import { AlertDialog } from "@base-ui/react/alert-dialog";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Ban, PlugZap, Trash2 } from "lucide-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import {
	dangerButton,
	iconButton,
	secondaryButton,
} from "#/components/buttonStyles";
import { dialogBackdrop, dialogPopup } from "#/components/dialogStyles";
import { favouritesQueryOptions } from "#/features/favourites/api";
import { pathsOf } from "#/features/favourites/tree";
import { AppLogo } from "./appTokens/AppLogo";
import {
	type AppToken,
	type AppTokenStatus,
	appNames,
	appTokensQueryKey,
	appTokensQueryOptions,
	deleteAppToken,
	revokeAppToken,
} from "./appTokens/api";
import { CreateAppTokenDialog } from "./CreateAppTokenDialog";

const badgeClass: Record<AppTokenStatus, string> = {
	active: "bg-accent/15 text-ink",
	expired: "bg-paper text-muted",
	revoked: "bg-danger/10 text-danger",
};

export function ConnectedAppsTab() {
	const { t } = useTranslation();
	const tokens = useQuery(appTokensQueryOptions);
	const favourites = useQuery(favouritesQueryOptions);
	const paths = favourites.data
		? pathsOf(favourites.data, t("favourites.title"))
		: null;

	return (
		<div>
			<div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
				<p className="max-w-[60ch] text-muted">{t("settings.apps.intro")}</p>
				<CreateAppTokenDialog />
			</div>
			<div className="mt-8">
				{tokens.isPending ? (
					<p className="text-muted">{t("common.loading")}</p>
				) : tokens.isError ? (
					<p role="alert" className="text-sm text-danger">
						{t("common.genericError")}
					</p>
				) : tokens.data.length === 0 ? (
					<div className="rounded-xl border border-dashed border-line bg-surface px-6 py-8 text-center">
						<PlugZap
							className="mx-auto size-8 text-muted"
							strokeWidth={1.75}
							aria-hidden
						/>
						<h2 className="mt-3 font-semibold">
							{t("settings.apps.emptyTitle")}
						</h2>
						<p className="mx-auto mt-1 max-w-[50ch] text-sm text-muted">
							{t("settings.apps.emptyText")}
						</p>
					</div>
				) : (
					<ul className="divide-y divide-line rounded-xl border border-line bg-surface">
						{tokens.data.map((token) => (
							<AppTokenItem
								key={token.id}
								token={token}
								rootPath={
									token.rootId === null
										? t("settings.apps.deletedFolder")
										: (paths?.get(token.rootId) ?? "…")
								}
							/>
						))}
					</ul>
				)}
			</div>
		</div>
	);
}

function AppTokenItem({
	token,
	rootPath,
}: {
	token: AppToken;
	rootPath: string;
}) {
	const { t, i18n } = useTranslation();
	const date = new Intl.DateTimeFormat(i18n.language, { dateStyle: "medium" });
	const format = (iso: string) => date.format(new Date(iso));
	const active = token.status === "active";

	const expiry = token.revokedAt
		? t("settings.apps.revokedAt", { date: format(token.revokedAt) })
		: token.expiresAt
			? t(active ? "settings.apps.expires" : "settings.apps.expired", {
					date: format(token.expiresAt),
				})
			: t("settings.apps.noExpiry");

	return (
		<li className="flex items-start gap-4 px-4 py-3">
			<AppLogo app={token.app} />
			<div className="min-w-0 flex-1">
				<p className="flex flex-wrap items-center gap-2">
					<span className="truncate font-medium">{token.name}</span>
					<span
						className={`rounded-full px-2 py-0.5 text-xs font-medium ${badgeClass[token.status]}`}
					>
						{t(`settings.apps.status.${token.status}`)}
					</span>
				</p>
				<p className="mt-0.5 truncate text-sm">
					{appNames[token.app]} · {rootPath}
				</p>
				<p className="mt-0.5 text-sm text-muted">
					<code className="font-mono">{token.prefix}…</code>
					{" · "}
					{t("settings.apps.created", { date: format(token.createdAt) })}
					{" · "}
					{token.lastUsedAt
						? t("settings.apps.lastUsed", { date: format(token.lastUsedAt) })
						: t("settings.apps.neverUsed")}
					{" · "}
					{expiry}
				</p>
			</div>
			{active ? <RevokeButton token={token} /> : <DeleteButton token={token} />}
		</li>
	);
}

function RevokeButton({ token }: { token: AppToken }) {
	const { t } = useTranslation();
	return (
		<ConfirmAction
			icon={<Ban className="size-4" aria-hidden />}
			label={t("settings.apps.revoke", { name: token.name })}
			title={t("settings.apps.revokeTitle")}
			text={t("settings.apps.revokeText", { name: token.name })}
			confirm={t("settings.apps.revokeConfirm")}
			action={() => revokeAppToken(token.id)}
		/>
	);
}

function DeleteButton({ token }: { token: AppToken }) {
	const { t } = useTranslation();
	return (
		<ConfirmAction
			icon={<Trash2 className="size-4" aria-hidden />}
			label={t("settings.apps.delete", { name: token.name })}
			title={t("settings.apps.deleteTitle")}
			text={t("settings.apps.deleteText", { name: token.name })}
			confirm={t("settings.apps.deleteConfirm")}
			action={() => deleteAppToken(token.id)}
		/>
	);
}

function ConfirmAction({
	icon,
	label,
	title,
	text,
	confirm,
	action,
}: {
	icon: ReactNode;
	label: string;
	title: string;
	text: string;
	confirm: string;
	action: () => Promise<unknown>;
}) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const run = useMutation({
		mutationFn: action,
		onSettled: () =>
			queryClient.invalidateQueries({ queryKey: appTokensQueryKey }),
	});

	return (
		<AlertDialog.Root onOpenChange={(open) => open && run.reset()}>
			<AlertDialog.Trigger
				className={`${iconButton} size-8 hover:text-danger`}
				aria-label={label}
			>
				{icon}
			</AlertDialog.Trigger>
			<AlertDialog.Portal>
				<AlertDialog.Backdrop className={dialogBackdrop} />
				<AlertDialog.Popup className={dialogPopup}>
					<AlertDialog.Title className="text-lg font-semibold">
						{title}
					</AlertDialog.Title>
					<AlertDialog.Description className="mt-1 text-sm text-muted">
						{text}
					</AlertDialog.Description>
					{run.isError && (
						<p role="alert" className="mt-3 text-sm text-danger">
							{t("common.genericError")}
						</p>
					)}
					<div className="mt-6 flex justify-end gap-3">
						<AlertDialog.Close className={secondaryButton}>
							{t("settings.apps.cancel")}
						</AlertDialog.Close>
						<button
							type="button"
							className={dangerButton}
							disabled={run.isPending}
							onClick={() => run.mutate()}
						>
							{confirm}
						</button>
					</div>
				</AlertDialog.Popup>
			</AlertDialog.Portal>
		</AlertDialog.Root>
	);
}
