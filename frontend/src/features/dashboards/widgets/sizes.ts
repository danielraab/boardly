/** The S/M/L size of a widget's content. */
export type WidgetSize = "S" | "M" | "L";

export const widgetSizes: WidgetSize[] = ["S", "M", "L"];

/** Icon size, text size, row padding and indent (in px) of each size. */
export const sizeStyles: Record<
	WidgetSize,
	{ icon: string; text: string; row: string; indent: number }
> = {
	S: { icon: "size-3.5", text: "text-xs", row: "py-0.5", indent: 12 },
	M: { icon: "size-4", text: "text-sm", row: "py-1", indent: 16 },
	L: { icon: "size-5", text: "text-base", row: "py-1.5", indent: 20 },
};
