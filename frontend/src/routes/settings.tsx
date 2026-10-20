import { createFileRoute, redirect } from "@tanstack/react-router";
import { meQueryOptions } from "#/features/auth/session";
import {
	SettingsPage,
	type SettingsTab,
} from "#/features/settings/SettingsPage";

type SettingsSearch = { tab?: SettingsTab };

const tabs: readonly unknown[] = [
	"profile",
	"passkeys",
] satisfies SettingsTab[];

export const Route = createFileRoute("/settings")({
	// Unknown tabs must be set to undefined, not left out: the raw search
	// params are merged under the validated ones.
	validateSearch: (search: Record<string, unknown>): SettingsSearch => ({
		tab: tabs.includes(search.tab) ? (search.tab as SettingsTab) : undefined,
	}),
	beforeLoad: async ({ context }) => {
		if (!(await context.queryClient.ensureQueryData(meQueryOptions))) {
			throw redirect({ to: "/login" });
		}
	},
	component: SettingsRoute,
});

function SettingsRoute() {
	const { tab = "profile" } = Route.useSearch();
	const navigate = Route.useNavigate();
	return (
		<SettingsPage
			tab={tab}
			onTabChange={(next) =>
				void navigate({ search: { tab: next }, replace: true })
			}
		/>
	);
}
