import { Globe, type LucideIcon } from "lucide-react";
import { ImageIcon } from "#/components/ImageIcon";
import { findSymbol } from "./symbols";

/**
 * The image if there is one, otherwise the symbol, otherwise the fallback
 * (a globe by default).
 */
export function ChosenIcon({
	imageId,
	icon,
	className,
	fallback,
}: {
	imageId: string | null;
	icon: string | null;
	className: string;
	fallback?: LucideIcon;
}) {
	if (imageId) return <ImageIcon imageId={imageId} className={className} />;
	const symbol = findSymbol(icon);
	const Icon = symbol ?? fallback ?? Globe;
	return (
		<Icon
			className={`${className} shrink-0 text-muted`}
			aria-hidden
			data-icon={symbol ? icon : fallback ? "fallback" : "globe"}
		/>
	);
}
