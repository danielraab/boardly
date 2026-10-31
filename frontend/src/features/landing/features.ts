import {
	FolderTree,
	LayoutGrid,
	type LucideIcon,
	StickyNote,
} from "lucide-react";

export type FeatureId = "dashboards" | "favourites" | "notes";

/** Features shown on the landing page; texts live under `features.<id>.*` in the catalogs. */
export const features: { id: FeatureId; icon: LucideIcon }[] = [
	{ id: "dashboards", icon: LayoutGrid },
	{ id: "favourites", icon: FolderTree },
	{ id: "notes", icon: StickyNote },
];
