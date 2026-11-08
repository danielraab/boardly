# Proposal

## Why

A feed widget mixes news and podcast episodes when a user has both kinds of feed. People who want a dashboard list of things to listen to currently can't get one without picking only podcast feeds, and that still doesn't help for feeds that mix articles and episodes.

## What Changes

- The `feed` widget gets the setting "Only items with audio" / "Nur Einträge mit Audio", off by default.
- When it is on, the widget shows only items that have audio. The number of items applies after this filter, so a widget still fills up when audio items are older than the latest articles.
- It works with every "Show" value and for visitors of public dashboards. Existing widgets behave as if it were off.

## Capabilities

### New Capabilities

### Modified Capabilities

- `feed-widget`: adds the setting and filters the items the widget shows.

## Impact

- **Backend**: `feeds.ItemQuery` gets a filter for items with audio, and the `feed` kind in `internal/widgets` gets the `audioOnly` config field.
- **Frontend**: the settings form and the edit-mode preview of the `feed` widget, and translations (en and de).
- **API docs**: no route or documented schema changes, because a widget's `config` is documented as a plain object.
- **Export and import**: the format version stays at 2. A widget's config is exported and imported as stored, so `audioOnly` travels with it and a test covers that. A config without the field still imports and means off.
