import {
	createMemoryHistory,
	createRouter,
	RouterProvider,
} from "@tanstack/react-router";
import { render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { routeTree } from "#/routeTree.gen";

/** Renders the whole app (shell + routes) at the given path. */
export function renderApp(path = "/") {
	const router = createRouter({
		routeTree,
		history: createMemoryHistory({ initialEntries: [path] }),
	});
	const user = userEvent.setup();
	const result = render(<RouterProvider router={router} />);
	return { ...result, router, user };
}
