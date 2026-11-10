import { X } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { iconButton } from "#/components/buttonStyles";
import {
	providerName,
	useEmbedConsentStored,
	withdrawEmbeds,
} from "./embedConsent";
import { useMediaPlayer } from "./player";

/** The height of the player of each shape: phones, then larger screens. */
const playerHeight: Record<string, string> = {
	audio: "h-[152px] sm:h-[352px]",
};

/** What the provider's player may use; the same as its own embed snippet. */
const allowedFeatures =
	"autoplay; clipboard-write; encrypted-media; fullscreen; picture-in-picture";

/** With consent stored for provider, takes that consent back. */
function WithdrawLink({ provider }: { provider: string }) {
	const { t } = useTranslation();
	const stored = useEmbedConsentStored(provider);
	const [withdrawn, setWithdrawn] = useState(false);
	const name = providerName(provider);
	if (withdrawn)
		return (
			<p role="status" className="truncate text-xs text-muted">
				{t("player.embedNotice.withdrawn", { provider: name })}
			</p>
		);
	if (!stored) return null;
	return (
		<button
			type="button"
			className="block max-w-full truncate text-xs text-muted underline underline-offset-2 hover:text-ink"
			onClick={() => {
				withdrawEmbeds(provider);
				setWithdrawn(true);
			}}
		>
			{t("player.embedNotice.withdraw", { provider: name })}
		</button>
	);
}

/**
 * Shows the open embed in the provider's own player, in a plain iframe. Like
 * the mini player it stays in the app shell, so the embed keeps playing
 * across pages. Boardly can't see or control what plays inside it: removing
 * the iframe is what stops it.
 */
export function EmbedDock() {
	const { t } = useTranslation();
	const { current, stop } = useMediaPlayer();
	if (current?.kind !== "embed" || !current.embedUrl) return null;

	return (
		<>
			<div aria-hidden className="h-[12.5rem] shrink-0 sm:h-[26rem]" />
			<section
				aria-label={t("player.embedLabel")}
				className="fixed inset-x-0 bottom-0 z-30 border-t border-line bg-surface shadow-[0_-8px_24px_-16px_var(--color-shadow)] sm:inset-x-auto sm:right-4 sm:bottom-4 sm:w-[360px] sm:overflow-hidden sm:rounded-xl sm:border"
			>
				<div className="flex h-12 items-center gap-2 px-3">
					<div className="min-w-0 flex-1">
						<p className="truncate text-sm font-medium">{current.title}</p>
						<WithdrawLink
							key={current.provider}
							provider={current.provider ?? ""}
						/>
					</div>
					<button
						type="button"
						onClick={stop}
						aria-label={t("player.close")}
						className={iconButton}
					>
						<X className="size-4" aria-hidden />
					</button>
				</div>
				{/* Keyed by address: a new title doesn't reload the player. */}
				<iframe
					key={current.embedUrl}
					src={current.embedUrl}
					title={current.title}
					allow={allowedFeatures}
					className={`block w-full border-0 ${playerHeight[current.shape ?? "audio"] ?? playerHeight.audio}`}
				/>
			</section>
		</>
	);
}
