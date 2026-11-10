import { AlertDialog } from "@base-ui/react/alert-dialog";
import {
	createContext,
	type ReactNode,
	useCallback,
	useContext,
	useId,
	useState,
} from "react";
import { useTranslation } from "react-i18next";
import { primaryButton, secondaryButton } from "#/components/buttonStyles";
import { dialogBackdrop, dialogPopup } from "#/components/dialogStyles";
import type { PlayableSource } from "./api";
import { allowEmbeds, isEmbedAllowed, providerName } from "./embedConsent";
import { useMediaPlayer } from "./player";

const OpenEmbedContext = createContext<((s: PlayableSource) => void) | null>(
	null,
);

/** Opens an embed in the dock, after the consent notice if that is due. */
export function useOpenEmbed() {
	const open = useContext(OpenEmbedContext);
	if (!open) throw new Error("useOpenEmbed outside EmbedNoticeProvider");
	return open;
}

/** Asks for consent before the browser connects to a provider. */
export function EmbedNoticeProvider({ children }: { children: ReactNode }) {
	const { t } = useTranslation();
	const { current, play } = useMediaPlayer();
	const [pending, setPending] = useState<PlayableSource | null>(null);
	const [always, setAlways] = useState(false);
	const alwaysId = useId();
	const currentId = current?.id;

	const open = useCallback(
		(source: PlayableSource) => {
			if (source.id === currentId) return;
			if (isEmbedAllowed(source.provider ?? "")) {
				play(source);
			} else {
				setAlways(false);
				setPending(source);
			}
		},
		[currentId, play],
	);

	const provider = pending?.provider ?? "";
	const name = providerName(provider);

	return (
		<OpenEmbedContext.Provider value={open}>
			{children}
			<AlertDialog.Root
				open={pending !== null}
				onOpenChange={(isOpen) => !isOpen && setPending(null)}
			>
				<AlertDialog.Portal>
					<AlertDialog.Backdrop className={dialogBackdrop} />
					<AlertDialog.Popup className={dialogPopup}>
						<AlertDialog.Title className="text-lg font-semibold">
							{t("player.embedNotice.title", { provider: name })}
						</AlertDialog.Title>
						<AlertDialog.Description className="mt-2 text-muted">
							{t("player.embedNotice.text", { provider: name })}
						</AlertDialog.Description>
						<div className="mt-4 flex items-center gap-2 text-sm text-muted">
							<input
								id={alwaysId}
								type="checkbox"
								checked={always}
								onChange={(e) => setAlways(e.target.checked)}
								className="size-4 accent-ink"
							/>
							<label htmlFor={alwaysId}>
								{t("player.embedNotice.always", { provider: name })}
							</label>
						</div>
						<div className="mt-6 flex justify-end gap-2">
							<AlertDialog.Close className={secondaryButton}>
								{t("media.cancel")}
							</AlertDialog.Close>
							<button
								type="button"
								onClick={() => {
									if (!pending) return;
									allowEmbeds(provider, always);
									play(pending);
									setPending(null);
								}}
								className={primaryButton}
							>
								{t("player.embedNotice.open")}
							</button>
						</div>
					</AlertDialog.Popup>
				</AlertDialog.Portal>
			</AlertDialog.Root>
		</OpenEmbedContext.Provider>
	);
}
