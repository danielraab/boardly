import { Tabs } from "@base-ui/react/tabs";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Navigate } from "@tanstack/react-router";
import { type FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import { primaryButton } from "#/features/auth/LoginPage";
import { meQueryKey, type User, useMe } from "#/features/auth/session";
import { dashboardViewQueryKey } from "#/features/dashboards/api";
import { ConfirmSlugChangeDialog } from "#/features/dashboards/ConfirmSlugChangeDialog";
import { SlugField, type SlugProblem } from "#/features/dashboards/SlugField";
import { slugError } from "#/features/dashboards/slug";
import { ApiError, api } from "#/lib/api";
import { PasskeysTab } from "./PasskeysTab";

export type SettingsTab = "profile" | "passkeys";

const tabClass =
	"-mb-px border-b-2 border-transparent px-3 py-2 font-medium text-muted hover:text-ink data-[active]:border-ink data-[active]:text-ink";

export function SettingsPage({
	tab,
	onTabChange,
}: {
	tab: SettingsTab;
	onTabChange: (tab: SettingsTab) => void;
}) {
	const { t } = useTranslation();
	const { data: me } = useMe();
	// The session can end while the page is open (401 from any request).
	if (me === null) return <Navigate to="/login" />;
	if (!me) return null;
	return (
		<div className="pt-12">
			<h1 className="text-3xl font-semibold tracking-tight">
				{t("settings.title")}
			</h1>
			<Tabs.Root
				className="mt-8"
				value={tab}
				onValueChange={(value) => onTabChange(value as SettingsTab)}
			>
				<Tabs.List
					className="flex gap-1 border-b border-line"
					aria-label={t("settings.tabs.label")}
				>
					<Tabs.Tab value="profile" className={tabClass}>
						{t("settings.tabs.profile")}
					</Tabs.Tab>
					<Tabs.Tab value="passkeys" className={tabClass}>
						{t("settings.tabs.passkeys")}
					</Tabs.Tab>
				</Tabs.List>
				<Tabs.Panel value="profile" className="pt-6">
					<ProfileTab user={me} />
				</Tabs.Panel>
				<Tabs.Panel value="passkeys" className="pt-6">
					<PasskeysTab user={me} />
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
							className={`${primaryButton} py-2.5`}
							disabled={unchanged || save.isPending}
						>
							{save.isPending ? t("profile.saving") : t("profile.save")}
						</button>
					</div>
					<p id="profile-name-hint" className="mt-2 text-sm text-muted">
						{t("profile.nameHint")}
					</p>
					{error && (
						<p role="alert" className="mt-2 text-sm text-red-700">
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
			</div>
		</section>
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
					className={`${primaryButton} py-2.5 sm:mt-7`}
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
