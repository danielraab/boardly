// The only module that loads MapLibre; import it with React.lazy so that the
// map code and its styles are only fetched when a map is shown.
import "maplibre-gl/dist/maplibre-gl.css";
import { Map as MapLibreMap, Marker, setWorkerUrl } from "maplibre-gl";
import workerUrl from "maplibre-gl/dist/maplibre-gl-worker.mjs?url";
import { useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import { roundCoordinate } from "../document/types";

// MapLibre looks for its worker next to its own module, which bundling moves.
setWorkerUrl(workerUrl);

/** The style maps are drawn from; the server sets it in index.html. */
function mapStyleUrl(): string {
	return (
		document
			.querySelector<HTMLMetaElement>('meta[name="boardly-map-style"]')
			?.content.trim() || "https://tiles.openfreemap.org/styles/liberty"
	);
}

type Point = { lat: number; lng: number };

/** How close a map with a point is zoomed in: streets and buildings. */
const pointZoom = 15;

const samePoint = (a: Point | null, b: Point | null) =>
	a !== null && b !== null && a.lat === b.lat && a.lng === b.lng;

/**
 * A vector map with a pin at point, or the whole world without one. The map
 * moves to the point whenever it changes, except when it changed because the
 * map was clicked. With onPick, clicking the map picks that point, rounded to
 * 6 decimals. onError is called when the map can't be shown, for example when
 * the style can't be loaded or the browser has no WebGL.
 */
export default function LocationMap({
	point,
	onPick,
	onError,
	className = "",
}: {
	point: Point | null;
	onPick?: (point: Point) => void;
	onError: () => void;
	className?: string;
}) {
	const { t } = useTranslation();
	const container = useRef<HTMLElement>(null);
	const map = useRef<MapLibreMap | null>(null);
	const marker = useRef<Marker | null>(null);
	const picked = useRef<Point | null>(null);
	// The map is made once; the latest callbacks are read when it calls them.
	const callbacks = useRef({ onPick, onError });
	callbacks.current = { onPick, onError };
	const initial = useRef(point);

	useEffect(() => {
		if (!container.current) return;
		const start = initial.current;
		let created: MapLibreMap;
		try {
			created = new MapLibreMap({
				container: container.current,
				style: mapStyleUrl(),
				center: start ? [start.lng, start.lat] : [0, 20],
				zoom: start ? pointZoom : 1,
			});
		} catch {
			callbacks.current.onError();
			return;
		}
		let loaded = false;
		created.on("load", () => {
			loaded = true;
		});
		// Errors after loading are single tiles; the map still works.
		created.on("error", () => {
			if (!loaded) callbacks.current.onError();
		});
		created.on("click", (e) => {
			const pick = callbacks.current.onPick;
			if (!pick) return;
			const next = {
				lat: roundCoordinate(e.lngLat.lat),
				lng: roundCoordinate(e.lngLat.wrap().lng),
			};
			picked.current = next;
			pick(next);
		});
		map.current = created;
		return () => {
			created.remove();
			map.current = null;
			marker.current = null;
		};
	}, []);

	useEffect(() => {
		const m = map.current;
		if (!m) return;
		if (!point) {
			marker.current?.remove();
			marker.current = null;
			return;
		}
		if (!marker.current) {
			const accent = getComputedStyle(document.documentElement)
				.getPropertyValue("--color-accent")
				.trim();
			marker.current = new Marker(accent ? { color: accent } : {});
			marker.current.setLngLat([point.lng, point.lat]).addTo(m);
		} else {
			marker.current.setLngLat([point.lng, point.lat]);
		}
		if (!samePoint(point, picked.current)) {
			m.easeTo({
				center: [point.lng, point.lat],
				zoom: Math.max(m.getZoom(), pointZoom),
			});
		}
		picked.current = null;
	}, [point]);

	return (
		<section
			ref={container}
			aria-label={t("notes.locations.map")}
			className={`overflow-hidden rounded-lg border border-line ${onPick ? "[&_canvas]:cursor-crosshair" : ""} ${className}`}
		/>
	);
}
