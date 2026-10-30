import { createFileRoute, redirect } from "@tanstack/react-router";
import {
	AttachmentsPage,
	type AttachmentsSearch,
	attachmentFilters,
} from "#/features/attachments/AttachmentsPage";
import { meQueryOptions } from "#/features/auth/session";

export const Route = createFileRoute("/attachments")({
	// Invalid values must be set to undefined, not left out: the raw search
	// params are merged under the validated ones.
	validateSearch: (search: Record<string, unknown>): AttachmentsSearch => {
		const filter = attachmentFilters.find(
			(f) => f !== "all" && f === search.filter,
		);
		return { filter: filter === "all" ? undefined : filter };
	},
	beforeLoad: async ({ context }) => {
		if (!(await context.queryClient.ensureQueryData(meQueryOptions))) {
			throw redirect({ to: "/login" });
		}
	},
	component: AttachmentsRoute,
});

function AttachmentsRoute() {
	const search = Route.useSearch();
	const navigate = Route.useNavigate();
	return (
		<AttachmentsPage
			search={search}
			onSearchChange={(next) => void navigate({ search: next })}
		/>
	);
}
