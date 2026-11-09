import { Music, Radio } from "lucide-react";
import type { PlayableSource } from "./api";

/** The icon of a source that has none of its own. */
export const sourceFallbackIcon = (
	source: Partial<Pick<PlayableSource, "kind">>,
) => (source.kind === "embed" ? Music : Radio);
