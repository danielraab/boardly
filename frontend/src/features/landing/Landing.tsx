import { Link } from "@tanstack/react-router";
import { ArrowUpRight } from "lucide-react";
import { useTranslation } from "react-i18next";
import { primaryButton, secondaryButton } from "#/components/buttonStyles";
import { GithubIcon } from "#/components/GithubIcon";
import { widePage } from "#/components/pageWidths";
import { appTitle } from "#/lib/appTitle";
import { githubRepoUrl } from "#/lib/sourceCode";
import { features } from "./features";

const tile = "rounded-md border border-line bg-paper/70";
const accentTile = "rounded-md border border-accent/60 bg-accent/25";

// Tiles on the 12-column grid, sketching a dashboard until a real screenshot exists.
// Three rows on every width; from sm the first tile spans two rows.
const sketchTiles = [
	`${tile} col-span-12 sm:col-span-8 sm:row-span-2`,
	`${tile} col-span-6 sm:col-span-4`,
	`${tile} col-span-6 sm:col-span-4`,
	`${tile} col-span-4 sm:col-span-3`,
	`${tile} col-span-4 sm:col-span-3`,
	`${accentTile} col-span-4 sm:col-span-6`,
];

export function Landing() {
	const { t } = useTranslation();

	return (
		<div className={widePage}>
			<section className="pt-12 pb-10 sm:pt-20">
				<h1 className="text-[clamp(3rem,11vw,7.5rem)] leading-[0.9] font-bold tracking-[-0.04em]">
					{appTitle()}
				</h1>
				<p className="mt-5 max-w-[34ch] text-xl text-muted sm:text-2xl">
					{t("landing.hero.tagline")}
				</p>
				<div className="mt-8 flex flex-wrap gap-3">
					<Link to="/login" className={primaryButton}>
						{t("landing.hero.cta")}
					</Link>
					<a
						href={githubRepoUrl}
						target="_blank"
						rel="noopener noreferrer"
						aria-label={`${t("landing.github")} ${t("common.opensInNewTab")}`}
						className={secondaryButton}
					>
						<GithubIcon className="size-4" />
						{t("landing.github")}
						<ArrowUpRight className="size-4 text-muted" aria-hidden />
					</a>
				</div>
			</section>

			<figure className="relative aspect-video overflow-hidden rounded-xl border border-dashed border-line bg-surface p-3 sm:p-5">
				<div
					aria-hidden
					className="grid h-full grid-cols-12 grid-rows-3 gap-2 sm:gap-3"
				>
					{sketchTiles.map((className) => (
						<div key={className} className={className} />
					))}
				</div>
				<figcaption className="absolute inset-0 flex items-center justify-center">
					<span className="rounded-full bg-surface px-4 py-1.5 text-sm text-muted shadow-sm">
						{t("landing.screenshot.placeholder")}
					</span>
				</figcaption>
			</figure>

			<section className="mt-16" aria-labelledby="features-title">
				<h2
					id="features-title"
					className="text-2xl font-semibold tracking-tight"
				>
					{t("landing.featuresTitle", { appName: appTitle() })}
				</h2>
				<ul className="mt-6 grid gap-8 sm:grid-cols-2 lg:grid-cols-3">
					{features.map(({ id, icon: Icon }) => (
						<li key={id} className="border-t border-line pt-5">
							<Icon
								className="size-6 text-ink"
								strokeWidth={1.75}
								aria-hidden
							/>
							<h3 className="mt-3 text-lg font-semibold">
								{t(`features.${id}.title`)}
							</h3>
							<p className="mt-1 max-w-[45ch] text-muted">
								{t(`features.${id}.description`)}
							</p>
						</li>
					))}
				</ul>
			</section>

			<footer className="mt-16 flex items-center gap-2 border-t border-line pt-5 text-sm text-muted">
				{t("landing.footer.openSource")}
				<span aria-hidden>·</span>
				<a
					href={githubRepoUrl}
					target="_blank"
					rel="noopener noreferrer"
					aria-label={`${t("landing.github")} ${t("common.opensInNewTab")}`}
					className="inline-flex items-center gap-1 rounded-sm hover:text-ink"
				>
					<GithubIcon className="size-4" />
					{t("landing.github")}
					<ArrowUpRight className="size-3.5" aria-hidden />
				</a>
			</footer>
		</div>
	);
}
