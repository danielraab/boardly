import { useEffect } from "react";

const appName = "Boardly";

/** Shows "<title> - Boardly" in the browser tab while the caller is mounted. */
export function usePageTitle(title?: string) {
	useEffect(() => {
		document.title = title ? `${title} - ${appName}` : appName;
		return () => {
			document.title = appName;
		};
	}, [title]);
}
