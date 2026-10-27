/**
 * The note colours, by the name that is stored. Each theme defines a
 * --color-note-<name> background for every one of them (see styles.css), so a
 * note's colour follows the active theme. Keep in sync with noteColours in
 * backend/internal/notes/icons.go.
 */
export const noteColours = [
	"red",
	"orange",
	"yellow",
	"green",
	"teal",
	"blue",
	"purple",
	"pink",
	"grey",
] as const;

export type NoteColour = (typeof noteColours)[number];

export function isNoteColour(value: unknown): value is NoteColour {
	return noteColours.includes(value as NoteColour);
}

/** The CSS background of a note colour, or undefined without one. */
export function noteBackground(colour: NoteColour | null): string | undefined {
	return colour ? `var(--color-note-${colour})` : undefined;
}
