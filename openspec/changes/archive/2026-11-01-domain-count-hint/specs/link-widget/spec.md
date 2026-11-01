# Spec Delta

## ADDED Requirements

### Requirement: Domain hint for a plain URL
With the source "URL", the link widget settings SHALL show the same domain hint under the URL field as the "New link" dialog: the number of the owner's favourite links on the entered URL's domain, linking to `/favourites?domain=<domain>` in a new tab. No hint is shown while the URL is invalid or when no favourite has that domain.

#### Scenario: Plain URL on a known domain
- **WHEN** Ann has two favourites on `nas.example.org`, chooses "URL" and enters `https://nas.example.org/files`
- **THEN** the settings show "2 favourites on nas.example.org"

#### Scenario: No favourite on the domain
- **WHEN** Ann chooses "URL" and enters `https://example.org`, and none of her favourites is on `example.org`
- **THEN** no hint is shown
