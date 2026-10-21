import { useTranslation } from "react-i18next";
import { MAX_SLUG, normalizeSlugInput, slugError } from "./slug";

/** Problems a slug field can show; "taken" comes from the server. */
export type SlugProblem = "invalid" | "reserved" | "taken";

/**
 * A slug input with its address prefix. It lowercases what is typed, shows
 * format and reserved-word problems, and warns that old links stop working
 * once the value differs from the saved one.
 */
export function SlugField({
	id,
	label,
	prefix,
	value,
	onChange,
	saved,
	reserved = false,
	serverProblem = null,
}: {
	id: string;
	label: string;
	/** Shown in front of the input, e.g. "/ann/". */
	prefix: string;
	value: string;
	onChange: (value: string) => void;
	/** The stored slug; undefined while creating. */
	saved?: string;
	/** Also reject reserved words (user slugs). */
	reserved?: boolean;
	serverProblem?: SlugProblem | null;
}) {
	const { t } = useTranslation();
	const problem =
		(value === "" ? null : slugError(value, { reserved })) ?? serverProblem;
	const renamed = saved !== undefined && value !== saved && problem === null;
	const hintId = `${id}-hint`;

	return (
		<div>
			<label htmlFor={id} className="block text-sm font-medium">
				{label}
			</label>
			<div className="mt-2 flex items-center rounded-lg border border-line bg-paper/40 focus-within:border-ink">
				<span className="truncate pl-4 text-muted" aria-hidden>
					{prefix}
				</span>
				<input
					id={id}
					autoComplete="off"
					autoCapitalize="none"
					spellCheck={false}
					maxLength={MAX_SLUG}
					aria-invalid={problem !== null}
					aria-describedby={problem || renamed ? hintId : undefined}
					value={value}
					onChange={(e) => onChange(normalizeSlugInput(e.target.value))}
					className="min-w-0 flex-1 bg-transparent py-2.5 pr-4 text-ink outline-none"
				/>
			</div>
			{problem ? (
				<p id={hintId} role="alert" className="mt-2 text-sm text-red-700">
					{t(`dashboards.slug.${problem}`)}
				</p>
			) : (
				renamed && (
					<p id={hintId} className="mt-2 text-sm text-muted">
						{t("dashboards.slug.renameHint")}
					</p>
				)
			)}
		</div>
	);
}
