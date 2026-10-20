import type { QueryClient } from "@tanstack/react-query";
import { createRootRouteWithContext, Outlet } from "@tanstack/react-router";
import { lazy, Suspense } from "react";
import { AppShell } from "#/components/AppShell";
import { NotFound } from "#/components/NotFound";

import "../styles.css";

const Devtools =
	import.meta.env.DEV && import.meta.env.MODE !== "test"
		? lazy(() => import("#/components/Devtools"))
		: () => null;

export type RouterContext = { queryClient: QueryClient };

export const Route = createRootRouteWithContext<RouterContext>()({
	component: RootComponent,
	notFoundComponent: NotFound,
});

function RootComponent() {
	return (
		<>
			<AppShell>
				<Outlet />
			</AppShell>
			<Suspense>
				<Devtools />
			</Suspense>
		</>
	);
}
