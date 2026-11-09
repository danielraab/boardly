## MODIFIED Requirements

### Requirement: Imported media sources
With `media`, every media source of the file SHALL be added after the user's media sources, in the order of the file, with its kind, title, URL and icon. A media source whose URL equals the URL of one of the user's media sources, or of one imported before it, SHALL be skipped and counted as a duplicate. A media source of a kind this server doesn't know is skipped and counted as invalid. A media source of the kind `embed` whose URL no provider accepts (see "Embed links" in the `media-library` capability) is skipped and counted as invalid. The import SHALL NOT contact the provider, and it SHALL NOT take a player address from the file.

#### Scenario: Duplicate station
- **WHEN** Ann has the station `https://radio.example.org/stream` and imports an export with the same URL
- **THEN** no second station is added, and the report counts one duplicate media source

#### Scenario: Embed source
- **WHEN** an export has an `embed` source with the URL `https://open.spotify.com/album/4aawyAB9vmqN3uQ7FjRGTy` and the title "Global Warming"
- **THEN** the source is added with its title and icon, and its player address is built by this server

#### Scenario: Embed with a link no provider accepts
- **WHEN** an export has an `embed` source with the URL `https://example.org/player`
- **THEN** the source isn't added, and the report counts one invalid media source
