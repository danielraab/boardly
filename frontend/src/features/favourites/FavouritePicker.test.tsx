import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it } from "vitest";
import { type FavouriteNode, favouritesQueryKey } from "./api";
import { FavouritePicker, type PickerMode } from "./FavouritePicker";

const tree: FavouriteNode = {
	id: "root",
	kind: "folder",
	title: null,
	children: [
		{
			id: "work",
			kind: "folder",
			title: "Work",
			children: [
				{
					id: "tools",
					kind: "folder",
					title: "Tools",
					children: [
						{
							id: "gitlab",
							kind: "link",
							title: "GitLab",
							url: "https://gitlab.example.org/",
							iconImageId: "img1",
						},
					],
				},
				{
					id: "mail",
					kind: "link",
					title: null,
					url: "https://mail.example.org/",
				},
			],
		},
		{ id: "empty", kind: "folder", title: "Empty", children: [] },
	],
};

function Harness({
	mode,
	initial,
	disabledIds,
}: {
	mode: PickerMode;
	initial: string;
	disabledIds?: ReadonlySet<string>;
}) {
	const [value, setValue] = useState(initial);
	return (
		<>
			<FavouritePicker
				id="p"
				label="Pick"
				mode={mode}
				value={value}
				onChange={setValue}
				disabledIds={disabledIds}
			/>
			<output data-testid="value">{value}</output>
		</>
	);
}

function renderPicker(
	mode: PickerMode,
	initial = "",
	data: FavouriteNode = tree,
	disabledIds?: ReadonlySet<string>,
) {
	const client = new QueryClient();
	client.setQueryData(favouritesQueryKey, data);
	render(
		<QueryClientProvider client={client}>
			<Harness mode={mode} initial={initial} disabledIds={disabledIds} />
		</QueryClientProvider>,
	);
	return {
		user: userEvent.setup(),
		group: screen.getByRole("radiogroup", { name: "Pick" }),
		radios: () =>
			within(screen.getByRole("radiogroup"))
				.queryAllByRole("radio")
				.map((r) => r.closest("label")?.textContent),
		value: () => screen.getByTestId("value").textContent,
	};
}

describe("FavouritePicker", () => {
	it("offers folders as a tree in folder mode", async () => {
		const { user, group, radios, value } = renderPicker("folder");
		expect(radios()).toEqual(["Favourites", "Work", "Empty"]);
		expect(within(group).queryByRole("radio", { checked: true })).toBeNull();
		// Empty has no subfolders, so it can't be expanded.
		expect(
			within(group).queryByRole("button", { name: "Expand Empty" }),
		).toBeNull();

		await user.click(
			within(group).getByRole("button", { name: "Expand Work" }),
		);
		expect(radios()).toEqual(["Favourites", "Work", "Tools", "Empty"]);
		await user.click(within(group).getByRole("radio", { name: "Tools" }));
		expect(value()).toBe("tools");
		expect(within(group).getByRole("radio", { name: "Tools" })).toBeChecked();
		expect(
			screen.getByText("Selected: Favourites / Work / Tools"),
		).toBeVisible();

		await user.click(
			within(group).getByRole("button", { name: "Collapse Work" }),
		);
		expect(radios()).toEqual(["Favourites", "Work", "Empty"]);
		// Still named below the tree.
		expect(
			screen.getByText("Selected: Favourites / Work / Tools"),
		).toBeVisible();
	});

	it("offers links in link mode, opening the folders above the value", async () => {
		const { user, group, radios, value } = renderPicker("link", "gitlab");
		// Work and Tools are open; Empty has no links and isn't shown.
		expect(radios()).toEqual(["GitLab", "https://mail.example.org/"]);
		expect(within(group).queryByText("Empty")).toBeNull();
		expect(within(group).getByRole("radio", { name: "GitLab" })).toBeChecked();
		expect(
			within(group)
				.getByRole("radio", { name: "GitLab" })
				.closest("label")
				?.querySelector("img"),
		).toHaveAttribute("src", "/api/images/img1");

		await user.click(
			within(group).getByRole("radio", { name: "https://mail.example.org/" }),
		);
		expect(value()).toBe("mail");
		expect(
			screen.getByText(
				"Selected: Favourites / Work / https://mail.example.org/",
			),
		).toBeVisible();
	});

	it("searches by name and URL, ignoring case, with paths", async () => {
		const { user, group, radios, value } = renderPicker("link");
		const search = screen.getByRole("searchbox", { name: "Search favourites" });

		await user.type(search, "GITLAB");
		expect(radios()).toEqual(["GitLab"]);
		expect(
			within(group).getByRole("radio", { name: "GitLab" }),
		).toHaveAccessibleDescription("Favourites / Work / Tools");

		await user.clear(search);
		await user.type(search, "mail.example");
		await user.click(
			within(group).getByRole("radio", { name: "https://mail.example.org/" }),
		);
		expect(value()).toBe("mail");

		await user.type(search, "zzz");
		expect(radios()).toEqual([]);
		expect(
			within(group).getByText("Nothing matches your search."),
		).toBeVisible();
	});

	it("says when there are no links", () => {
		const { group } = renderPicker("link", "", {
			id: "root",
			kind: "folder",
			title: null,
			children: [{ id: "f", kind: "folder", title: "F", children: [] }],
		});
		expect(
			within(group).getByText("There are no links in your favourites yet."),
		).toBeVisible();
	});

	it("selects nothing for a value that isn't in the tree", () => {
		const { group } = renderPicker("folder", "gone");
		expect(within(group).queryByRole("radio", { checked: true })).toBeNull();
		expect(screen.queryByText(/^Selected:/)).toBeNull();
	});

	it("can't choose disabled folders or anything below them", async () => {
		const { user, group, value } = renderPicker(
			"folder",
			"",
			tree,
			new Set(["work"]),
		);
		await user.click(
			within(group).getByRole("button", { name: "Expand Work" }),
		);
		expect(within(group).getByRole("radio", { name: "Work" })).toBeDisabled();
		expect(within(group).getByRole("radio", { name: "Tools" })).toBeDisabled();
		expect(within(group).getByRole("radio", { name: "Empty" })).toBeEnabled();
		await user.click(within(group).getByRole("radio", { name: "Tools" }));
		expect(value()).toBe("");
	});
});
