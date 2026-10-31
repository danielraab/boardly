import { useLexicalComposerContext } from "@lexical/react/LexicalComposerContext";
import { useLexicalEditable } from "@lexical/react/useLexicalEditable";
import {
	$getNodeByKey,
	COMMAND_PRIORITY_EDITOR,
	createCommand,
	DecoratorNode,
	type LexicalCommand,
	type LexicalNode,
	type NodeKey,
	type SerializedLexicalNode,
	type Spread,
} from "lexical";
import {
	createContext,
	type ReactNode,
	useContext,
	useEffect,
	useMemo,
	useRef,
	useState,
} from "react";
import type { LocationValue } from "../document/types";
import { LocationChip } from "../locations/LocationChip";
import {
	LocationFormDialog,
	LocationMapDialog,
} from "../locations/LocationDialogs";
import { $insertBlocksAtCursor, useLastSelection } from "./insertBlock";

/** What the location blocks of an editor do when activated. */
type EditorLocations = {
	/** Opens the map of the location node with the key; chip gets the focus back. */
	open: (key: NodeKey, chip: HTMLElement) => void;
};

export const EditorLocationsContext = createContext<EditorLocations>({
	open: () => {},
});

function EditorLocation({
	nodeKey,
	location,
}: {
	nodeKey: NodeKey;
	location: LocationValue;
}) {
	const { open } = useContext(EditorLocationsContext);
	return (
		<LocationChip location={location} onOpen={(chip) => open(nodeKey, chip)} />
	);
}

export type SerializedLocationNode = Spread<
	{ location: LocationValue },
	SerializedLexicalNode
>;

/** A location block; the document stores it as {type: "location", lat, lng, label?}. */
export class LocationNode extends DecoratorNode<ReactNode> {
	__location: LocationValue;

	static getType(): string {
		return "location";
	}

	static clone(node: LocationNode): LocationNode {
		return new LocationNode(node.__location, node.__key);
	}

	static importJSON(json: SerializedLocationNode): LocationNode {
		return $createLocationNode(json.location);
	}

	constructor(location: LocationValue, key?: NodeKey) {
		super(key);
		this.__location = location;
	}

	exportJSON(): SerializedLocationNode {
		return { ...super.exportJSON(), location: this.__location };
	}

	createDOM(): HTMLElement {
		return document.createElement("div");
	}

	updateDOM(): false {
		return false;
	}

	isInline(): false {
		return false;
	}

	getLocation(): LocationValue {
		return this.getLatest().__location;
	}

	setLocation(location: LocationValue): this {
		const writable = this.getWritable();
		writable.__location = location;
		return writable;
	}

	getTextContent(): string {
		return "";
	}

	decorate(): ReactNode {
		return <EditorLocation nodeKey={this.__key} location={this.__location} />;
	}
}

/** A location node; a label that is empty or missing is left out. */
export function $createLocationNode({
	lat,
	lng,
	label,
}: LocationValue): LocationNode {
	return new LocationNode(label ? { lat, lng, label } : { lat, lng });
}

export function $isLocationNode(
	node: LexicalNode | null | undefined,
): node is LocationNode {
	return node instanceof LocationNode;
}

/** Opens the location form to add a location at the cursor, or at the end. */
export const OPEN_LOCATION_FORM_COMMAND: LexicalCommand<void> = createCommand(
	"OPEN_LOCATION_FORM_COMMAND",
);

/** The open dialog: the map of a location, or the form for one (key null adds one). */
type OpenDialog =
	| { mode: "view"; key: NodeKey; location: LocationValue }
	| { mode: "form"; key: NodeKey | null; location: LocationValue | null };

/**
 * The location dialogs of an editor around children, which must hold the
 * content editable: activating a location chip opens its map, whose Edit
 * button (only while the editor is editable) opens the location form, and
 * OPEN_LOCATION_FORM_COMMAND opens the form to add one.
 */
export function EditorLocations({ children }: { children: ReactNode }) {
	const [editor] = useLexicalComposerContext();
	const editable = useLexicalEditable();
	const lastSelection = useLastSelection(editor);
	const [dialog, setDialog] = useState<OpenDialog | null>(null);
	const [session, setSession] = useState(0);
	// The chip the open dialog belongs to, which gets the focus back.
	const chip = useRef<HTMLElement | null>(null);

	const openForm = (key: NodeKey | null, location: LocationValue | null) => {
		setSession((s) => s + 1);
		setDialog({ mode: "form", key, location });
	};

	useEffect(
		() =>
			editor.registerCommand(
				OPEN_LOCATION_FORM_COMMAND,
				() => {
					if (!editor.isEditable()) return false;
					chip.current = null;
					setSession((s) => s + 1);
					setDialog({ mode: "form", key: null, location: null });
					return true;
				},
				COMMAND_PRIORITY_EDITOR,
			),
		[editor],
	);

	const context = useMemo<EditorLocations>(
		() => ({
			open: (key, element) => {
				const location = editor.getEditorState().read(() => {
					const node = $getNodeByKey(key);
					return $isLocationNode(node) ? node.getLocation() : null;
				});
				if (!location) return;
				chip.current = element;
				setDialog({ mode: "view", key, location });
			},
		}),
		[editor],
	);

	const close = () => setDialog(null);
	const form = dialog?.mode === "form" ? dialog : null;

	function save(location: LocationValue) {
		const key = form?.key ?? null;
		editor.update(() => {
			if (key === null) {
				$insertBlocksAtCursor(
					[$createLocationNode(location)],
					lastSelection.current,
				);
				return;
			}
			const node = $getNodeByKey(key);
			if ($isLocationNode(node)) node.setLocation(location);
		});
		close();
	}

	function remove() {
		const key = form?.key;
		if (key)
			editor.update(() => {
				$getNodeByKey(key)?.remove();
			});
		close();
	}

	return (
		<EditorLocationsContext value={context}>
			{children}
			<LocationMapDialog
				location={dialog?.mode === "view" ? dialog.location : null}
				returnFocus={chip.current}
				onClose={close}
				onEdit={
					editable && dialog?.mode === "view"
						? () => openForm(dialog.key, dialog.location)
						: undefined
				}
			/>
			<LocationFormDialog
				open={form !== null}
				session={session}
				initial={form?.location ?? null}
				returnFocus={chip.current}
				onSave={save}
				onRemove={form?.key ? remove : undefined}
				onClose={close}
			/>
		</EditorLocationsContext>
	);
}
