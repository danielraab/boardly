import { OTPField } from "@base-ui/react/otp-field";
import { Fragment } from "react";
import { useTranslation } from "react-i18next";

/** Number of characters of the one-time code from the sign-in email. */
export const CODE_LENGTH = 6;

const upperCase = (value: string) => value.toUpperCase();

const slot =
	"size-11 rounded-lg border border-line bg-surface text-center font-mono text-xl font-semibold text-ink outline-none focus-visible:border-ink data-[invalid]:border-danger sm:size-12";

/**
 * The one-time code as 6 slots in two groups of three, like "K7Q-4MZ" in the
 * email. Accepts letters and digits only, so a pasted hyphen is dropped.
 */
export function CodeField({
	id,
	value,
	onChange,
	disabled,
}: {
	id: string;
	value: string;
	onChange: (value: string) => void;
	disabled?: boolean;
}) {
	const { t } = useTranslation();
	return (
		<OTPField.Root
			id={id}
			length={CODE_LENGTH}
			validationType="alphanumeric"
			normalizeValue={upperCase}
			value={value}
			onValueChange={onChange}
			disabled={disabled}
			className="mt-2 flex items-center justify-center gap-2"
		>
			{Array.from({ length: CODE_LENGTH }, (_, index) => (
				// biome-ignore lint/suspicious/noArrayIndexKey: fixed slot positions
				<Fragment key={index}>
					{index === CODE_LENGTH / 2 && (
						<OTPField.Separator
							className="w-3 border-t-2 border-muted"
							aria-hidden
						/>
					)}
					<OTPField.Input
						className={slot}
						aria-label={t("login.codeSlot", {
							index: index + 1,
							total: CODE_LENGTH,
						})}
					/>
				</Fragment>
			))}
		</OTPField.Root>
	);
}
