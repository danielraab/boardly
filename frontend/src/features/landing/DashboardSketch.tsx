import { Check, ChevronDown, ChevronRight, Folder, Play } from "lucide-react";
import type { CSSProperties, ReactNode } from "react";
import { useTranslation } from "react-i18next";

// A stylised, decorative dashboard: favourites tree, links, a note with todos
// and a media widget. Only theme tokens are used, so it follows every theme.

const panel =
	"min-h-0 overflow-hidden rounded-lg border border-line bg-surface p-2 shadow-[0_6px_18px_-14px_var(--color-shadow)] sm:p-3 lg:p-5 motion-safe:animate-sketch-in";

/** Panels rise in one after another. */
function delay(step: number): CSSProperties {
	return { animationDelay: `${step * 90}ms` };
}

function Bar({ className = "" }: { className?: string }) {
	return (
		<span className={`block h-1.5 rounded-full sm:h-2 lg:h-2.5 ${className}`} />
	);
}

function Favicon({ colour }: { colour: string }) {
	return (
		<span
			className="size-2.5 shrink-0 rounded-[3px] sm:size-3.5 lg:size-5 lg:rounded"
			style={{ background: `var(--color-note-${colour})` }}
		/>
	);
}

function PanelTitle({ className }: { className: string }) {
	return <Bar className={`mb-2 bg-ink/70 sm:mb-3 lg:mb-5 ${className}`} />;
}

function TreeRow({
	depth = 0,
	children,
}: {
	depth?: number;
	children: ReactNode;
}) {
	return (
		<div
			className={`flex items-center gap-1.5 py-0.5 sm:py-1 lg:gap-2.5 lg:py-2 ${
				depth ? "pl-3 lg:pl-7" : ""
			}`}
		>
			{children}
		</div>
	);
}

function FavouritesPanel() {
	const folder = "size-3 shrink-0 text-muted sm:size-3.5 lg:size-5";
	return (
		<div
			className={`${panel} col-span-1 row-span-1 sm:col-span-4 sm:row-span-2`}
			style={delay(0)}
		>
			<PanelTitle className="w-1/2" />
			<TreeRow>
				<ChevronDown className={folder} />
				<Folder className={folder} />
				<Bar className="w-16 bg-ink/45" />
			</TreeRow>
			<TreeRow depth={1}>
				<Favicon colour="blue" />
				<Bar className="w-20 bg-muted/40" />
			</TreeRow>
			<TreeRow depth={1}>
				<Favicon colour="green" />
				<Bar className="w-14 bg-muted/40" />
			</TreeRow>
			<TreeRow depth={1}>
				<Favicon colour="orange" />
				<Bar className="w-16 bg-muted/40" />
			</TreeRow>
			<TreeRow>
				<ChevronRight className={folder} />
				<Folder className={folder} />
				<Bar className="w-12 bg-ink/45" />
			</TreeRow>
			<TreeRow>
				<ChevronDown className={folder} />
				<Folder className={folder} />
				<Bar className="w-20 bg-ink/45" />
			</TreeRow>
			<TreeRow depth={1}>
				<Favicon colour="purple" />
				<Bar className="w-14 bg-muted/40" />
			</TreeRow>
			<TreeRow depth={1}>
				<Favicon colour="pink" />
				<Bar className="w-10 bg-muted/40" />
			</TreeRow>
			<TreeRow>
				<ChevronDown className={folder} />
				<Folder className={folder} />
				<Bar className="w-14 bg-ink/45" />
			</TreeRow>
			<TreeRow depth={1}>
				<Favicon colour="teal" />
				<Bar className="w-16 bg-muted/40" />
			</TreeRow>
			<TreeRow depth={1}>
				<Favicon colour="yellow" />
				<Bar className="w-12 bg-muted/40" />
			</TreeRow>
			<TreeRow>
				<ChevronRight className={folder} />
				<Folder className={folder} />
				<Bar className="w-16 bg-ink/45" />
			</TreeRow>
		</div>
	);
}

const linkTiles = [
	{ colour: "blue", width: "w-3/4", accent: false },
	{ colour: "red", width: "w-1/2", accent: true },
	{ colour: "teal", width: "w-2/3", accent: false },
	{ colour: "yellow", width: "w-1/2", accent: false },
	{ colour: "purple", width: "w-3/5", accent: false },
	{ colour: "green", width: "w-2/3", accent: false },
	{ colour: "orange", width: "w-1/2", accent: false },
	{ colour: "pink", width: "w-3/4", accent: false },
	{ colour: "grey", width: "w-3/5", accent: false },
];

function LinksPanel() {
	return (
		<div className={`${panel} col-span-1 sm:col-span-8`} style={delay(1)}>
			<PanelTitle className="w-1/4" />
			<div className="grid grid-cols-2 gap-1.5 sm:grid-cols-3 sm:gap-2 lg:gap-3">
				{linkTiles.map(({ colour, width, accent }, index) => (
					<div
						key={colour}
						className={`flex items-center gap-1.5 rounded-md border bg-paper px-1.5 py-1 sm:gap-2 sm:px-2 sm:py-1.5 lg:gap-2.5 lg:px-3 lg:py-3 ${
							accent
								? "border-accent/70 motion-safe:animate-sketch-accent"
								: "border-line"
						} ${index >= 4 ? "max-sm:hidden" : ""}`}
					>
						<Favicon colour={colour} />
						<Bar className={`${width} bg-ink/40`} />
					</div>
				))}
			</div>
		</div>
	);
}

function NotePanel() {
	return (
		<div
			className={`${panel} col-span-1 border-transparent sm:col-span-4`}
			style={{ ...delay(2), background: "var(--color-note-yellow)" }}
		>
			<PanelTitle className="w-1/2" />
			{[true, true, false, false, false].map((done, index) => (
				<div
					// biome-ignore lint/suspicious/noArrayIndexKey: static decorative rows
					key={index}
					className={`flex items-center gap-1.5 py-0.5 sm:py-1 lg:gap-2.5 lg:py-2 ${
						index >= 3 ? "max-sm:hidden" : ""
					}`}
				>
					<span className="flex size-2.5 shrink-0 items-center justify-center rounded-[3px] border border-ink/50 sm:size-3.5 lg:size-5 lg:rounded">
						{done && <Check className="size-2 text-ink sm:size-3 lg:size-4" />}
					</span>
					<Bar
						className={`${index % 2 ? "w-3/5" : "w-4/5"} ${done ? "bg-ink/25" : "bg-ink/55"}`}
					/>
				</div>
			))}
		</div>
	);
}

const stations = [
	{ colour: "teal", width: "w-3/4", playing: true },
	{ colour: "pink", width: "w-1/2", playing: false },
	{ colour: "grey", width: "w-2/3", playing: false },
	{ colour: "blue", width: "w-3/5", playing: false },
	{ colour: "orange", width: "w-1/2", playing: false },
];

function MediaPanel() {
	return (
		<div className={`${panel} col-span-1 sm:col-span-4`} style={delay(3)}>
			<PanelTitle className="w-1/3" />
			{stations.map(({ colour, width, playing }, index) => (
				<div
					key={colour}
					className={`flex items-center gap-1.5 rounded-md px-1 py-0.5 sm:py-1 lg:gap-2.5 lg:px-2 lg:py-2 ${
						playing ? "bg-accent/15" : ""
					} ${index >= 3 ? "max-sm:hidden" : ""}`}
				>
					<span
						className="flex size-3 shrink-0 items-center justify-center rounded sm:size-4 lg:size-6"
						style={{ background: `var(--color-note-${colour})` }}
					>
						{playing && (
							<Play className="size-2 fill-ink text-ink sm:size-2.5 lg:size-3" />
						)}
					</span>
					<Bar className={`${width} bg-ink/40`} />
				</div>
			))}
		</div>
	);
}

/** Theme-aware illustration of a Boardly dashboard for the landing page. */
export function DashboardSketch() {
	const { t } = useTranslation();
	return (
		<figure
			aria-label={t("landing.sketch")}
			className="aspect-[4/3] overflow-hidden rounded-xl border border-line bg-paper p-2 sm:aspect-video sm:p-4"
		>
			<div
				aria-hidden
				className="grid h-full grid-cols-2 grid-rows-2 gap-2 sm:grid-cols-12 sm:grid-rows-[2fr_3fr] sm:gap-3 lg:gap-4"
			>
				<FavouritesPanel />
				<LinksPanel />
				<NotePanel />
				<MediaPanel />
			</div>
		</figure>
	);
}
