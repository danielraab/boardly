import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { apiError, type MockResponse, mockApi } from "#/test/api";
import {
	type FavouriteNode,
	type OfflineCopy,
	offlinePollInterval,
} from "./api";
import { OfflineCopyDialog } from "./OfflineCopyDialog";

const go: FavouriteNode = {
	id: "go",
	kind: "link",
	title: "Go tour",
	url: "https://go.dev/tour",
};

const ready: OfflineCopy = {
	status: "ready",
	capturedAt: "2026-10-01T08:00:00.000Z",
	title: "A Tour of Go",
	hasArticle: true,
	hasScreenshot: true,
	error: null,
};

function renderDialog(
	status: () => OfflineCopy,
	refresh: () => MockResponse = () => ({
		status: 202,
		body: { ...status(), status: "pending" },
	}),
) {
	const api = mockApi({
		"GET /api/favourites/go/offline": () => ({ body: status() }),
		"POST /api/favourites/go/offline": refresh,
	});
	const onClose = vi.fn();
	render(
		<QueryClientProvider
			client={
				new QueryClient({ defaultOptions: { queries: { retry: false } } })
			}
		>
			<OfflineCopyDialog link={go} onClose={onClose} />
		</QueryClientProvider>,
	);
	return { api, onClose };
}

afterEach(() => {
	vi.useRealTimers();
});

describe("OfflineCopyDialog", () => {
	it("shows the article of a ready copy in a sandboxed frame", async () => {
		renderDialog(() => ready);
		const dialog = await screen.findByRole("dialog", { name: "Go tour" });
		const time = new Intl.DateTimeFormat("en", {
			dateStyle: "medium",
			timeStyle: "short",
		}).format(new Date(ready.capturedAt as string));
		expect(await within(dialog).findByText(`Saved ${time}`)).toBeVisible();
		expect(
			within(dialog).getByRole("tab", { name: "Article" }),
		).toHaveAttribute("aria-selected", "true");
		const frame = within(dialog).getByTitle("Article of Go tour");
		expect(frame).toHaveAttribute(
			"src",
			`/api/favourites/go/offline/article?v=${encodeURIComponent(ready.capturedAt as string)}`,
		);
		const sandbox = frame.getAttribute("sandbox") ?? "";
		expect(sandbox).not.toContain("allow-scripts");
		expect(sandbox).not.toContain("allow-same-origin");
		expect(frame).toHaveAttribute("referrerpolicy", "no-referrer");
		const original = within(dialog).getByRole("link", {
			name: "Open original",
		});
		expect(original).toHaveAttribute("href", "https://go.dev/tour");
		expect(original).toHaveAttribute("target", "_blank");
	});

	it("shows the screenshot in its tab", async () => {
		renderDialog(() => ready);
		const user = userEvent.setup();
		await user.click(await screen.findByRole("tab", { name: "Screenshot" }));
		expect(
			screen.getByRole("img", { name: "Screenshot of Go tour" }),
		).toHaveAttribute(
			"src",
			expect.stringMatching(/^\/api\/favourites\/go\/offline\/screenshot\?v=/),
		);
	});

	it("disables a missing part and selects the other", async () => {
		renderDialog(() => ({ ...ready, hasScreenshot: false }));
		expect(
			await screen.findByRole("tab", { name: "Screenshot" }),
		).toHaveAttribute("aria-disabled", "true");
		expect(screen.getByRole("tab", { name: "Article" })).toHaveAttribute(
			"aria-selected",
			"true",
		);
	});

	it("selects the screenshot when there is no article", async () => {
		renderDialog(() => ({ ...ready, hasArticle: false }));
		expect(
			await screen.findByRole("tab", { name: "Screenshot" }),
		).toHaveAttribute("aria-selected", "true");
		expect(
			screen.getByRole("img", { name: "Screenshot of Go tour" }),
		).toBeVisible();
	});

	it("says when there is no copy yet", async () => {
		renderDialog(() => ({
			...ready,
			status: "none",
			capturedAt: null,
			hasArticle: false,
			hasScreenshot: false,
		}));
		expect(await screen.findByText("No offline copy yet.")).toBeVisible();
		expect(screen.queryByRole("tab")).toBeNull();
		expect(screen.getByRole("button", { name: "Refresh" })).toBeEnabled();
	});

	it("shows why a capture failed, with the earlier copy", async () => {
		renderDialog(() => ({ ...ready, status: "failed", error: "fetch_failed" }));
		expect(
			await screen.findByText(
				"The offline copy couldn't be saved. The page couldn't be loaded.",
			),
		).toBeVisible();
		expect(screen.getByText(/^Saved /)).toBeVisible();
		expect(screen.getByTitle("Article of Go tour")).toBeVisible();
	});

	it("refreshes, waits while pending and shows the new copy", async () => {
		vi.useFakeTimers({ shouldAdvanceTime: true });
		let current: OfflineCopy = {
			...ready,
			status: "none",
			capturedAt: null,
			hasArticle: false,
			hasScreenshot: false,
		};
		const { api } = renderDialog(
			() => current,
			() => {
				current = { ...current, status: "pending" };
				return { status: 202, body: current };
			},
		);
		const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
		await user.click(await screen.findByRole("button", { name: "Refresh" }));
		expect(await screen.findByText("Saving offline copy…")).toBeVisible();
		expect(screen.getByRole("button", { name: "Refresh" })).toBeDisabled();
		expect(api.callsTo("POST", "/api/favourites/go/offline")).toHaveLength(1);

		current = ready;
		await act(() => vi.advanceTimersByTimeAsync(offlinePollInterval));
		await waitFor(() =>
			expect(screen.getByTitle("Article of Go tour")).toBeVisible(),
		);
		expect(screen.queryByText("Saving offline copy…")).toBeNull();
		expect(screen.getByRole("button", { name: "Refresh" })).toBeEnabled();
	});

	it("says when refreshes are rate limited", async () => {
		renderDialog(
			() => ready,
			() => apiError(429, "rate_limited", { "Retry-After": "60" }),
		);
		const user = userEvent.setup();
		await user.click(await screen.findByRole("button", { name: "Refresh" }));
		expect(await screen.findByRole("alert")).toHaveTextContent(
			"Too many refreshes, try again later.",
		);
	});

	it("closes with Escape and the close button", async () => {
		const { onClose } = renderDialog(() => ready);
		const user = userEvent.setup();
		await screen.findByRole("dialog");
		await user.keyboard("{Escape}");
		expect(onClose).toHaveBeenCalledOnce();
		await user.click(screen.getByRole("button", { name: "Close" }));
		expect(onClose).toHaveBeenCalledTimes(2);
	});
});
