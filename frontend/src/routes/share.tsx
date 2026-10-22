import { createFileRoute, redirect } from "@tanstack/react-router";
import { meQueryOptions } from "#/features/auth/session";
import { SharePage } from "#/features/share/SharePage";
import type { ShareParams } from "#/features/share/sharedLink";

// Search values are parsed like JSON, so a shared "2024" arrives as a number.
const text = (value: unknown) =>
	typeof value === "string" || typeof value === "number"
		? String(value)
		: undefined;

export const Route = createFileRoute("/share")({
	// The manifest's share target opens /share?title=…&text=…&url=….
	validateSearch: (search: Record<string, unknown>): ShareParams => ({
		title: text(search.title),
		text: text(search.text),
		url: text(search.url),
	}),
	beforeLoad: async ({ context, location }) => {
		if (!(await context.queryClient.ensureQueryData(meQueryOptions))) {
			throw redirect({ to: "/login", search: { redirect: location.href } });
		}
	},
	component: ShareRoute,
});

function ShareRoute() {
	return <SharePage shared={Route.useSearch()} />;
}
