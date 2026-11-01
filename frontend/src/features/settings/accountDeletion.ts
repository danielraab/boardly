import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useSyncExternalStore } from "react";
import { useTranslation } from "react-i18next";
import { meQueryKey, type User } from "#/features/auth/session";
import { api } from "#/lib/api";

// Whether the "deletion cancelled" notice is shown. It lives outside React so
// the banner in the app shell and the settings section share it.
let noticeShown = false;
const listeners = new Set<() => void>();

function setNotice(shown: boolean) {
	noticeShown = shown;
	for (const l of listeners) l();
}

function subscribe(listener: () => void) {
	listeners.add(listener);
	return () => {
		listeners.delete(listener);
	};
}

/** Closes the notice shown after cancelling a deletion. */
export function closeCancelledNotice() {
	setNotice(false);
}

/** The notice shown after cancelling a deletion, and a way to close it. */
export function useCancelledNotice() {
	const shown = useSyncExternalStore(
		subscribe,
		() => noticeShown,
		() => false,
	);
	return { shown, close: closeCancelledNotice };
}

/** Schedules the deletion of the signed-in user's account. */
export function useScheduleDeletion() {
	const queryClient = useQueryClient();
	const { i18n } = useTranslation();
	return useMutation({
		mutationFn: (email: string) =>
			api<User>("/api/me", {
				method: "DELETE",
				json: { email, locale: i18n.resolvedLanguage },
			}),
		onSuccess: (user) => {
			setNotice(false);
			queryClient.setQueryData(meQueryKey, user);
		},
	});
}

/** Cancels the scheduled deletion and shows the notice about revoked apps. */
export function useCancelDeletion() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: () => api<User>("/api/me/restore", { method: "POST" }),
		onSuccess: (user) => {
			queryClient.setQueryData(meQueryKey, user);
			setNotice(true);
		},
	});
}

/** Formats a deletion time with date and time in the app's language. */
export function formatDeletionDate(value: string, language: string) {
	return new Intl.DateTimeFormat(language, {
		dateStyle: "long",
		timeStyle: "short",
	}).format(new Date(value));
}
