// Shared Tailwind classes for a page's outer width inside the app shell.

/** The whole shell container (1536 px): dashboards, the notes overview. */
export const fullPage = "w-full";

/** At most 1152 px, centred: lists and overviews. */
export const widePage = "mx-auto w-full max-w-6xl";

/** At most 768 px, centred: forms and single items. */
export const narrowPage = "mx-auto w-full max-w-3xl";

/** At most 448 px, centred: short single-purpose pages like login. */
export const focusPage = "mx-auto w-full max-w-md";
