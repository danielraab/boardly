import {
	createContext,
	type FocusEvent,
	type KeyboardEvent,
	useContext,
} from "react";
import type { Section } from "../api";

/** Where a pointer drag into another grid would drop, while dragging. */
export type DropTarget = {
	parentId: string | null;
	index: number;
	/** The section the drop goes in front of, or null for the end of the grid. */
	overId: string | null;
	/** The drop would nest too deep. */
	blocked: boolean;
};

/** What the section grid needs from the editor; absent in view mode. */
export type Editor = {
	onAdd: (parentId: string | null) => void;
	onEdit: (id: string) => void;
	/** The section picked up with the keyboard, if any. */
	pickedId: string | null;
	dropTarget: DropTarget | null;
	onHandleKeyDown: (event: KeyboardEvent, section: Section) => void;
	onHandleBlur: (event: FocusEvent, section: Section) => void;
	registerHandle: (id: string, el: HTMLButtonElement | null) => void;
};

export const EditorContext = createContext<Editor | null>(null);

export function useEditor() {
	return useContext(EditorContext);
}

/** The droppable ID of a section's children area; "root" is the top level. */
export function containerId(parentId: string | null) {
	return `children:${parentId ?? "root"}`;
}

/** The parent ID of a droppable children area, or undefined for other IDs. */
export function parentOfContainer(id: string): string | null | undefined {
	if (!id.startsWith("children:")) return undefined;
	const parent = id.slice("children:".length);
	return parent === "root" ? null : parent;
}
