import {
	createContext,
	type ReactNode,
	useContext,
	useEffect,
	useState,
} from "react";
import type { DashboardViewData } from "./api";

type Value = {
	current: DashboardViewData | null;
	set: (view: DashboardViewData | null) => void;
};

const CurrentDashboardContext = createContext<Value>({
	current: null,
	set: () => {},
});

/** Holds the dashboard the page shows, so the header can offer the switcher. */
export function CurrentDashboardProvider({
	children,
}: {
	children: ReactNode;
}) {
	const [current, set] = useState<DashboardViewData | null>(null);
	return (
		<CurrentDashboardContext.Provider value={{ current, set }}>
			{children}
		</CurrentDashboardContext.Provider>
	);
}

/** The dashboard the page shows, or null on other pages. */
export function useCurrentDashboard() {
	return useContext(CurrentDashboardContext).current;
}

/** Marks view as the shown dashboard while the calling component is mounted. */
export function useShowDashboard(view: DashboardViewData) {
	const { set } = useContext(CurrentDashboardContext);
	useEffect(() => {
		set(view);
		return () => set(null);
	}, [set, view]);
}
