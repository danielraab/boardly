import { createFileRoute } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

export const Route = createFileRoute("/login")({ component: Login });

function Login() {
	const { t } = useTranslation();

	return (
		<section className="max-w-prose py-16">
			<h1 className="text-3xl font-semibold tracking-tight">
				{t("login.title")}
			</h1>
			<p className="mt-3 text-muted">{t("login.comingSoon")}</p>
		</section>
	);
}
