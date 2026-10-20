import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

export function NotFound() {
	const { t } = useTranslation();

	return (
		<section className="max-w-prose py-16">
			<h1 className="text-3xl font-semibold tracking-tight">
				{t("notFound.title")}
			</h1>
			<p className="mt-3 text-muted">{t("notFound.text")}</p>
			<Link
				to="/"
				className="mt-6 inline-block font-medium underline underline-offset-4"
			>
				{t("notFound.backHome")}
			</Link>
		</section>
	);
}
