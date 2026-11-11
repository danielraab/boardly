import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, Navigate } from "@tanstack/react-router";
import { Share2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { primaryButton } from "#/components/buttonStyles";
import { widePage } from "#/components/pageWidths";
import { useMe } from "#/features/auth/session";
import { groupsQueryKey } from "#/features/groups/api";
import {
	acceptInvitation,
	type OpenInvitation,
	openInvitationsQueryKey,
	openInvitationsQueryOptions,
} from "#/features/invitations/api";
import { usePageTitle } from "#/lib/usePageTitle";
import { sharedWithMeQueryKey, sharedWithMeQueryOptions } from "./api";

export function SharedPage() {
	const { t } = useTranslation();
	usePageTitle(t("sharing.page.title"));
	const { data: me } = useMe();
	const queryClient = useQueryClient();
	const invitations = useQuery({
		...openInvitationsQueryOptions,
		enabled: !!me,
	});
	const shared = useQuery({ ...sharedWithMeQueryOptions, enabled: !!me });
	const accept = useMutation({
		mutationFn: (id: string) => acceptInvitation({ invitationId: id }),
		onSettled: () => {
			void queryClient.invalidateQueries({
				queryKey: openInvitationsQueryKey,
			});
			void queryClient.invalidateQueries({ queryKey: sharedWithMeQueryKey });
			void queryClient.invalidateQueries({ queryKey: groupsQueryKey });
		},
	});

	if (me === null) return <Navigate to="/login" />;
	if (!me) return null;

	const open = invitations.data ?? [];
	const dashboards = shared.data?.dashboards ?? [];
	const loading = invitations.isPending || shared.isPending;
	const failed = invitations.isError || shared.isError;

	return (
		<section className={`${widePage} pt-12`}>
			<h1 className="text-3xl font-semibold tracking-tight">
				{t("sharing.page.title")}
			</h1>
			<div className="mt-8 space-y-6">
				{accept.isError && (
					<p role="alert" className="text-sm text-red-700">
						{t("invitations.invalid")}
					</p>
				)}
				{open.length > 0 && (
					<div>
						<h2 className="text-lg font-semibold">
							{t("sharing.page.invitations")}
						</h2>
						<ul className="mt-3 divide-y divide-line rounded-xl border border-line bg-surface">
							{open.map((inv) => (
								<InvitationRow
									key={inv.id}
									invitation={inv}
									busy={accept.isPending}
									onAccept={() => accept.mutate(inv.id)}
								/>
							))}
						</ul>
					</div>
				)}
				{loading ? (
					<p className="text-muted">{t("common.loading")}</p>
				) : failed ? (
					<p role="alert" className="text-sm text-red-700">
						{t("common.genericError")}
					</p>
				) : dashboards.length > 0 ? (
					<ul className="divide-y divide-line rounded-xl border border-line bg-surface">
						{dashboards.map((d) => (
							<li key={`${d.ownerSlug}/${d.slug}`} className="px-4 py-3">
								<Link
									to="/$userSlug/$dashboardSlug"
									params={{ userSlug: d.ownerSlug, dashboardSlug: d.slug }}
									className="block truncate font-medium underline-offset-4 hover:underline"
								>
									{d.title}
								</Link>
								<p className="truncate text-sm text-muted">{d.ownerName}</p>
							</li>
						))}
					</ul>
				) : (
					open.length === 0 && (
						<div className="rounded-xl border border-dashed border-line bg-surface px-6 py-12 text-center">
							<Share2
								className="mx-auto size-8 text-muted"
								strokeWidth={1.75}
								aria-hidden
							/>
							<h2 className="mt-4 text-lg font-semibold">
								{t("sharing.page.emptyTitle")}
							</h2>
							<p className="mx-auto mt-1 max-w-[45ch] text-muted">
								{t("sharing.page.emptyText")}
							</p>
						</div>
					)
				)}
			</div>
		</section>
	);
}

function InvitationRow({
	invitation,
	busy,
	onAccept,
}: {
	invitation: OpenInvitation;
	busy: boolean;
	onAccept: () => void;
}) {
	const { t } = useTranslation();
	return (
		<li className="flex items-center gap-3 px-4 py-3">
			<p className="min-w-0 flex-1">
				{t(`invitations.text.${invitation.type}`, {
					name: invitation.inviterName,
					title: invitation.title,
				})}
			</p>
			<button
				type="button"
				className={primaryButton}
				disabled={busy}
				aria-label={t("invitations.acceptNamed", { title: invitation.title })}
				onClick={onAccept}
			>
				{t("invitations.accept")}
			</button>
		</li>
	);
}
