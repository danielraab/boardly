import { Link } from "@tanstack/react-router";
import { ArrowUpRight } from "lucide-react";
import { Fragment } from "react";
import { useTranslation } from "react-i18next";
import { GithubIcon } from "#/components/GithubIcon";
import { openConsentSettings } from "#/lib/consent";
import { consentSettings } from "#/lib/consent/settings";
import { legalLink, legalPageNames } from "#/lib/legalLinks";
import { githubRepoUrl } from "#/lib/sourceCode";

const linkClass = "inline-flex items-center gap-1 rounded-sm hover:text-ink";

/**
 * The slim footer at the end of every page: links to the configured legal
 * pages, "Cookie settings" when analytics consent is on, and
 * "Open source · GitHub".
 */
export function AppFooter({ className = "" }: { className?: string }) {
	const { t } = useTranslation();
	const newTab = t("common.opensInNewTab");
	const legal = legalPageNames.flatMap((page) => {
		const link = legalLink(page);
		return link ? [{ page, link }] : [];
	});
	const consent = consentSettings() !== null;

	return (
		<footer
			className={`${className} flex flex-wrap items-center justify-between gap-x-6 gap-y-2 border-t border-line py-4 text-sm text-muted`}
		>
			{(legal.length > 0 || consent) && (
				<nav
					aria-label={t("shell.footer.legalLinks")}
					className="flex items-center gap-2"
				>
					{legal.map(({ page, link }, i) => (
						<Fragment key={page}>
							{i > 0 && <span aria-hidden>·</span>}
							{link.kind === "page" ? (
								<Link to={link.href} className={linkClass}>
									{t(`legal.${page}`)}
								</Link>
							) : (
								<a
									href={link.href}
									target="_blank"
									rel="noopener noreferrer"
									aria-label={`${t(`legal.${page}`)} ${newTab}`}
									className={linkClass}
								>
									{t(`legal.${page}`)}
									<ArrowUpRight className="size-3.5" aria-hidden />
								</a>
							)}
						</Fragment>
					))}
					{consent && (
						<>
							{legal.length > 0 && <span aria-hidden>·</span>}
							<button
								type="button"
								onClick={() => void openConsentSettings()}
								className={`${linkClass} cursor-pointer`}
							>
								{t("shell.footer.cookieSettings")}
							</button>
						</>
					)}
				</nav>
			)}
			<p className="flex items-center gap-2 sm:ml-auto">
				{t("shell.footer.openSource")}
				<span aria-hidden>·</span>
				<a
					href={githubRepoUrl}
					target="_blank"
					rel="noopener noreferrer"
					aria-label={`${t("shell.github")} ${newTab}`}
					className={linkClass}
				>
					<GithubIcon className="size-4" />
					{t("shell.github")}
					<ArrowUpRight className="size-3.5" aria-hidden />
				</a>
			</p>
		</footer>
	);
}
