import { Popover } from "@base-ui/react/popover";
import { Ban, Palette } from "lucide-react";
import { useTranslation } from "react-i18next";
import { type NoteColour, noteBackground, noteColours } from "./colours";

/**
 * The note colours and "No colour" as a group of radio swatches, each shown
 * in the active theme.
 */
export function ColourChoices({
	value,
	onChange,
	name,
}: {
	value: NoteColour | null;
	onChange: (colour: NoteColour | null) => void;
	/** The radio group's name, unique on the page. */
	name: string;
}) {
	const { t } = useTranslation();
	return (
		<div
			role="radiogroup"
			aria-label={t("notes.colours.label")}
			className="flex flex-wrap gap-1.5"
		>
			{[null, ...noteColours].map((colour) => {
				const label = colour
					? t(`notes.colours.${colour}`)
					: t("notes.colours.none");
				return (
					<label
						key={colour ?? "none"}
						title={label}
						className="inline-flex size-8 cursor-pointer items-center justify-center rounded-full border border-line bg-surface text-muted has-[:checked]:ring-2 has-[:checked]:ring-ink has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-accent"
						style={{ background: noteBackground(colour) }}
					>
						<input
							type="radio"
							name={name}
							checked={value === colour}
							onChange={() => onChange(colour)}
							className="sr-only"
						/>
						{!colour && <Ban className="size-4" aria-hidden />}
						<span className="sr-only">{label}</span>
					</label>
				);
			})}
		</div>
	);
}

/** An icon button that opens the colour choices in a popover. */
export function ColourButton({
	value,
	onChange,
	disabled,
}: {
	value: NoteColour | null;
	onChange: (colour: NoteColour | null) => void;
	disabled?: boolean;
}) {
	const { t } = useTranslation();
	const name = value ? t(`notes.colours.${value}`) : t("notes.colours.none");
	return (
		<Popover.Root>
			<Popover.Trigger
				disabled={disabled}
				aria-label={t("notes.colours.button", { name })}
				title={t("notes.colours.button", { name })}
				className="inline-flex size-10 shrink-0 items-center justify-center rounded-lg border border-line text-muted hover:text-ink disabled:opacity-50 data-[popup-open]:ring-2 data-[popup-open]:ring-ink"
				style={{ background: noteBackground(value) }}
			>
				<Palette className="size-5" aria-hidden />
			</Popover.Trigger>
			<Popover.Portal>
				<Popover.Positioner sideOffset={6} align="end">
					<Popover.Popup className="max-w-[calc(100vw-2rem)] rounded-xl border border-line bg-surface p-3 text-ink shadow-[0_8px_24px_-12px_var(--color-shadow)] outline-none">
						<Popover.Title className="sr-only">
							{t("notes.colours.label")}
						</Popover.Title>
						<div className="w-[11.5rem]">
							<ColourChoices
								value={value}
								onChange={onChange}
								name="note-colour"
							/>
						</div>
					</Popover.Popup>
				</Popover.Positioner>
			</Popover.Portal>
		</Popover.Root>
	);
}
