import { TanStackDevtools } from "@tanstack/react-devtools";
import { TanStackRouterDevtoolsPanel } from "@tanstack/react-router-devtools";

// Loaded only in development (see routes/__root.tsx).
export default function Devtools() {
	return (
		<TanStackDevtools
			config={{ position: "bottom-right" }}
			plugins={[
				{ name: "TanStack Router", render: <TanStackRouterDevtoolsPanel /> },
			]}
		/>
	);
}
