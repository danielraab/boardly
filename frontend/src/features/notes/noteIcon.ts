import type { IconChoice } from "#/components/IconChooser";
import type { NoteIcon } from "./api";

/** The icon chooser's value for a note or category icon. */
export function iconChoice(icon: NoteIcon | null): IconChoice {
	return {
		source: icon?.type ?? "none",
		imageId: icon?.type === "image" ? icon.imageId : null,
		icon: icon?.type === "symbol" ? icon.name : null,
	};
}

/** The note icon an icon chooser's value stands for. */
export function noteIcon(choice: IconChoice): NoteIcon | null {
	if (choice.source === "symbol" && choice.icon)
		return { type: "symbol", name: choice.icon };
	if (choice.source === "image" && choice.imageId)
		return { type: "image", imageId: choice.imageId };
	return null;
}

/** The props of ChosenIcon that show a note icon. */
export function iconProps(icon: NoteIcon | null) {
	return {
		imageId: icon?.type === "image" ? icon.imageId : null,
		icon: icon?.type === "symbol" ? icon.name : null,
	};
}
