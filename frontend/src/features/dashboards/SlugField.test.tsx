import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it } from "vitest";
import { SlugField } from "./SlugField";

function Field({
	initial = "",
	saved,
	reserved = false,
}: {
	initial?: string;
	saved?: string;
	reserved?: boolean;
}) {
	const [value, setValue] = useState(initial);
	return (
		<SlugField
			id="slug"
			label="Address"
			prefix="/ann/"
			value={value}
			onChange={setValue}
			saved={saved}
			reserved={reserved}
		/>
	);
}

describe("SlugField", () => {
	it("lowercases typed letters", async () => {
		const user = userEvent.setup();
		render(<Field />);
		await user.type(screen.getByLabelText("Address"), "Work");
		expect(screen.getByLabelText("Address")).toHaveValue("work");
		expect(screen.queryByRole("alert")).toBeNull();
	});

	it("shows the format hint for an invalid slug", async () => {
		const user = userEvent.setup();
		render(<Field />);
		await user.type(screen.getByLabelText("Address"), "my_work");
		expect(screen.getByRole("alert")).toHaveTextContent(
			"Use 1 to 32 characters",
		);
		expect(screen.getByLabelText("Address")).toHaveAttribute(
			"aria-invalid",
			"true",
		);
	});

	it("shows the reserved hint only for user slugs", async () => {
		const user = userEvent.setup();
		const { unmount } = render(<Field reserved />);
		await user.type(screen.getByLabelText("Address"), "login");
		expect(screen.getByRole("alert")).toHaveTextContent("reserved");
		unmount();

		render(<Field />);
		await user.type(screen.getByLabelText("Address"), "login");
		expect(screen.queryByRole("alert")).toBeNull();
	});

	it("warns about old links only when the value differs from the saved one", async () => {
		const user = userEvent.setup();
		render(<Field initial="home" saved="home" />);
		const hint = "Links that use the old address will stop working.";
		expect(screen.queryByText(hint)).toBeNull();

		await user.type(screen.getByLabelText("Address"), "2");
		expect(screen.getByText(hint)).toBeVisible();

		await user.type(screen.getByLabelText("Address"), "{Backspace}");
		expect(screen.queryByText(hint)).toBeNull();
	});
});
