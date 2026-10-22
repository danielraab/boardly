import type { ParseKeys } from "i18next";
import { StickyNote } from "lucide-react";
import { useState } from "react";
import type { ContentWidget } from "#/features/dashboards/api";
import type { WidgetKind } from "#/features/dashboards/widgets/registry";
import { WidgetSettingsForm } from "#/features/dashboards/widgets/WidgetSettingsForm";

/** A content widget of the test-only kind "note". */
export function note(
	id: string,
	fields: Partial<ContentWidget> = {},
): ContentWidget {
	return {
		id,
		kind: "note",
		width: 2,
		height: "auto",
		config: { text: id },
		...fields,
	};
}

/**
 * The test-only kind "note": it shows its text and, when the server resolved
 * any, its data. Its settings have one required text field.
 */
export const noteKind: WidgetKind = {
	kind: "note",
	icon: StickyNote,
	labelKey: "Note" as ParseKeys,
	descriptionKey: "Shows a text" as ParseKeys,
	defaultConfig: () => ({ text: "" }),
	View: ({ widget, editing }) => (
		<p>
			{(widget.config as { text: string }).text}
			{widget.data !== undefined && ` [${String(widget.data)}]`}
			{editing && " (editing)"}
		</p>
	),
	Settings: (props) => {
		const [text, setText] = useState(
			(props.widget.config as { text: string }).text,
		);
		return (
			<WidgetSettingsForm
				title={props.isNew ? "New note" : "Edit note"}
				widget={props.widget}
				isNew={props.isNew}
				desktopColumns={props.desktopColumns}
				valid={text.trim() !== ""}
				onApply={(size) =>
					props.onApply({ ...props.widget, ...size, config: { text } })
				}
				onDelete={props.onDelete}
			>
				<label className="block text-sm font-medium">
					Text
					<input value={text} onChange={(e) => setText(e.target.value)} />
				</label>
			</WidgetSettingsForm>
		);
	},
};
