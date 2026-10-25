import { useEffect } from "react";
import { appTitle } from "./appTitle";

/** Shows "<title> - <app title>" in the browser tab while the caller is mounted. */
export function usePageTitle(title?: string) {
	useEffect(() => {
		const appName = appTitle();
		document.title = title ? `${title} - ${appName}` : appName;
		return () => {
			document.title = appName;
		};
	}, [title]);
}
