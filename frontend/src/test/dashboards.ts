import {
	type DashboardViewData,
	defaultColumns,
	type Section,
} from "#/features/dashboards/api";

/** A section whose ID is also its title, full width, with a border. */
export function section(
	id: string,
	fields: Partial<Section> = {},
	...children: Section[]
): Section {
	return {
		id,
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
	sections: Section[],
	fields: Partial<DashboardViewData> = {},
): DashboardViewData {
	return {
		id: "d1",
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
