import { AlertDialog } from "@base-ui/react/alert-dialog";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { dangerButton, secondaryButton } from "#/components/buttonStyles";
import { dialogBackdrop, dialogPopup } from "#/components/dialogStyles";
import {
	type Dashboard,
	dashboardsQueryKey,
	dashboardViewQueryKey,
	deleteDashboard,
} from "./api";

export function DeleteDashboardDialog({
	dashboard,
	onClose,
}: {
	dashboard: Dashboard | null;
	onClose: () => void;
}) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const remove = useMutation({
		mutationFn: (id: string) => deleteDashboard(id),
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: dashboardsQueryKey });
			void queryClient.invalidateQueries({ queryKey: dashboardViewQueryKey });
			onClose();
		},
	});

	return (
		<AlertDialog.Root
			open={dashboard !== null}
			onOpenChange={(open) => {
				if (!open) {
					remove.reset();
					onClose();
				}
			}}
		>
			<AlertDialog.Portal>
				<AlertDialog.Backdrop className={dialogBackdrop} />
				<AlertDialog.Popup className={dialogPopup}>
					{dashboard && (
						<>
							<AlertDialog.Title className="text-lg font-semibold">
								{t("dashboards.deleteDialog.title")}
							</AlertDialog.Title>
							<AlertDialog.Description className="mt-1 text-sm text-muted">
								{t("dashboards.deleteDialog.text", { title: dashboard.title })}
							</AlertDialog.Description>
						</>
					)}
					{remove.isError && (
						<p role="alert" className="mt-3 text-sm text-red-700">
							{t("common.genericError")}
						</p>
					)}
					<div className="mt-6 flex justify-end gap-3">
						<AlertDialog.Close className={secondaryButton}>
							{t("dashboards.deleteDialog.cancel")}
						</AlertDialog.Close>
						<button
							type="button"
							className={dangerButton}
							disabled={remove.isPending}
							onClick={() => dashboard && remove.mutate(dashboard.id)}
						>
							{t("dashboards.deleteDialog.confirm")}
						</button>
					</div>
				</AlertDialog.Popup>
			</AlertDialog.Portal>
		</AlertDialog.Root>
	);
}
