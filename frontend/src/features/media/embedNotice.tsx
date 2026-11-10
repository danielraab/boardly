import { AlertDialog } from "@base-ui/react/alert-dialog";
import {
	createContext,
	type ReactNode,
	useCallback,
	useContext,
	useState,
} from "react";
import { useTranslation } from "react-i18next";
import { primaryButton, secondaryButton } from "#/components/buttonStyles";
import { dialogBackdrop, dialogPopup } from "#/components/dialogStyles";
import type { PlayableSource } from "./api";
import { useMediaPlayer } from "./player";

/** The name to show for each provider of the server's registry. */
const providerNames: Record<string, string> = { spotify: "Spotify" };

const noticeKey = "boardly.embedNotice";

function confirmedProviders() {
	try {
		return (localStorage.getItem(noticeKey) ?? "").split(",").filter(Boolean);
	} catch {
		return [];
	}
}

function rememberProvider(provider: string) {
	try {
		localStorage.setItem(
			noticeKey,
			[...confirmedProviders(), provider].join(","),
		);
	} catch {
		// Without storage the notice is just shown again next time.
	}
}

const OpenEmbedContext = createContext<((s: PlayableSource) => void) | null>(
	null,
);

/** Opens an embed in the dock, after the first-use notice if that is due. */
export function useOpenEmbed() {
	const open = useContext(OpenEmbedContext);
	if (!open) throw new Error("useOpenEmbed outside EmbedNoticeProvider");
	return open;
}

/** Asks once per provider before the browser connects to it. */
export function EmbedNoticeProvider({ children }: { children: ReactNode }) {
	const { t } = useTranslation();
	const { current, play } = useMediaPlayer();
	const [pending, setPending] = useState<PlayableSource | null>(null);
	const currentId = current?.id;

	const open = useCallback(
		(source: PlayableSource) => {
			if (source.id === currentId) return;
			if (confirmedProviders().includes(source.provider ?? "")) play(source);
			else setPending(source);
		},
		[currentId, play],
	);

	const provider = pending?.provider ?? "";
	const name = providerNames[provider] ?? provider;

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
						<div className="mt-6 flex justify-end gap-2">
							<AlertDialog.Close className={secondaryButton}>
								{t("media.cancel")}
							</AlertDialog.Close>
							<button
								type="button"
								onClick={() => {
									if (!pending) return;
									rememberProvider(provider);
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
