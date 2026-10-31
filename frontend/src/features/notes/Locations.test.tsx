import {
	act,
	fireEvent,
	screen,
	waitFor,
	within,
} from "@testing-library/react";
import { $getRoot, $isElementNode } from "lexical";
import { useEffect } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { editorOf } from "#/test/lexical";
import { fakeNotesServer, note } from "#/test/notes";
import { renderApp } from "#/test/renderApp";
import type { Block, LocationBlock } from "./document/types";

// jsdom has no WebGL, so the map is a stub: it shows its point and, when it
// can pick, offers a button that picks a fixed point.
const map = vi.hoisted(() => ({ loaded: 0, fail: false }));
vi.mock("./locations/LocationMap", () => {
	map.loaded++;
	return {
		default: function StubMap({
			point,
			onPick,
			onError,
		}: {
			point: { lat: number; lng: number } | null;
			onPick?: (p: { lat: number; lng: number }) => void;
			onError: () => void;
		}) {
			useEffect(() => {
				if (map.fail) onError();
			}, [onError]);
			return (
				<div data-testid="map">
					{point ? `pin at ${point.lat}, ${point.lng}` : "no pin"}
					{onPick && (
						<button
							type="button"
							onClick={() => onPick({ lat: 47.269212, lng: 11.404102 })}
						>
							pick
						</button>
					)}
				</div>
			);
		},
	};
});

beforeEach(() => {
	map.fail = false;
});

const text = (t: string): Block => ({
	type: "paragraph",
	content: [{ type: "text", text: t }],
});

const stephansplatz: LocationBlock = {
	type: "location",
	lat: 48.208176,
	lng: 16.373819,
	label: "Stephansplatz",
};

async function openNote(blocks: Block[]) {
	const server = fakeNotesServer([
		note("n1", { title: "Trip", content: { v: 1, blocks } }),
	]);
	const app = renderApp("/notes/n1");
	const content = await screen.findByRole("textbox", { name: "Content" });
	return { ...server, ...app, content };
}

/** The blocks of the saved note, once the save after version is done. */
async function savedBlocks(
	server: { notes: { version: number; content: { blocks: Block[] } }[] },
	version: number,
) {
	await waitFor(() => expect(server.notes[0].version).toBe(version), {
		timeout: 3000,
	});
	return server.notes[0].content.blocks;
}

describe("locations on cards", () => {
	// First in this file, so that nothing has loaded the map yet.
	it("shows a chip that opens the map without editing or loading it before", async () => {
		fakeNotesServer([
			note("Trip", {
				content: {
					v: 1,
					blocks: [stephansplatz, { type: "location", lat: 48.2, lng: 16.37 }],
				},
			}),
		]);
		const { user } = renderApp("/notes");
		const chip = await screen.findByRole("button", {
			name: "Location: Stephansplatz",
		});
		expect(
			screen.getByRole("button", { name: "Location: 48.2, 16.37" }),
		).toBeInTheDocument();
		expect(map.loaded).toBe(0);
		await user.click(chip);
		const view = await screen.findByRole("dialog", { name: "Stephansplatz" });
		expect(await within(view).findByTestId("map")).toBeInTheDocument();
		expect(map.loaded).toBe(1);
		expect(within(view).queryByRole("button", { name: "Edit" })).toBeNull();
		expect(screen.queryByRole("textbox", { name: "Content" })).toBeNull();
		await user.click(within(view).getByRole("button", { name: "Close" }));
		await waitFor(() => expect(chip).toHaveFocus());
	});
});

describe("adding locations", () => {
	it("offers Add location right after Add attachment", async () => {
		await openNote([]);
		const names = within(screen.getByRole("toolbar"))
			.getAllByRole("button")
			.map((b) => b.getAttribute("aria-label"));
		expect(names.slice(-2)).toEqual(["Add attachment", "Add location"]);
	});

	it("adds a location with typed coordinates at the end", async () => {
		const server = await openNote([text("Trip")]);
		const { user } = server;
		await user.click(screen.getByRole("button", { name: "Add location" }));
		const form = await screen.findByRole("dialog", { name: "Add location" });
		const save = within(form).getByRole("button", { name: "Save" });
		expect(save).toBeDisabled();
		await user.type(within(form).getByLabelText("Latitude"), "48,208176");
		await user.type(within(form).getByLabelText("Longitude"), "16.373819");
		expect(await within(form).findByTestId("map")).toHaveTextContent(
			"pin at 48.208176, 16.373819",
		);
		await user.click(save);
		expect(await savedBlocks(server, 2)).toEqual([
			text("Trip"),
			{ type: "location", lat: 48.208176, lng: 16.373819 },
		]);
		expect(
			screen.getByRole("button", { name: "Location: 48.208176, 16.373819" }),
		).toBeInTheDocument();
	});

	it("adds a location picked on the map at the cursor, with a label", async () => {
		const server = await openNote([text("first"), text("second")]);
		const { user } = server;
		await act(async () => {
			editorOf(server.content).update(
				() => {
					const first = $getRoot().getFirstChild();
					if ($isElementNode(first)) first.selectEnd();
				},
				{ discrete: true },
			);
		});
		await user.click(screen.getByRole("button", { name: "Add location" }));
		const form = await screen.findByRole("dialog", { name: "Add location" });
		await user.click(await within(form).findByRole("button", { name: "pick" }));
		expect(within(form).getByLabelText("Latitude")).toHaveValue("47.269212");
		expect(within(form).getByLabelText("Longitude")).toHaveValue("11.404102");
		await user.type(within(form).getByLabelText(/Label/), "  Innsbruck ");
		await user.click(within(form).getByRole("button", { name: "Save" }));
		expect(await savedBlocks(server, 2)).toEqual([
			text("first"),
			{ type: "location", lat: 47.269212, lng: 11.404102, label: "Innsbruck" },
			text("second"),
		]);
	});

	it("rejects coordinates out of range", async () => {
		const { user } = await openNote([]);
		await user.click(screen.getByRole("button", { name: "Add location" }));
		const form = await screen.findByRole("dialog", { name: "Add location" });
		await user.type(within(form).getByLabelText("Latitude"), "95");
		await user.type(within(form).getByLabelText("Longitude"), "abc");
		expect(
			within(form).getByText("Enter a latitude between -90 and 90."),
		).toBeInTheDocument();
		expect(
			within(form).getByText("Enter a longitude between -180 and 180."),
		).toBeInTheDocument();
		expect(within(form).getByRole("button", { name: "Save" })).toBeDisabled();
	});

	it("inserts nothing when cancelled", async () => {
		const server = await openNote([text("Trip")]);
		const { user } = server;
		await user.click(screen.getByRole("button", { name: "Add location" }));
		const form = await screen.findByRole("dialog", { name: "Add location" });
		await user.type(within(form).getByLabelText("Latitude"), "1");
		await user.type(within(form).getByLabelText("Longitude"), "2");
		await user.click(within(form).getByRole("button", { name: "Cancel" }));
		await waitFor(() =>
			expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
		);
		expect(
			screen.queryByRole("button", { name: /^Location:/ }),
		).not.toBeInTheDocument();
		expect(server.notes[0].version).toBe(1);
	});
});

describe("using the device's position", () => {
	const geolocation = (
		answer: (
			ok: PositionCallback,
			fail: (e: Partial<GeolocationPositionError>) => void,
		) => void,
	) =>
		Object.defineProperty(navigator, "geolocation", {
			configurable: true,
			value: { getCurrentPosition: vi.fn(answer) },
		});

	afterEach(() => {
		Object.defineProperty(navigator, "geolocation", {
			configurable: true,
			value: undefined,
		});
	});

	it("fills in the device's position", async () => {
		geolocation((ok) =>
			ok({
				coords: { latitude: 48.2081763, longitude: 16.3738189 },
			} as GeolocationPosition),
		);
		const { user } = await openNote([]);
		await user.click(screen.getByRole("button", { name: "Add location" }));
		const form = await screen.findByRole("dialog", { name: "Add location" });
		await user.click(
			within(form).getByRole("button", { name: "Use my location" }),
		);
		expect(within(form).getByLabelText("Latitude")).toHaveValue("48.208176");
		expect(within(form).getByLabelText("Longitude")).toHaveValue("16.373819");
		expect(within(form).getByRole("button", { name: "Save" })).toBeEnabled();
	});

	it("says when access was denied and keeps the fields", async () => {
		geolocation((_, fail) => fail({ code: 1, PERMISSION_DENIED: 1 }));
		const { user } = await openNote([]);
		await user.click(screen.getByRole("button", { name: "Add location" }));
		const form = await screen.findByRole("dialog", { name: "Add location" });
		await user.type(within(form).getByLabelText("Latitude"), "1");
		await user.click(
			within(form).getByRole("button", { name: "Use my location" }),
		);
		expect(
			await within(form).findByText("Access to your location was denied."),
		).toBeInTheDocument();
		expect(within(form).getByLabelText("Latitude")).toHaveValue("1");
	});

	it("says when the position can't be found", async () => {
		const { user } = await openNote([]);
		await user.click(screen.getByRole("button", { name: "Add location" }));
		const form = await screen.findByRole("dialog", { name: "Add location" });
		await user.click(
			within(form).getByRole("button", { name: "Use my location" }),
		);
		expect(
			await within(form).findByText("Your location couldn't be found."),
		).toBeInTheDocument();
	});
});

describe("locations in the editor", () => {
	it("opens the map of a location, which can be edited", async () => {
		const server = await openNote([stephansplatz, text("Meet here")]);
		const { user } = server;
		const chip = screen.getByRole("button", {
			name: "Location: Stephansplatz",
		});
		await user.click(chip);
		const view = await screen.findByRole("dialog", { name: "Stephansplatz" });
		expect(view).toHaveTextContent("48.208176, 16.373819");
		expect(await within(view).findByTestId("map")).toHaveTextContent(
			"pin at 48.208176, 16.373819",
		);
		await user.click(within(view).getByRole("button", { name: "Edit" }));
		const form = await screen.findByRole("dialog", { name: "Edit location" });
		expect(within(form).getByLabelText("Latitude")).toHaveValue("48.208176");
		const label = within(form).getByLabelText(/Label/);
		expect(label).toHaveValue("Stephansplatz");
		await user.clear(label);
		await user.type(label, "Dom");
		await user.click(within(form).getByRole("button", { name: "Save" }));
		expect(await savedBlocks(server, 2)).toEqual([
			{ ...stephansplatz, label: "Dom" },
			text("Meet here"),
		]);
		await waitFor(() =>
			expect(
				screen.getByRole("button", { name: "Location: Dom" }),
			).toHaveFocus(),
		);
	});

	// The editor handles Enter and Space itself, which would keep the chip
	// from being activated with the keyboard.
	it("keeps Enter and Space on a chip from the editor", async () => {
		const server = await openNote([stephansplatz, text("b")]);
		const editorKeys = vi.fn();
		server.content.addEventListener("keydown", editorKeys);
		const chip = screen.getByRole("button", {
			name: "Location: Stephansplatz",
		});
		fireEvent.keyDown(chip, { key: "Enter", code: "Enter" });
		fireEvent.keyDown(chip, { key: " ", code: "Space" });
		fireEvent.keyDown(chip, { key: "a", code: "KeyA" });
		expect(editorKeys.mock.calls.map(([e]) => e.key)).toEqual(["a"]);
	});

	it("removes a location from its form", async () => {
		const server = await openNote([text("a"), stephansplatz, text("b")]);
		const { user } = server;
		await user.click(
			screen.getByRole("button", { name: "Location: Stephansplatz" }),
		);
		const view = await screen.findByRole("dialog", { name: "Stephansplatz" });
		await user.click(within(view).getByRole("button", { name: "Edit" }));
		const form = await screen.findByRole("dialog", { name: "Edit location" });
		await user.click(within(form).getByRole("button", { name: "Remove" }));
		expect(await savedBlocks(server, 2)).toEqual([text("a"), text("b")]);
	});

	it("still shows the location when the map can't be loaded", async () => {
		map.fail = true;
		const { user } = await openNote([stephansplatz]);
		await user.click(
			screen.getByRole("button", { name: "Location: Stephansplatz" }),
		);
		const view = await screen.findByRole("dialog", { name: "Stephansplatz" });
		expect(
			await within(view).findByText("The map couldn't be loaded."),
		).toBeInTheDocument();
		expect(view).toHaveTextContent("48.208176, 16.373819");
	});
});
