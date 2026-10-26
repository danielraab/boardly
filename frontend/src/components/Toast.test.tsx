import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Toast } from "./Toast";

describe("Toast", () => {
	beforeEach(() => vi.useFakeTimers());
	afterEach(() => vi.useRealTimers());

	const message = { id: 1, text: "3 links deleted" };

	it("shows the message with its action and closes after the duration", () => {
		const onClose = vi.fn();
		const onClick = vi.fn();
		render(
			<Toast
				toast={{ ...message, action: { label: "Undo", onClick } }}
				onClose={onClose}
				duration={1000}
			/>,
		);
		expect(screen.getByRole("status")).toHaveTextContent("3 links deleted");
		fireEvent.click(screen.getByRole("button", { name: "Undo" }));
		expect(onClick).toHaveBeenCalled();
		act(() => vi.advanceTimersByTime(999));
		expect(onClose).not.toHaveBeenCalled();
		act(() => vi.advanceTimersByTime(1));
		expect(onClose).toHaveBeenCalledTimes(1);
	});

	it("closes with its close button", () => {
		const onClose = vi.fn();
		render(<Toast toast={message} onClose={onClose} />);
		fireEvent.click(screen.getByRole("button", { name: "Close" }));
		expect(onClose).toHaveBeenCalled();
	});

	it("stops the time while hovered or focused", () => {
		const onClose = vi.fn();
		render(<Toast toast={message} onClose={onClose} duration={1000} />);
		act(() => vi.advanceTimersByTime(600));
		fireEvent.pointerEnter(screen.getByRole("status"));
		act(() => vi.advanceTimersByTime(5000));
		expect(onClose).not.toHaveBeenCalled();
		fireEvent.pointerLeave(screen.getByRole("status"));
		act(() => vi.advanceTimersByTime(399));
		expect(onClose).not.toHaveBeenCalled();
		act(() => vi.advanceTimersByTime(1));
		expect(onClose).toHaveBeenCalledTimes(1);
	});

	it("isn't paused by a toast that was removed under the pointer", () => {
		const onClose = vi.fn();
		const { rerender } = render(
			<Toast toast={message} onClose={onClose} duration={1000} />,
		);
		fireEvent.pointerEnter(screen.getByRole("status"));
		fireEvent.focus(screen.getByRole("button", { name: "Close" }));
		rerender(<Toast toast={null} onClose={onClose} duration={1000} />);
		rerender(
			<Toast
				toast={{ id: 2, text: "Again" }}
				onClose={onClose}
				duration={1000}
			/>,
		);
		act(() => vi.advanceTimersByTime(1000));
		expect(onClose).toHaveBeenCalledTimes(1);
	});
});
