# Design

The label is wrapped in a `<span className="max-sm:hidden">`, the same pattern the landing page
uses for mobile-only hiding. The button's padding becomes `px-2` below `sm` so the icon button is
square-ish. The `aria-label` already names the station, so screen readers are unaffected. A CSS
breakpoint is used instead of a JS media query, so there is no layout shift and nothing to test
for resize.
