import { useId } from "react";

/** The rose of the favourite star; the only colour that doesn't follow the theme. */
export const STAR_COLOR = "#E11D48";
const STAR =
	"M25.00 1.20L26.59 5.42L31.09 5.62L27.57 8.43L28.76 12.78L25.00 10.30L21.24 12.78L22.43 8.43L18.91 5.62L23.41 5.42Z";

/**
 * The Boardly icon: a dashboard window with a widget and note lines, and a
 * favourite star. Drawn in currentColor, so it follows the theme's text colour.
 * public/favicon.svg holds the same drawing. className sets the size.
 */
export function AppIcon({ className }: { className: string }) {
	const mask = useId();
	return (
		<svg
			viewBox="0 0 32 32"
			className={`${className} shrink-0`}
			aria-hidden
			data-icon="app"
		>
			<mask id={mask}>
				<rect width="32" height="32" fill="#fff" />
				<circle cx="25" cy="7.6" r="8.4" fill="#000" />
			</mask>
			<g
				mask={`url(#${mask})`}
				stroke="currentColor"
				strokeWidth="2.5"
				strokeLinecap="round"
				strokeLinejoin="round"
				fill="none"
			>
				<rect x="3.25" y="5.25" width="25.5" height="22.5" rx="3" />
				<path d="M3.5 11.5h25" />
			</g>
			<rect x="7" y="15" width="8" height="8.5" rx="1.5" fill="currentColor" />
			<path
				d="M18.5 16h5.5M18.5 19.25h5.5M18.5 22.5h3.5"
				stroke="currentColor"
				strokeWidth="2"
				strokeLinecap="round"
			/>
			<path
				d={STAR}
				fill={STAR_COLOR}
				stroke={STAR_COLOR}
				strokeWidth="1"
				strokeLinejoin="round"
			/>
		</svg>
	);
}
