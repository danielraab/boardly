import {
	type DashboardViewData,
	defaultColumns,
	type Section,
	type Widget,
} from "#/features/dashboards/api";

/** A section whose ID is also its title, full width, with a border. */
export function section(
	id: string,
	fields: Partial<Omit<Section, "kind">> = {},
	...children: Widget[]
): Section {
	return {
		id,
		kind: "section",
		width: 6,
		title: id,
		border: true,
		height: "auto",
		children,
		...fields,
	};
}

/** Ann's dashboard "Home" as its owner sees it. */
export function viewData(
	sections: Widget[],
	fields: Partial<DashboardViewData> = {},
): DashboardViewData {
	return {
		id: "d1",
		isDefault: false,
		isPublic: false,
		dashboard: { slug: "home", title: "Home" },
		ownerSlug: "ann",
		isOwner: true,
		dashboards: [{ slug: "home", title: "Home" }],
		columns: defaultColumns,
		layoutVersion: 3,
		sections,
		...fields,
	};
}
