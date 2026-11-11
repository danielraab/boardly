import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { primaryButton } from "#/components/buttonStyles";
import { focusPage } from "#/components/pageWidths";
import { groupsQueryKey } from "#/features/groups/api";
import { sharedWithMeQueryKey } from "#/features/sharing/api";
import { ApiError } from "#/lib/api";
import { usePageTitle } from "#/lib/usePageTitle";
import {
	acceptInvitation,
	invitationPreviewQueryOptions,
	openInvitationsQueryKey,
} from "./api";

export function InvitationPage({ token }: { token: string }) {
	const { t } = useTranslation();
	usePageTitle(t("invitations.title"));
	const navigate = useNavigate();
	const queryClient = useQueryClient();
	const preview = useQuery({
		...invitationPreviewQueryOptions(token),
		enabled: token !== "",
	});
	const accept = useMutation({
		mutationFn: () => acceptInvitation({ token }),
		onSuccess: (result) => {
			void queryClient.invalidateQueries({
				queryKey: openInvitationsQueryKey,
			});
			void queryClient.invalidateQueries({ queryKey: sharedWithMeQueryKey });
			void queryClient.invalidateQueries({ queryKey: groupsQueryKey });
			if (result.type === "dashboard" && result.ownerSlug && result.slug) {
				void navigate({
					to: "/$userSlug/$dashboardSlug",
					params: { userSlug: result.ownerSlug, dashboardSlug: result.slug },
				});
			} else {
				void navigate({ to: "/groups", search: { group: result.groupId } });
			}
		},
	});

	const error = preview.error ?? accept.error;
	const code = error instanceof ApiError ? error.code : "";
	const message =
		token === "" || code === "invitation_not_found"
			? t("invitations.invalid")
			: code === "invitation_wrong_account"
				? t("invitations.wrongAccount")
				: error
					? t("common.genericError")
					: null;

	return (
		<section className={`${focusPage} pt-12`}>
			<h1 className="text-2xl font-semibold tracking-tight">
				{t("invitations.title")}
			</h1>
			{message ? (
				<p role="alert" className="mt-4 text-muted">
					{message}
				</p>
			) : preview.data ? (
				<>
					<p className="mt-4">
						{t(`invitations.text.${preview.data.type}`, {
							name: preview.data.inviterName,
							title: preview.data.title,
						})}
					</p>
					<button
						type="button"
						className={`${primaryButton} mt-6`}
						disabled={accept.isPending}
						onClick={() => accept.mutate()}
					>
						{t("invitations.accept")}
					</button>
				</>
			) : (
				<p className="mt-4 text-muted">{t("common.loading")}</p>
			)}
		</section>
	);
}
