import { Dialog } from "@base-ui/react/dialog";
import { LoaderCircle, LocateFixed, Pencil, X } from "lucide-react";
import {
	type ComponentProps,
	type ComponentType,
	type FormEvent,
	useEffect,
	useMemo,
	useRef,
	useState,
} from "react";
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
import {
	formatCoordinate,
	formatPoint,
	isValidPoint,
	type LocationValue,
	maxLocationLabel,
	roundCoordinate,
} from "../document/types";
import { locationName } from "./LocationChip";
import type LocationMapComponent from "./LocationMap";

type MapProps = Omit<ComponentProps<typeof LocationMapComponent>, "onError">;

/**
 * The map of a location, loaded on first use. While it loads it shows "Loading
 * map…", and when it can't be shown, a message instead.
 */
function LazyMap({ className, ...props }: MapProps) {
	const { t } = useTranslation();
	const [LoadedMap, setLoadedMap] = useState<ComponentType<
		ComponentProps<typeof LocationMapComponent>
	> | null>(null);
	const [failed, setFailed] = useState(false);
	useEffect(() => {
		let active = true;
		import("./LocationMap").then(
			(m) => active && setLoadedMap(() => m.default),
			() => active && setFailed(true),
		);
		return () => {
			active = false;
		};
	}, []);
	if (failed)
		return (
			<p
				role="alert"
				className={`flex items-center justify-center rounded-lg border border-line p-4 text-center text-sm text-muted ${className}`}
			>
				{t("notes.locations.mapUnavailable")}
			</p>
		);
	if (!LoadedMap)
		return (
			<p
				className={`flex items-center justify-center gap-2 rounded-lg border border-line text-sm text-muted ${className}`}
			>
				<LoaderCircle className="size-4 animate-spin" aria-hidden />
				{t("notes.locations.loadingMap")}
			</p>
		);
	return (
		<LoadedMap
			{...props}
			className={className}
			onError={() => setFailed(true)}
		/>
	);
}

// Above the note editor modal, like the image viewer.
const backdrop = dialogBackdrop.replace("z-40", "z-50");
const popup = `${dialogPopup.replace("z-40", "z-50").replace("w-[28rem]", "w-[40rem]")} max-h-[calc(100dvh-2rem)] overflow-y-auto`;

/**
 * Shows location on a map, with its label and coordinates. With onEdit, as in
 * the editors, it has an Edit button. Closing it gives the focus back to
 * returnFocus, unless it closes for Edit. location null is closed.
 */
export function LocationMapDialog({
	location,
	returnFocus,
	onClose,
	onEdit,
}: {
	location: LocationValue | null;
	returnFocus?: HTMLElement | null;
	onClose: () => void;
	onEdit?: () => void;
}) {
	const { t } = useTranslation();
	const editing = useRef(false);
	// The last location stays shown while the dialog closes.
	const [shown, setShown] = useState(location);
	if (location && location !== shown) setShown(location);

	return (
		<Dialog.Root
			open={location !== null}
			onOpenChange={(open) => !open && onClose()}
		>
			<Dialog.Portal>
				<Dialog.Backdrop className={backdrop} />
				<Dialog.Popup
					className={popup}
					initialFocus={() => {
						editing.current = false;
						return true;
					}}
					finalFocus={() => (editing.current ? false : (returnFocus ?? true))}
				>
					{shown && (
						<>
							<div className="mb-4 flex items-start gap-2">
								<div className="min-w-0 flex-1">
									<Dialog.Title className="truncate text-lg font-semibold">
										{locationName(shown)}
									</Dialog.Title>
									{shown.label !== undefined && (
										<p className="text-sm text-muted">{formatPoint(shown)}</p>
									)}
								</div>
								{onEdit && (
									<button
										type="button"
										className={secondaryButton}
										onClick={() => {
											editing.current = true;
											onEdit();
										}}
									>
										<Pencil className="size-4" aria-hidden />
										{t("notes.locations.edit")}
									</button>
								)}
								<Dialog.Close
									aria-label={t("common.close")}
									className={`${iconButton} size-9`}
								>
									<X className="size-5" aria-hidden />
								</Dialog.Close>
							</div>
							<LazyMap point={shown} className="h-80 max-h-[60dvh]" />
						</>
					)}
				</Dialog.Popup>
			</Dialog.Portal>
		</Dialog.Root>
	);
}

/** Reads a coordinate typed with "." or "," as the decimal separator. */
function parseCoordinate(text: string): number {
	const t = text.trim();
	if (!/^[+-]?(\d+([.,]\d*)?|[.,]\d+)$/.test(t)) return Number.NaN;
	return Number(t.replace(",", "."));
}

type GeoError = "denied" | "notFound";

/** Asks the device for its position. */
function locate(): Promise<{ lat: number; lng: number }> {
	return new Promise((resolve, reject) => {
		if (!navigator.geolocation) return reject("notFound" satisfies GeoError);
		navigator.geolocation.getCurrentPosition(
			(p) => resolve({ lat: p.coords.latitude, lng: p.coords.longitude }),
			(e) =>
				reject(
					(e.code === e.PERMISSION_DENIED
						? "denied"
						: "notFound") satisfies GeoError,
				),
			{ enableHighAccuracy: true, timeout: 10_000, maximumAge: 60_000 },
		);
	});
}

/** The fields of the location form; mounted anew each time the form opens. */
function LocationForm({
	initial,
	onSave,
	onRemove,
	onCancel,
}: {
	initial: LocationValue | null;
	onSave: (location: LocationValue) => void;
	onRemove?: () => void;
	onCancel: () => void;
}) {
	const { t } = useTranslation();
	const [latText, setLatText] = useState(
		initial ? formatCoordinate(initial.lat) : "",
	);
	const [lngText, setLngText] = useState(
		initial ? formatCoordinate(initial.lng) : "",
	);
	const [label, setLabel] = useState(initial?.label ?? "");
	const [locating, setLocating] = useState(false);
	const [geoError, setGeoError] = useState<GeoError | null>(null);

	const lat = parseCoordinate(latText);
	const lng = parseCoordinate(lngText);
	const latInvalid = latText.trim() !== "" && !(lat >= -90 && lat <= 90);
	const lngInvalid = lngText.trim() !== "" && !(lng >= -180 && lng <= 180);
	const valid = isValidPoint(lat, lng);
	const point = useMemo(
		() =>
			valid ? { lat: roundCoordinate(lat), lng: roundCoordinate(lng) } : null,
		[valid, lat, lng],
	);

	const setPoint = (p: { lat: number; lng: number }) => {
		setLatText(formatCoordinate(p.lat));
		setLngText(formatCoordinate(p.lng));
	};

	async function locateDevice() {
		setLocating(true);
		setGeoError(null);
		try {
			setPoint(await locate());
		} catch (e) {
			setGeoError(e === "denied" ? "denied" : "notFound");
		} finally {
			setLocating(false);
		}
	}

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		if (!point) return;
		const trimmed = label.trim();
		onSave(trimmed ? { ...point, label: trimmed } : point);
	}

	return (
		<form onSubmit={onSubmit} noValidate>
			<Dialog.Title className="mb-4 text-lg font-semibold">
				{t(
					initial
						? "notes.locations.form.editTitle"
						: "notes.locations.form.addTitle",
				)}
			</Dialog.Title>
			<div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
				<label className="block text-sm font-medium">
					{t("notes.locations.form.latitude")}
					<input
						inputMode="decimal"
						autoComplete="off"
						value={latText}
						onChange={(e) => setLatText(e.target.value)}
						aria-invalid={latInvalid}
						aria-describedby={latInvalid ? "location-lat-error" : undefined}
						className={textInput}
					/>
				</label>
				<label className="block text-sm font-medium">
					{t("notes.locations.form.longitude")}
					<input
						inputMode="decimal"
						autoComplete="off"
						value={lngText}
						onChange={(e) => setLngText(e.target.value)}
						aria-invalid={lngInvalid}
						aria-describedby={lngInvalid ? "location-lng-error" : undefined}
						className={textInput}
					/>
				</label>
			</div>
			{latInvalid && (
				<p id="location-lat-error" className="mt-2 text-sm text-danger">
					{t("notes.locations.form.invalidLatitude")}
				</p>
			)}
			{lngInvalid && (
				<p id="location-lng-error" className="mt-2 text-sm text-danger">
					{t("notes.locations.form.invalidLongitude")}
				</p>
			)}
			<div className="mt-3 flex flex-wrap items-center gap-3">
				<button
					type="button"
					className={secondaryButton}
					disabled={locating}
					onClick={locateDevice}
				>
					{locating ? (
						<LoaderCircle className="size-4 animate-spin" aria-hidden />
					) : (
						<LocateFixed className="size-4" aria-hidden />
					)}
					{t(
						locating
							? "notes.locations.form.locating"
							: "notes.locations.form.useMyLocation",
					)}
				</button>
				{geoError && (
					<p role="alert" className="text-sm text-danger">
						{t(`notes.locations.form.${geoError}`)}
					</p>
				)}
			</div>
			<p className="mt-3 mb-1.5 text-sm text-muted">
				{t("notes.locations.form.mapHint")}
			</p>
			<LazyMap point={point} onPick={setPoint} className="h-64 max-h-[40dvh]" />
			<label className="mt-3 block text-sm font-medium">
				{t("notes.locations.form.label")}{" "}
				<span className="font-normal text-muted">
					({t("notes.locations.form.optional")})
				</span>
				<input
					value={label}
					maxLength={maxLocationLabel}
					onChange={(e) => setLabel(e.target.value)}
					className={textInput}
				/>
			</label>
			<div className="mt-5 flex flex-wrap justify-end gap-2">
				{onRemove && (
					<button
						type="button"
						className={`${dangerButton} mr-auto`}
						onClick={onRemove}
					>
						{t("notes.locations.form.remove")}
					</button>
				)}
				<button type="button" className={secondaryButton} onClick={onCancel}>
					{t("notes.locations.form.cancel")}
				</button>
				<button type="submit" className={primaryButton} disabled={!point}>
					{t("notes.locations.form.save")}
				</button>
			</div>
		</form>
	);
}

/**
 * The form to add a location (initial null) or edit one (with onRemove).
 * session changes each time it opens, which starts it anew. Closing it gives
 * the focus back to returnFocus, or to what had it before.
 */
export function LocationFormDialog({
	open,
	session,
	initial,
	returnFocus,
	onSave,
	onRemove,
	onClose,
}: {
	open: boolean;
	session: number;
	initial: LocationValue | null;
	returnFocus?: HTMLElement | null;
	onSave: (location: LocationValue) => void;
	onRemove?: () => void;
	onClose: () => void;
}) {
	return (
		<Dialog.Root open={open} onOpenChange={(next) => !next && onClose()}>
			<Dialog.Portal>
				<Dialog.Backdrop className={backdrop} />
				<Dialog.Popup
					className={popup}
					finalFocus={() => (returnFocus?.isConnected ? returnFocus : true)}
				>
					<LocationForm
						key={session}
						initial={initial}
						onSave={onSave}
						onRemove={onRemove}
						onCancel={onClose}
					/>
				</Dialog.Popup>
			</Dialog.Portal>
		</Dialog.Root>
	);
}
