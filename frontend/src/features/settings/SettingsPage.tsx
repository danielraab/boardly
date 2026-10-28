import { Tabs } from "@base-ui/react/tabs";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Navigate } from "@tanstack/react-router";
import { type FormEvent, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { primaryButton } from "#/components/buttonStyles";
import { narrowPage } from "#/components/pageWidths";
import { meQueryKey, type User, useMe } from "#/features/auth/session";
import { dashboardViewQueryKey } from "#/features/dashboards/api";
import { ConfirmSlugChangeDialog } from "#/features/dashboards/ConfirmSlugChangeDialog";
import { SlugField, type SlugProblem } from "#/features/dashboards/SlugField";
import { slugError } from "#/features/dashboards/slug";
import { favouritesQueryOptions } from "#/features/favourites/api";
import { FavouritePicker } from "#/features/favourites/FavouritePicker";
import { CategoryPicker } from "#/features/notes/categories/CategoryPicker";
import { ApiError, api } from "#/lib/api";
import { usePageTitle } from "#/lib/usePageTitle";
import { ConnectedAppsTab } from "./ConnectedAppsTab";
import { ImportExportTab } from "./ImportExportTab";
import { PasskeysTab } from "./PasskeysTab";

export type SettingsTab = "profile" | "passkeys" | "apps" | "data";

const tabClass =
	"shrink-0 whitespace-nowrap border-b-2 border-transparent px-3 py-2 font-medium text-muted hover:text-ink data-[active]:border-ink data-[active]:text-ink";

export function SettingsPage({
	tab,
	onTabChange,
}: {
	tab: SettingsTab;
	onTabChange: (tab: SettingsTab) => void;
}) {
	const { t } = useTranslation();
	usePageTitle(t("settings.title"));
	const { data: me } = useMe();
	const tabList = useRef<HTMLDivElement>(null);
	// On narrow screens the tab bar scrolls; keep the selected tab visible.
	// biome-ignore lint/correctness/useExhaustiveDependencies: rerun when the tab or the rendered list changes
	useEffect(() => {
		tabList.current
			?.querySelector<HTMLElement>('[aria-selected="true"]')
			?.scrollIntoView?.({ block: "nearest", inline: "nearest" });
	}, [tab, me]);
	// The session can end while the page is open (401 from any request).
	if (me === null) return <Navigate to="/login" />;
	if (!me) return null;
	return (
		<div className={`${narrowPage} pt-12`}>
			<h1 className="text-3xl font-semibold tracking-tight">
				{t("settings.title")}
			</h1>
			<Tabs.Root
				className="mt-8"
				value={tab}
				onValueChange={(value) => onTabChange(value as SettingsTab)}
			>
				<Tabs.List
					ref={tabList}
					className="flex gap-1 overflow-x-auto shadow-[inset_0_-1px_0_var(--color-line)] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
					aria-label={t("settings.tabs.label")}
				>
					<Tabs.Tab value="profile" className={tabClass}>
						{t("settings.tabs.profile")}
					</Tabs.Tab>
					<Tabs.Tab value="passkeys" className={tabClass}>
						{t("settings.tabs.passkeys")}
					</Tabs.Tab>
					<Tabs.Tab value="apps" className={tabClass}>
						{t("settings.tabs.apps")}
					</Tabs.Tab>
					<Tabs.Tab value="data" className={tabClass}>
						{t("settings.tabs.data")}
					</Tabs.Tab>
				</Tabs.List>
				<Tabs.Panel value="profile" className="pt-6">
					<ProfileTab user={me} />
				</Tabs.Panel>
				<Tabs.Panel value="passkeys" className="pt-6">
					<PasskeysTab user={me} />
				</Tabs.Panel>
				<Tabs.Panel value="apps" className="pt-6">
					<ConnectedAppsTab />
				</Tabs.Panel>
				<Tabs.Panel value="data" className="pt-6">
					<ImportExportTab />
				</Tabs.Panel>
			</Tabs.Root>
		</div>
	);
}

function ProfileTab({ user }: { user: User }) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const [name, setName] = useState(user.name ?? "");

	const save = useMutation({
		mutationFn: (value: string) =>
			api<User>("/api/me", {
				method: "PATCH",
				json: { name: value.trim() === "" ? null : value },
			}),
		onSuccess: (updated) => {
			queryClient.setQueryData(meQueryKey, updated);
			setName(updated.name ?? "");
		},
	});

	const unchanged = name.trim() === (user.name ?? "");

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		save.mutate(name);
	}

	const error =
		save.error instanceof ApiError && save.error.code === "invalid_name"
			? t("profile.nameTooLong")
			: save.error
				? t("common.genericError")
				: null;

	return (
		<section>
			<div className="rounded-xl border border-line bg-surface p-6">
				<dl>
					<dt className="text-sm font-medium text-muted">
						{t("profile.email")}
					</dt>
					<dd className="mt-1 break-all">{user.email}</dd>
				</dl>
				<form className="mt-6" onSubmit={onSubmit}>
					<label htmlFor="profile-name" className="block text-sm font-medium">
						{t("profile.nameLabel")}
					</label>
					<div className="mt-2 flex max-w-xl flex-col gap-3 sm:flex-row">
						<input
							id="profile-name"
							name="name"
							autoComplete="name"
							placeholder={t("profile.namePlaceholder")}
							aria-describedby="profile-name-hint"
							value={name}
							onChange={(e) => {
								setName(e.target.value);
								save.reset();
							}}
							className="min-w-0 flex-1 rounded-lg border border-line bg-paper/40 px-4 py-2.5 text-ink outline-none focus-visible:border-ink"
						/>
						<button
							type="submit"
							className={primaryButton}
							disabled={unchanged || save.isPending}
						>
							{save.isPending ? t("profile.saving") : t("profile.save")}
						</button>
					</div>
					<p id="profile-name-hint" className="mt-2 text-sm text-muted">
						{t("profile.nameHint")}
					</p>
					{error && (
						<p role="alert" className="mt-2 text-sm text-danger">
							{error}
						</p>
					)}
					{save.isSuccess && unchanged && (
						<output className="mt-2 block text-sm text-muted">
							{t("profile.saved")}
						</output>
					)}
				</form>
				<SlugForm user={user} />
				<ShareFolderForm user={user} />
				<NoteCategoryForm user={user} />
			</div>
		</section>
	);
}

/** The folder the "Save link" page starts in; the root is saved as null. */
function ShareFolderForm({ user }: { user: User }) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const root = useQuery(favouritesQueryOptions).data;
	const [chosen, setChosen] = useState<string | null>(null);

	const save = useMutation({
		mutationFn: (id: string) =>
			api<User>("/api/me", {
				method: "PATCH",
				json: { shareFolderId: id === root?.id ? null : id },
			}),
		onSuccess: (updated) => {
			queryClient.setQueryData(meQueryKey, updated);
			setChosen(null);
		},
	});

	const saved = user.shareFolderId ?? root?.id ?? "";
	const folder = chosen ?? saved;
	const unchanged = folder === saved;

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		save.mutate(folder);
	}

	return (
		<form className="mt-6 max-w-xl" onSubmit={onSubmit}>
			<FavouritePicker
				id="profile-share-folder"
				label={t("profile.shareFolderLabel")}
				mode="folder"
				value={folder}
				onChange={(id) => {
					setChosen(id);
					save.reset();
				}}
			/>
			<p className="mt-2 text-sm text-muted">{t("profile.shareFolderHint")}</p>
			<button
				type="submit"
				className={`${primaryButton} mt-3`}
				aria-label={t("profile.saveShareFolder")}
				disabled={!root || unchanged || save.isPending}
			>
				{save.isPending ? t("profile.saving") : t("profile.save")}
			</button>
			{save.error && (
				<p role="alert" className="mt-2 text-sm text-danger">
					{save.error instanceof ApiError &&
					save.error.code === "invalid_folder"
						? t("profile.shareFolderInvalid")
						: t("common.genericError")}
				</p>
			)}
			{save.isSuccess && unchanged && (
				<output className="mt-2 block text-sm text-muted">
					{t("profile.saved")}
				</output>
			)}
		</form>
	);
}

/** The category notes from the share page start in; null for none. */
function NoteCategoryForm({ user }: { user: User }) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const [chosen, setChosen] = useState<{ id: string | null } | null>(null);

	const save = useMutation({
		mutationFn: (id: string | null) =>
			api<User>("/api/me", {
				method: "PATCH",
				json: { noteCategoryId: id },
			}),
		onSuccess: (updated) => {
			queryClient.setQueryData(meQueryKey, updated);
			setChosen(null);
		},
	});

	const saved = user.noteCategoryId;
	const category = chosen ? chosen.id : saved;
	const unchanged = category === saved;

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		save.mutate(category);
	}

	return (
		<form className="mt-6 max-w-xl" onSubmit={onSubmit}>
			<CategoryPicker
				id="profile-note-category"
				label={t("profile.noteCategoryLabel")}
				emptyLabel={t("profile.noteCategoryNone")}
				value={category}
				onChange={(id) => {
					setChosen({ id });
					save.reset();
				}}
			/>
			<p className="mt-2 text-sm text-muted">{t("profile.noteCategoryHint")}</p>
			<button
				type="submit"
				className={`${primaryButton} mt-3`}
				aria-label={t("profile.saveNoteCategory")}
				disabled={unchanged || save.isPending}
			>
				{save.isPending ? t("profile.saving") : t("profile.save")}
			</button>
			{save.error && (
				<p role="alert" className="mt-2 text-sm text-danger">
					{save.error instanceof ApiError &&
					save.error.code === "invalid_category"
						? t("profile.noteCategoryInvalid")
						: t("common.genericError")}
				</p>
			)}
			{save.isSuccess && unchanged && (
				<output className="mt-2 block text-sm text-muted">
					{t("profile.saved")}
				</output>
			)}
		</form>
	);
}

function SlugForm({ user }: { user: User }) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const [slug, setSlug] = useState(user.slug);
	const [confirming, setConfirming] = useState(false);

	const save = useMutation({
		mutationFn: (value: string) =>
			api<User>("/api/me", { method: "PATCH", json: { slug: value } }),
		onSuccess: (updated) => {
			queryClient.setQueryData(meQueryKey, updated);
			void queryClient.invalidateQueries({ queryKey: dashboardViewQueryKey });
			setSlug(updated.slug);
		},
		onSettled: () => setConfirming(false),
	});

	const code = save.error instanceof ApiError ? save.error.code : null;
	const serverProblem: SlugProblem | null =
		code === "slug_taken"
			? "taken"
			: code === "slug_reserved"
				? "reserved"
				: code === "invalid_slug"
					? "invalid"
					: null;
	const unchanged = slug === user.slug;

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		if (!unchanged && !slugError(slug, { reserved: true })) setConfirming(true);
	}

	return (
		<form className="mt-6" onSubmit={onSubmit}>
			<div className="flex max-w-xl flex-col gap-3 sm:flex-row sm:items-start">
				<div className="min-w-0 flex-1">
					<SlugField
						id="profile-slug"
						label={t("profile.slugLabel")}
						prefix="/"
						value={slug}
						onChange={(value) => {
							setSlug(value);
							save.reset();
						}}
						saved={user.slug}
						reserved
						serverProblem={serverProblem}
					/>
				</div>
				<button
					type="submit"
					className={`${primaryButton} sm:mt-7`}
					aria-label={t("profile.saveSlug")}
					disabled={
						unchanged ||
						save.isPending ||
						slugError(slug, { reserved: true }) !== null
					}
				>
					{save.isPending ? t("profile.saving") : t("profile.save")}
				</button>
			</div>
			<p className="mt-2 text-sm text-muted">
				{t("profile.slugHint", { slug: user.slug })}
			</p>
			{save.error && !serverProblem && (
				<p role="alert" className="mt-2 text-sm text-red-700">
					{t("common.genericError")}
				</p>
			)}
			{save.isSuccess && unchanged && (
				<output className="mt-2 block text-sm text-muted">
					{t("profile.saved")}
				</output>
			)}
			<ConfirmSlugChangeDialog
				change={
					confirming ? { from: `/${user.slug}/…`, to: `/${slug}/…` } : null
				}
				onConfirm={() => save.mutate(slug)}
				onCancel={() => setConfirming(false)}
			/>
		</form>
	);
}
