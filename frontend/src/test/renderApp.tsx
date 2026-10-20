import { QueryClientProvider } from "@tanstack/react-query";
import {
	createMemoryHistory,
	createRouter,
	RouterProvider,
} from "@tanstack/react-router";
import { render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createQueryClient } from "#/features/auth/session";
import { routeTree } from "#/routeTree.gen";

/**
 * Renders the whole app (shell + routes) at the given path. Mock the API with
 * mockApi() first; the test setup installs a signed-out default.
 */
export function renderApp(path = "/") {
	const queryClient = createQueryClient();
	const router = createRouter({
		routeTree,
		context: { queryClient },
		history: createMemoryHistory({ initialEntries: [path] }),
	});
	const user = userEvent.setup();
	const result = render(
		<QueryClientProvider client={queryClient}>
			<RouterProvider router={router} />
		</QueryClientProvider>,
	);
	return { ...result, router, user, queryClient };
}
