import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it } from "vitest";
import { fakeNotesServer, note, paragraphs } from "#/test/notes";
import type { Note } from "./api";
import { useAutosave } from "./useAutosave";

function renderSaver(initial: Note) {
	const client = new QueryClient();
	const wrapper = ({ children }: { children: ReactNode }) => (
		<QueryClientProvider client={client}>{children}</QueryClientProvider>
	);
	return renderHook(
		({ n }: { n: Note }) => useAutosave({ note: n, draft: n }),
		{ wrapper, initialProps: { n: initial } },
	);
}

const patches = (calls: { method: string; body: unknown }[]) =>
	calls.filter((c) => c.method === "PATCH").map((c) => c.body);

describe("useAutosave", () => {
	it("sends saveNow changes one after another with the latest version", async () => {
		const n = note("n1");
		const server = fakeNotesServer([n]);
		const { result } = renderSaver(n);
		act(() => {
			result.current.saveNow({ content: paragraphs("a") });
			result.current.saveNow({ content: paragraphs("b") });
		});
		await waitFor(() => expect(server.notes[0].version).toBe(3));
		expect(patches(server.api.calls)).toEqual([
			{ version: 1, content: paragraphs("a") },
			{ version: 2, content: paragraphs("b") },
		]);
	});

	it("takes over a newer version of the note while idle", async () => {
		const n = note("n1");
		const server = fakeNotesServer([n]);
		const { result, rerender } = renderSaver(n);
		// Changed elsewhere, e.g. the category from the card's badge.
		server.notes[0].version = 4;
		rerender({ n: { ...n, version: 4 } });
		act(() => result.current.saveNow({ title: "x" }));
		await waitFor(() => expect(server.notes[0].version).toBe(5));
		expect(patches(server.api.calls)).toEqual([{ version: 4, title: "x" }]);
	});
});
