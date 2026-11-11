import { Dialog } from "@base-ui/react/dialog";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Trash2, Users } from "lucide-react";
import { type FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import {
	iconButton,
	primaryButton,
	secondaryButton,
} from "#/components/buttonStyles";
import {
	dialogBackdrop,
	dialogPopup,
	textInput,
} from "#/components/dialogStyles";
import {
	dashboardsQueryKey,
	dashboardViewQueryKey,
	updateDashboard,
} from "#/features/dashboards/api";
import { Toggle } from "#/features/dashboards/DashboardDialog";
import { groupsQueryOptions } from "#/features/groups/api";
import { ApiError } from "#/lib/api";
import {
	inviteToDashboard,
	removeShare,
	revokeDashboardInvitation,
	type Share,
	sharesQueryKey,
	sharesQueryOptions,
	shareWithGroup,
} from "./api";

/** The dashboard the share dialog is for. */
export type ShareTarget = { id: string; title: string; isPublic: boolean };

export function ShareDialog({
	target,
	onClose,
}: {
	target: ShareTarget | null;
	onClose: () => void;
}) {
	return (
		<Dialog.Root
			open={target !== null}
			onOpenChange={(open) => !open && onClose()}
		>
			<Dialog.Portal>
				<Dialog.Backdrop className={dialogBackdrop} />
				<Dialog.Popup
					className={`${dialogPopup} max-h-[calc(100dvh-2rem)] overflow-y-auto`}
				>
					{target && <ShareForm key={target.id} target={target} />}
				</Dialog.Popup>
			</Dialog.Portal>
		</Dialog.Root>
	);
}

const errorTexts = {
	invalid_email: "sharing.errors.invalid_email",
	already_invited: "sharing.errors.already_invited",
	cannot_invite_self: "sharing.errors.cannot_invite_self",
	rate_limited: "sharing.errors.rate_limited",
	already_shared: "sharing.errors.already_shared",
	group_not_found: "sharing.errors.group_not_found",
} as const;

/** The translated message of a failed request. */
export function useShareError() {
	const { t } = useTranslation();
	return (error: unknown) => {
		const code = error instanceof ApiError ? error.code : "";
		return code in errorTexts
			? t(errorTexts[code as keyof typeof errorTexts])
			: t("common.genericError");
	};
}

function ShareForm({ target }: { target: ShareTarget }) {
	const { t, i18n } = useTranslation();
	const queryClient = useQueryClient();
	const errorText = useShareError();
	const list = useQuery(sharesQueryOptions(target.id));
	const groups = useQuery(groupsQueryOptions);
	const [isPublic, setIsPublic] = useState(target.isPublic);
	const [email, setEmail] = useState("");
	const [groupId, setGroupId] = useState("");

	const refresh = () =>
		queryClient.invalidateQueries({
			queryKey: [...sharesQueryKey, target.id],
		});

	const publish = useMutation({
		mutationFn: (value: boolean) =>
			updateDashboard(target.id, { isPublic: value }),
		onMutate: (value) => setIsPublic(value),
		onError: (_e, value) => setIsPublic(!value),
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: dashboardsQueryKey });
			void queryClient.invalidateQueries({ queryKey: dashboardViewQueryKey });
		},
	});
	const invite = useMutation({
		mutationFn: (address: string) =>
			inviteToDashboard(target.id, address, i18n.resolvedLanguage),
		onSuccess: () => {
			setEmail("");
			void refresh();
		},
	});
	const share = useMutation({
		mutationFn: (id: string) => shareWithGroup(target.id, id),
		onSuccess: () => {
			setGroupId("");
			void refresh();
		},
	});
	const remove = useMutation({
		mutationFn: (shareId: string) => removeShare(target.id, shareId),
		onSuccess: () => void refresh(),
	});
	const revoke = useMutation({
		mutationFn: (id: string) => revokeDashboardInvitation(target.id, id),
		onSuccess: () => void refresh(),
	});

	function onInvite(event: FormEvent) {
		event.preventDefault();
		share.reset();
		if (email.trim() !== "") invite.mutate(email.trim());
	}

	function onShare(event: FormEvent) {
		event.preventDefault();
		invite.reset();
		if (groupId !== "") share.mutate(groupId);
	}

	const shares = list.data?.shares ?? [];
	const invitations = list.data?.invitations ?? [];
	const sharedGroups = new Set(shares.map((s) => s.group?.id));
	const choices = (groups.data ?? []).filter((g) => !sharedGroups.has(g.id));
	const showGroupHint = groupId !== "" || shares.some((s) => s.group);
	const failure =
		publish.error ??
		invite.error ??
		share.error ??
		remove.error ??
		revoke.error;

	return (
		<>
			<Dialog.Title className="text-lg font-semibold">
				{t("sharing.dialog.title", { title: target.title })}
			</Dialog.Title>
			<div className="mt-5 space-y-5">
				<Toggle
					id="share-public"
					label={t("dashboards.dialog.publicLabel")}
					hint={t("dashboards.dialog.publicHint")}
					checked={isPublic}
					onChange={(value) => publish.mutate(value)}
				/>

				<form onSubmit={onInvite} noValidate>
					<label htmlFor="share-email" className="block text-sm font-medium">
						{t("sharing.dialog.emailLabel")}
					</label>
					<div className="flex items-start gap-2">
						<input
							id="share-email"
							type="email"
							autoComplete="off"
							aria-describedby="share-email-hint"
							value={email}
							onChange={(e) => setEmail(e.target.value)}
							className={textInput}
						/>
						<button
							type="submit"
							className={`${primaryButton} mt-2 py-2.5`}
							disabled={invite.isPending || email.trim() === ""}
						>
							{t("sharing.dialog.invite")}
						</button>
					</div>
					<p id="share-email-hint" className="mt-1 text-sm text-muted">
						{t("sharing.dialog.emailHint")}
					</p>
				</form>

				<form onSubmit={onShare}>
					<label htmlFor="share-group" className="block text-sm font-medium">
						{t("sharing.dialog.groupLabel")}
					</label>
					{groups.data?.length === 0 ? (
						<p className="mt-1 text-sm text-muted">
							{t("sharing.dialog.noGroups")}{" "}
							<Link to="/groups" className="underline underline-offset-4">
								{t("sharing.dialog.manageGroups")}
							</Link>
						</p>
					) : (
						<div className="flex items-start gap-2">
							<select
								id="share-group"
								value={groupId}
								onChange={(e) => setGroupId(e.target.value)}
								className={textInput}
							>
								<option value="">{t("sharing.dialog.chooseGroup")}</option>
								{choices.map((g) => (
									<option key={g.id} value={g.id}>
										{g.name}
									</option>
								))}
							</select>
							<button
								type="submit"
								className={`${secondaryButton} mt-2 py-2.5`}
								disabled={share.isPending || groupId === ""}
							>
								{t("sharing.dialog.shareWithGroup")}
							</button>
						</div>
					)}
				</form>

				{showGroupHint && (
					<p className="text-sm text-muted">{t("sharing.dialog.groupHint")}</p>
				)}

				{failure !== null && (
					<p role="alert" className="text-sm text-red-700">
						{errorText(failure)}
					</p>
				)}

				{list.isError ? (
					<p role="alert" className="text-sm text-red-700">
						{t("common.genericError")}
					</p>
				) : (
					(shares.length > 0 || invitations.length > 0) && (
						<div>
							<h3 className="text-sm font-medium">
								{t("sharing.dialog.accessList")}
							</h3>
							<ul className="mt-2 divide-y divide-line rounded-lg border border-line">
								{shares.map((s) => (
									<ShareRow
										key={s.id}
										share={s}
										busy={remove.isPending}
										onRemove={() => remove.mutate(s.id)}
									/>
								))}
								{invitations.map((inv) => (
									<li
										key={inv.id}
										className="flex items-center gap-2 px-3 py-2 text-sm"
									>
										<span className="min-w-0 flex-1 truncate">{inv.email}</span>
										<span className="text-muted">{t("sharing.invited")}</span>
										<button
											type="button"
											className={iconButton}
											aria-label={t("sharing.revoke", { email: inv.email })}
											disabled={revoke.isPending}
											onClick={() => revoke.mutate(inv.id)}
										>
											<Trash2 className="size-4" aria-hidden />
										</button>
									</li>
								))}
							</ul>
						</div>
					)
				)}

				<div className="flex justify-end">
					<Dialog.Close className={secondaryButton}>
						{t("common.close")}
					</Dialog.Close>
				</div>
			</div>
		</>
	);
}

function ShareRow({
	share,
	busy,
	onRemove,
}: {
	share: Share;
	busy: boolean;
	onRemove: () => void;
}) {
	const { t } = useTranslation();
	const name = share.user
		? (share.user.name ?? share.user.slug)
		: (share.group?.name ?? "");
	return (
		<li className="flex items-center gap-2 px-3 py-2 text-sm">
			{share.group && (
				<Users className="size-4 shrink-0 text-muted" aria-hidden />
			)}
			<span className="min-w-0 flex-1 truncate">
				{name}
				{share.user?.name && (
					<span className="text-muted"> /{share.user.slug}</span>
				)}
			</span>
			<span className="text-muted">{t("sharing.canView")}</span>
			<button
				type="button"
				className={iconButton}
				aria-label={t("sharing.remove", { name })}
				disabled={busy}
				onClick={onRemove}
			>
				<Trash2 className="size-4" aria-hidden />
			</button>
		</li>
	);
}
