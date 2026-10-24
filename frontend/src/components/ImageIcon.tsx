import { Globe } from "lucide-react";
import { useState } from "react";

/** The URL of a stored image. */
export function imageUrl(id: string) {
	return `/api/images/${encodeURIComponent(id)}`;
}

/**
 * A decorative icon from the image store, such as a favicon. Without an image,
 * or when it can't be loaded, it shows a globe instead. className sets the size.
 * A changed version loads the image again, for favicons refreshed in place.
 */
export function ImageIcon({
	imageId,
	className,
	version,
}: {
	imageId: string | null | undefined;
	className: string;
	version?: number;
}) {
	const [failed, setFailed] = useState<string | null>(null);
	if (!imageId || failed === imageId) {
		return (
			<Globe
				className={`${className} shrink-0 text-muted`}
				aria-hidden
				data-icon="globe"
			/>
		);
	}
	return (
		<img
			src={version ? `${imageUrl(imageId)}?v=${version}` : imageUrl(imageId)}
			alt=""
			aria-hidden
			loading="lazy"
			onError={() => setFailed(imageId)}
			className={`${className} shrink-0 object-contain`}
		/>
	);
}
