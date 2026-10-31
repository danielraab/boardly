import { AlertDialog } from "@base-ui/react/alert-dialog";
import { Dialog } from "@base-ui/react/dialog";
import { Menu } from "@base-ui/react/menu";
import {
	closestCenter,
	DndContext,
	type DragEndEvent,
	KeyboardSensor,
	PointerSensor,
	TouchSensor,
	useSensor,
	useSensors,
} from "@dnd-kit/core";
import {
	arrayMove,
	SortableContext,
	sortableKeyboardCoordinates,
	useSortable,
	verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
	EllipsisVertical,
	GripVertical,
	Pause,
	Pencil,
	Play,
	Plus,
	Radio,
	Trash2,
} from "lucide-react";
import { type FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import {
	dangerButton,
	iconButton,
	primaryButton,
	secondaryButton,
} from "#/components/buttonStyles";
import {
	dialogBackdrop,
	dialogPopup,
	textInput,
} from "#/components/dialogStyles";
import { type IconChoice, IconChooser } from "#/components/IconChooser";
import { ChosenIcon } from "#/components/icons/ChosenIcon";
import { menuItem, menuPopup } from "#/components/menuStyles";
import { widePage } from "#/components/pageWidths";
import { iconChoice, iconProps, noteIcon } from "#/features/notes/noteIcon";
import { ApiError } from "#/lib/api";
import { usePageTitle } from "#/lib/usePageTitle";
import { AddStationDialog } from "./AddStationDialog";
import {
	deleteSource,
	hostOf,
	type MediaSource,
	mediaQueryKey,
	mediaQueryOptions,
	reorderSources,
	updateSource,
} from "./api";
import { useMediaPlayer } from "./player";

/** /media: the user's media library, for now internet radio stations. */
export function MediaPage() {
	const { t } = useTranslation();
	usePageTitle(t("media.title"));
	const list = useQuery(mediaQueryOptions);
	const [adding, setAdding] = useState(false);
	const [editing, setEditing] = useState<MediaSource | null>(null);
	const [deleting, setDeleting] = useState<MediaSource | null>(null);
	const sources = list.data ?? [];

	return (
		<section className={`${widePage} pt-12 pb-16`}>
			<div className="flex flex-wrap items-end justify-between gap-4">
				<div>
					<h1 className="text-3xl font-semibold tracking-tight">
						{t("media.title")}
					</h1>
					<p className="mt-2 max-w-[60ch] text-muted">{t("media.intro")}</p>
				</div>
				<button
					type="button"
					onClick={() => setAdding(true)}
					className={primaryButton}
				>
					<Plus className="size-4" aria-hidden />
					{t("media.addStation")}
				</button>
			</div>
			{list.isError && (
				<p role="alert" className="mt-6 text-sm text-danger">
					{t("common.genericError")}
				</p>
			)}
			{list.data && sources.length === 0 && (
				<div className="mt-8 rounded-xl border border-dashed border-line p-10 text-center">
					<Radio className="mx-auto size-8 text-muted" aria-hidden />
					<p className="mt-3 font-medium">{t("media.empty")}</p>
					<p className="mt-1 text-sm text-muted">{t("media.emptyHint")}</p>
				</div>
			)}
			{sources.length > 0 && (
				<StationList
					sources={sources}
					onEdit={setEditing}
					onDelete={setDeleting}
				/>
			)}
			<AddStationDialog
				open={adding}
				onClose={() => setAdding(false)}
				library={sources}
			/>
			<EditStationDialog source={editing} onClose={() => setEditing(null)} />
			<DeleteStationDialog
				source={deleting}
				onClose={() => setDeleting(null)}
			/>
		</section>
	);
}

function StationList({
	sources,
	onEdit,
	onDelete,
}: {
	sources: MediaSource[];
	onEdit: (source: MediaSource) => void;
	onDelete: (source: MediaSource) => void;
}) {
	const queryClient = useQueryClient();
	const sensors = useSensors(
		useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
		useSensor(TouchSensor, {
			activationConstraint: { delay: 200, tolerance: 5 },
		}),
		useSensor(KeyboardSensor, {
			coordinateGetter: sortableKeyboardCoordinates,
		}),
	);
	const reorder = useMutation({
		mutationFn: (ids: string[]) => reorderSources(ids),
		onMutate: (ids) => {
			const byId = new Map(sources.map((s) => [s.id, s]));
			queryClient.setQueryData<MediaSource[]>(
				mediaQueryKey,
				ids.flatMap((id, position) => {
					const s = byId.get(id);
					return s ? [{ ...s, position }] : [];
				}),
			);
		},
		onSettled: () =>
			void queryClient.invalidateQueries({ queryKey: mediaQueryKey }),
	});
	const ids = sources.map((s) => s.id);

	function onDragEnd({ active, over }: DragEndEvent) {
		if (!over || active.id === over.id) return;
		reorder.mutate(
			arrayMove(
				ids,
				ids.indexOf(String(active.id)),
				ids.indexOf(String(over.id)),
			),
		);
	}

	return (
		<DndContext
			sensors={sensors}
			collisionDetection={closestCenter}
			onDragEnd={onDragEnd}
		>
			<SortableContext items={ids} strategy={verticalListSortingStrategy}>
				<ul className="mt-8 divide-y divide-line rounded-xl border border-line bg-surface">
					{sources.map((source) => (
						<StationRow
							key={source.id}
							source={source}
							onEdit={() => onEdit(source)}
							onDelete={() => onDelete(source)}
						/>
					))}
				</ul>
			</SortableContext>
		</DndContext>
	);
}

function StationRow({
	source,
	onEdit,
	onDelete,
}: {
	source: MediaSource;
	onEdit: () => void;
	onDelete: () => void;
}) {
	const { t } = useTranslation();
	const player = useMediaPlayer();
	const {
		attributes,
		listeners,
		setNodeRef,
		setActivatorNodeRef,
		transform,
		transition,
		isDragging,
	} = useSortable({ id: source.id });
	const isCurrent = player.current?.id === source.id;
	const playing =
		isCurrent && (player.status === "playing" || player.status === "loading");

	return (
		<li
			ref={setNodeRef}
			style={{
				transform: transform
					? `translate3d(0, ${Math.round(transform.y)}px, 0)`
					: undefined,
				transition,
			}}
			className={`flex items-center gap-3 bg-surface px-2 py-3 first:rounded-t-xl last:rounded-b-xl sm:px-3 ${isDragging ? "relative z-10 shadow-md" : ""}`}
		>
			<button
				type="button"
				ref={setActivatorNodeRef}
				{...attributes}
				{...listeners}
				aria-label={t("media.drag", { name: source.title })}
				className="flex size-8 shrink-0 cursor-grab touch-none items-center justify-center rounded-md text-muted hover:text-ink focus-visible:ring-2 focus-visible:ring-accent focus-visible:outline-none"
			>
				<GripVertical className="size-4" aria-hidden />
			</button>
			<span className="flex size-10 shrink-0 items-center justify-center overflow-hidden rounded-md border border-line bg-paper">
				<ChosenIcon
					{...iconProps(source.icon)}
					fallback={Radio}
					className="size-6"
				/>
			</span>
			<div className="min-w-0 flex-1">
				<p className="truncate font-medium">{source.title}</p>
				<p className="truncate text-sm text-muted">{hostOf(source.url)}</p>
			</div>
			<button
				type="button"
				onClick={() => (isCurrent ? player.toggle() : player.play(source))}
				aria-label={
					playing
						? t("media.pauseNamed", { name: source.title })
						: t("media.playNamed", { name: source.title })
				}
				aria-pressed={playing}
				className="inline-flex size-10 shrink-0 items-center justify-center rounded-full border border-line text-ink hover:bg-paper aria-pressed:bg-ink aria-pressed:text-paper"
			>
				{playing ? (
					<Pause className="size-5" aria-hidden />
				) : (
					<Play className="size-5" aria-hidden />
				)}
			</button>
			<Menu.Root>
				<Menu.Trigger
					aria-label={t("media.actions", { name: source.title })}
					className={iconButton}
				>
					<EllipsisVertical className="size-4" aria-hidden />
				</Menu.Trigger>
				<Menu.Portal>
					<Menu.Positioner align="end" sideOffset={4} className="z-50">
						<Menu.Popup className={menuPopup}>
							<Menu.Item className={menuItem} onClick={onEdit}>
								<Pencil className="size-4 text-muted" aria-hidden />
								{t("media.edit")}
							</Menu.Item>
							<Menu.Item
								className={`${menuItem} text-danger`}
								onClick={onDelete}
							>
								<Trash2 className="size-4" aria-hidden />
								{t("media.delete")}
							</Menu.Item>
						</Menu.Popup>
					</Menu.Positioner>
				</Menu.Portal>
			</Menu.Root>
		</li>
	);
}

function EditStationDialog({
	source,
	onClose,
}: {
	source: MediaSource | null;
	onClose: () => void;
}) {
	return (
		<Dialog.Root
			open={source !== null}
			onOpenChange={(open) => !open && onClose()}
		>
			<Dialog.Portal>
				<Dialog.Backdrop className={dialogBackdrop} />
				<Dialog.Popup className={dialogPopup}>
					{source && (
						<EditForm key={source.id} source={source} onClose={onClose} />
					)}
				</Dialog.Popup>
			</Dialog.Portal>
		</Dialog.Root>
	);
}

function EditForm({
	source,
	onClose,
}: {
	source: MediaSource;
	onClose: () => void;
}) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const player = useMediaPlayer();
	const [title, setTitle] = useState(source.title);
	const [url, setUrl] = useState(source.url);
	const [icon, setIcon] = useState<IconChoice>(() => iconChoice(source.icon));
	const save = useMutation({
		mutationFn: () =>
			updateSource(source.id, {
				title,
				url: url.trim(),
				icon: noteIcon(icon),
			}),
		onSuccess: (saved) => {
			queryClient.setQueryData<MediaSource[]>(mediaQueryKey, (list) =>
				list?.map((s) => (s.id === saved.id ? saved : s)),
			);
			// The bar shows the new title and icon of what plays.
			player.update(saved);
			onClose();
		},
	});
	const code =
		save.error instanceof ApiError
			? save.error.code
			: save.error
				? "error"
				: null;

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		save.mutate();
	}

	return (
		<form onSubmit={onSubmit} className="flex flex-col gap-4">
			<Dialog.Title className="text-lg font-semibold">
				{t("media.editTitle")}
			</Dialog.Title>
			<div>
				<span className="block text-sm font-medium">
					{t("media.iconLabel")}
				</span>
				<div className="mt-2">
					<IconChooser value={icon} onChange={setIcon} />
				</div>
			</div>
			<div>
				<label
					htmlFor="edit-station-name"
					className="block text-sm font-medium"
				>
					{t("media.add.nameLabel")}
				</label>
				<input
					id="edit-station-name"
					required
					maxLength={200}
					value={title}
					onChange={(e) => setTitle(e.target.value)}
					className={textInput}
				/>
			</div>
			<div>
				<label htmlFor="edit-station-url" className="block text-sm font-medium">
					{t("media.add.urlLabel")}
				</label>
				<input
					id="edit-station-url"
					type="url"
					required
					value={url}
					onChange={(e) => setUrl(e.target.value)}
					aria-describedby="edit-station-url-hint"
					className={textInput}
				/>
				{code ? (
					<p
						id="edit-station-url-hint"
						role="alert"
						className="mt-1 text-sm text-danger"
					>
						{t(
							code === "invalid_title"
								? "media.add.errors.invalidTitle"
								: code === "invalid_url"
									? "media.add.errors.invalidUrl"
									: code === "stream_not_https"
										? "media.add.errors.notHttps"
										: code === "stream_unreachable"
											? "media.add.errors.unreachable"
											: "common.genericError",
						)}
					</p>
				) : (
					<p id="edit-station-url-hint" className="mt-1 text-sm text-muted">
						{t("media.add.urlHint")}
					</p>
				)}
			</div>
			<div className="flex justify-end gap-2">
				<button type="button" onClick={onClose} className={secondaryButton}>
					{t("media.cancel")}
				</button>
				<button
					type="submit"
					disabled={save.isPending}
					className={primaryButton}
				>
					{t("media.save")}
				</button>
			</div>
		</form>
	);
}

function DeleteStationDialog({
	source,
	onClose,
}: {
	source: MediaSource | null;
	onClose: () => void;
}) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const player = useMediaPlayer();
	const remove = useMutation({
		mutationFn: (id: string) => deleteSource(id),
		onSuccess: (_, id) => {
			if (player.current?.id === id) player.stop();
			queryClient.setQueryData<MediaSource[]>(mediaQueryKey, (list) =>
				list?.filter((s) => s.id !== id),
			);
			onClose();
		},
	});

	return (
		<AlertDialog.Root
			open={source !== null}
			onOpenChange={(open) => !open && onClose()}
		>
			<AlertDialog.Portal>
				<AlertDialog.Backdrop className={dialogBackdrop} />
				<AlertDialog.Popup className={dialogPopup}>
					<AlertDialog.Title className="text-lg font-semibold">
						{t("media.deleteTitle")}
					</AlertDialog.Title>
					<AlertDialog.Description className="mt-2 text-muted">
						{t("media.deleteText", { name: source?.title ?? "" })}
					</AlertDialog.Description>
					{remove.isError && (
						<p role="alert" className="mt-3 text-sm text-danger">
							{t("common.genericError")}
						</p>
					)}
					<div className="mt-6 flex justify-end gap-2">
						<AlertDialog.Close className={secondaryButton}>
							{t("media.cancel")}
						</AlertDialog.Close>
						<button
							type="button"
							disabled={remove.isPending}
							onClick={() => source && remove.mutate(source.id)}
							className={dangerButton}
						>
							{t("media.delete")}
						</button>
					</div>
				</AlertDialog.Popup>
			</AlertDialog.Portal>
		</AlertDialog.Root>
	);
}
