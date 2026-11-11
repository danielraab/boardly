import { AlertDialog } from "@base-ui/react/alert-dialog";
import { Dialog } from "@base-ui/react/dialog";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, Navigate, useNavigate } from "@tanstack/react-router";
import { ArrowLeft, Plus, Trash2, Users } from "lucide-react";
import { type FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import {
	dangerButton,
	iconButton,
	primaryButton,
	secondaryButton,
} from "#/components/buttonStyles";
import {
	dialogBackdrop,
	dialogPopup,
	textInput,
} from "#/components/dialogStyles";
import { widePage } from "#/components/pageWidths";
import { useMe } from "#/features/auth/session";
import { useShareError } from "#/features/sharing/ShareDialog";
import { ApiError } from "#/lib/api";
import { usePageTitle } from "#/lib/usePageTitle";
import {
	createGroup,
	deleteGroup,
	type GroupDetails,
	type GroupMember,
	type GroupRole,
	groupQueryOptions,
	groupsQueryKey,
	groupsQueryOptions,
	inviteToGroup,
	MAX_GROUP_NAME,
	removeMember,
	renameGroup,
	revokeGroupInvitation,
	setMemberRole,
} from "./api";

/** The translated message of a failed group request. */
function useGroupError() {
	const { t } = useTranslation();
	return (error: unknown) => {
		const code = error instanceof ApiError ? error.code : "";
		if (code === "last_admin") return t("groups.errors.lastAdmin");
		if (code === "invalid_name") return t("groups.errors.invalidName");
		return t("common.genericError");
	};
}

export function GroupsPage({ groupId }: { groupId?: string }) {
	const { t } = useTranslation();
	usePageTitle(t("groups.title"));
	const { data: me } = useMe();

	if (me === null) return <Navigate to="/login" />;
	if (!me) return null;

	return (
		<section className={`${widePage} pt-12`}>
			{groupId ? (
				<GroupView key={groupId} id={groupId} meId={me.id} />
			) : (
				<GroupList />
			)}
		</section>
	);
}

function GroupList() {
	const { t } = useTranslation();
	const list = useQuery(groupsQueryOptions);
	const [creating, setCreating] = useState(false);

	const createButton = (
		<button
			type="button"
			className={secondaryButton}
			onClick={() => setCreating(true)}
		>
			<Plus className="size-4" aria-hidden />
			{t("groups.create")}
		</button>
	);

	return (
		<>
			<div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
				<div>
					<h1 className="text-3xl font-semibold tracking-tight">
						{t("groups.title")}
					</h1>
					<p className="mt-2 max-w-[60ch] text-muted">{t("groups.intro")}</p>
				</div>
				{list.data && list.data.length > 0 && (
					<div className="shrink-0">{createButton}</div>
				)}
			</div>
			<div className="mt-8">
				{list.isPending ? (
					<p className="text-muted">{t("common.loading")}</p>
				) : list.isError ? (
					<p role="alert" className="text-sm text-red-700">
						{t("common.genericError")}
					</p>
				) : list.data.length === 0 ? (
					<div className="rounded-xl border border-dashed border-line bg-surface px-6 py-12 text-center">
						<Users
							className="mx-auto size-8 text-muted"
							strokeWidth={1.75}
							aria-hidden
						/>
						<h2 className="mt-4 text-lg font-semibold">
							{t("groups.empty.title")}
						</h2>
						<p className="mx-auto mt-1 max-w-[45ch] text-muted">
							{t("groups.empty.text")}
						</p>
						<div className="mt-6">{createButton}</div>
					</div>
				) : (
					<ul className="divide-y divide-line rounded-xl border border-line bg-surface">
						{list.data.map((g) => (
							<li key={g.id} className="px-4 py-3">
								<Link
									to="/groups"
									search={{ group: g.id }}
									className="block truncate font-medium underline-offset-4 hover:underline"
								>
									{g.name}
								</Link>
								<p className="text-sm text-muted">
									{t("groups.memberCount", { count: g.memberCount })}
									{" · "}
									{t(`groups.role.${g.role}`)}
								</p>
							</li>
						))}
					</ul>
				)}
			</div>
			<NameDialog
				open={creating}
				title={t("groups.createDialog.title")}
				initial=""
				onSubmit={createGroup}
				onClose={() => setCreating(false)}
			/>
		</>
	);
}

function NameDialog({
	open,
	title,
	initial,
	onSubmit,
	onClose,
}: {
	open: boolean;
	title: string;
	initial: string;
	onSubmit: (name: string) => Promise<unknown>;
	onClose: () => void;
}) {
	return (
		<Dialog.Root open={open} onOpenChange={(o) => !o && onClose()}>
			<Dialog.Portal>
				<Dialog.Backdrop className={dialogBackdrop} />
				<Dialog.Popup className={dialogPopup}>
					{open && (
						<NameForm
							title={title}
							initial={initial}
							onSubmit={onSubmit}
							onClose={onClose}
						/>
					)}
				</Dialog.Popup>
			</Dialog.Portal>
		</Dialog.Root>
	);
}

function NameForm({
	title,
	initial,
	onSubmit,
	onClose,
}: {
	title: string;
	initial: string;
	onSubmit: (name: string) => Promise<unknown>;
	onClose: () => void;
}) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const errorText = useGroupError();
	const [name, setName] = useState(initial);
	const save = useMutation({
		mutationFn: () => onSubmit(name),
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: groupsQueryKey });
			onClose();
		},
	});

	function submit(event: FormEvent) {
		event.preventDefault();
		save.mutate();
	}

	return (
		<>
			<Dialog.Title className="text-lg font-semibold">{title}</Dialog.Title>
			<form className="mt-5 space-y-4" onSubmit={submit} noValidate>
				<div>
					<label htmlFor="group-name" className="block text-sm font-medium">
						{t("groups.nameLabel")}
					</label>
					<input
						id="group-name"
						autoComplete="off"
						maxLength={MAX_GROUP_NAME}
						value={name}
						onChange={(e) => setName(e.target.value)}
						className={textInput}
					/>
				</div>
				{save.isError && (
					<p role="alert" className="text-sm text-red-700">
						{errorText(save.error)}
					</p>
				)}
				<div className="flex justify-end gap-3 pt-2">
					<Dialog.Close className={secondaryButton}>
						{t("groups.cancel")}
					</Dialog.Close>
					<button
						type="submit"
						className={primaryButton}
						disabled={save.isPending || name.trim() === ""}
					>
						{t("groups.save")}
					</button>
				</div>
			</form>
		</>
	);
}

/** An action that asks first and shows its error in the same dialog. */
type Confirmation = {
	title: string;
	text: string;
	confirm: string;
	run: () => Promise<unknown>;
	after?: () => void;
};

function ConfirmDialog({
	action,
	onClose,
}: {
	action: Confirmation | null;
	onClose: () => void;
}) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const errorText = useGroupError();
	const run = useMutation({
		mutationFn: () => {
			if (!action) throw new Error("nothing to confirm");
			return action.run();
		},
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: groupsQueryKey });
			action?.after?.();
			onClose();
		},
	});

	return (
		<AlertDialog.Root
			open={action !== null}
			onOpenChange={(open) => {
				if (!open) {
					run.reset();
					onClose();
				}
			}}
		>
			<AlertDialog.Portal>
				<AlertDialog.Backdrop className={dialogBackdrop} />
				<AlertDialog.Popup className={dialogPopup}>
					{action && (
						<>
							<AlertDialog.Title className="text-lg font-semibold">
								{action.title}
							</AlertDialog.Title>
							<AlertDialog.Description className="mt-1 text-sm text-muted">
								{action.text}
							</AlertDialog.Description>
						</>
					)}
					{run.isError && (
						<p role="alert" className="mt-3 text-sm text-red-700">
							{errorText(run.error)}
						</p>
					)}
					<div className="mt-6 flex justify-end gap-3">
						<AlertDialog.Close className={secondaryButton}>
							{t("groups.cancel")}
						</AlertDialog.Close>
						<button
							type="button"
							className={dangerButton}
							disabled={run.isPending}
							onClick={() => run.mutate()}
						>
							{action?.confirm}
						</button>
					</div>
				</AlertDialog.Popup>
			</AlertDialog.Portal>
		</AlertDialog.Root>
	);
}

function GroupView({ id, meId }: { id: string; meId: string }) {
	const { t } = useTranslation();
	const navigate = useNavigate();
	const group = useQuery(groupQueryOptions(id));
	const [renaming, setRenaming] = useState(false);
	const [confirming, setConfirming] = useState<Confirmation | null>(null);

	const backLink = (
		<Link
			to="/groups"
			className="inline-flex items-center gap-1 text-sm text-muted underline-offset-4 hover:underline"
		>
			<ArrowLeft className="size-4" aria-hidden />
			{t("groups.back")}
		</Link>
	);

	if (group.isPending) {
		return <p className="text-muted">{t("common.loading")}</p>;
	}
	if (group.isError) {
		return (
			<>
				{backLink}
				<p role="alert" className="mt-4 text-muted">
					{group.error instanceof ApiError && group.error.status === 404
						? t("groups.notFound")
						: t("common.genericError")}
				</p>
			</>
		);
	}

	const data = group.data;
	const isAdmin = data.role === "admin";
	const backToList = () => void navigate({ to: "/groups", search: {} });

	return (
		<>
			{backLink}
			<div className="mt-4 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
				<h1 className="min-w-0 text-3xl font-semibold tracking-tight break-words">
					{data.name}
				</h1>
				<div className="flex shrink-0 flex-wrap gap-2">
					{isAdmin && (
						<button
							type="button"
							className={secondaryButton}
							onClick={() => setRenaming(true)}
						>
							{t("groups.rename")}
						</button>
					)}
					<button
						type="button"
						className={secondaryButton}
						onClick={() =>
							setConfirming({
								title: t("groups.leaveDialog.title"),
								text: t("groups.leaveDialog.text", { name: data.name }),
								confirm: t("groups.leave"),
								run: () => removeMember(id, meId),
								after: backToList,
							})
						}
					>
						{t("groups.leave")}
					</button>
					{isAdmin && (
						<button
							type="button"
							className={secondaryButton}
							onClick={() =>
								setConfirming({
									title: t("groups.deleteDialog.title"),
									text: t("groups.deleteDialog.text", { name: data.name }),
									confirm: t("groups.delete"),
									run: () => deleteGroup(id),
									after: backToList,
								})
							}
						>
							{t("groups.delete")}
						</button>
					)}
				</div>
			</div>

			{isAdmin && <InviteForm group={data} />}

			<h2 className="mt-8 text-lg font-semibold">{t("groups.members")}</h2>
			<ul className="mt-3 divide-y divide-line rounded-xl border border-line bg-surface">
				{data.members.map((m) => (
					<MemberRow
						key={m.id}
						groupId={id}
						member={m}
						isAdmin={isAdmin}
						isMe={m.id === meId}
						onRemove={() =>
							setConfirming({
								title: t("groups.removeDialog.title"),
								text: t("groups.removeDialog.text", {
									name: m.name ?? m.slug,
								}),
								confirm: t("groups.remove"),
								run: () => removeMember(id, m.id),
							})
						}
					/>
				))}
			</ul>

			<NameDialog
				open={renaming}
				title={t("groups.renameDialog.title")}
				initial={data.name}
				onSubmit={(name) => renameGroup(id, name)}
				onClose={() => setRenaming(false)}
			/>
			<ConfirmDialog action={confirming} onClose={() => setConfirming(null)} />
		</>
	);
}

function MemberRow({
	groupId,
	member,
	isAdmin,
	isMe,
	onRemove,
}: {
	groupId: string;
	member: GroupMember;
	isAdmin: boolean;
	isMe: boolean;
	onRemove: () => void;
}) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const errorText = useGroupError();
	const name = member.name ?? member.slug;
	const change = useMutation({
		mutationFn: (role: GroupRole) => setMemberRole(groupId, member.id, role),
		onSettled: () =>
			void queryClient.invalidateQueries({ queryKey: groupsQueryKey }),
	});

	return (
		<li className="px-4 py-3">
			<div className="flex items-center gap-3">
				<div className="min-w-0 flex-1">
					<p className="truncate font-medium">{name}</p>
					<p className="truncate text-sm text-muted">/{member.slug}</p>
				</div>
				{isAdmin ? (
					<select
						aria-label={t("groups.roleOf", { name })}
						value={member.role}
						disabled={change.isPending}
						onChange={(e) => change.mutate(e.target.value as GroupRole)}
						className="rounded-lg border border-line bg-paper/40 px-2 py-1.5 text-sm"
					>
						<option value="admin">{t("groups.role.admin")}</option>
						<option value="member">{t("groups.role.member")}</option>
					</select>
				) : (
					<span className="text-sm text-muted">
						{t(`groups.role.${member.role}`)}
					</span>
				)}
				{isAdmin && !isMe && (
					<button
						type="button"
						className={iconButton}
						aria-label={t("groups.removeNamed", { name })}
						onClick={onRemove}
					>
						<Trash2 className="size-4" aria-hidden />
					</button>
				)}
			</div>
			{change.isError && (
				<p role="alert" className="mt-2 text-sm text-red-700">
					{errorText(change.error)}
				</p>
			)}
		</li>
	);
}

function InviteForm({ group }: { group: GroupDetails }) {
	const { t, i18n } = useTranslation();
	const queryClient = useQueryClient();
	const errorText = useShareError();
	const [email, setEmail] = useState("");
	const [role, setRole] = useState<GroupRole>("member");
	const refresh = () =>
		queryClient.invalidateQueries({ queryKey: groupsQueryKey });
	const invite = useMutation({
		mutationFn: () =>
			inviteToGroup(group.id, email.trim(), role, i18n.resolvedLanguage),
		onSuccess: () => {
			setEmail("");
			void refresh();
		},
	});
	const revoke = useMutation({
		mutationFn: (id: string) => revokeGroupInvitation(group.id, id),
		onSuccess: () => void refresh(),
	});
	const failure = invite.error ?? revoke.error;
	const invitations = group.invitations ?? [];

	function submit(event: FormEvent) {
		event.preventDefault();
		if (email.trim() !== "") invite.mutate();
	}

	return (
		<div className="mt-8">
			<form onSubmit={submit} noValidate>
				<label htmlFor="group-email" className="block text-sm font-medium">
					{t("sharing.dialog.emailLabel")}
				</label>
				<div className="flex flex-col gap-2 sm:flex-row sm:items-start">
					<input
						id="group-email"
						type="email"
						autoComplete="off"
						aria-describedby="group-email-hint"
						value={email}
						onChange={(e) => setEmail(e.target.value)}
						className={textInput}
					/>
					<select
						aria-label={t("groups.inviteRole")}
						value={role}
						onChange={(e) => setRole(e.target.value as GroupRole)}
						className={`${textInput} sm:w-48`}
					>
						<option value="member">{t("groups.role.member")}</option>
						<option value="admin">{t("groups.role.admin")}</option>
					</select>
					<button
						type="submit"
						className={`${primaryButton} py-2.5 sm:mt-2`}
						disabled={invite.isPending || email.trim() === ""}
					>
						{t("sharing.dialog.invite")}
					</button>
				</div>
				<p id="group-email-hint" className="mt-1 text-sm text-muted">
					{t("sharing.dialog.emailHint")}
				</p>
			</form>
			{failure !== null && (
				<p role="alert" className="mt-2 text-sm text-red-700">
					{errorText(failure)}
				</p>
			)}
			{invitations.length > 0 && (
				<ul className="mt-4 divide-y divide-line rounded-xl border border-line bg-surface">
					{invitations.map((inv) => (
						<li
							key={inv.id}
							className="flex items-center gap-3 px-4 py-3 text-sm"
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
			)}
		</div>
	);
}
