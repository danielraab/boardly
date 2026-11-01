## MODIFIED Requirements

### Requirement: Station search
The "Add station" dialog SHALL have the tabs "Search" / "Suchen" (default) and "Enter URL" / "URL eingeben". The search tab has a search field. After the user stops typing for a moment, at least 2 characters are sent **from the browser directly** to the public radio-browser.info API, without going through Boardly's server. It shows at most 30 results, ordered by popularity. Each result shows the station's logo (or a radio icon), name, country, tags and bitrate, if known. A station without an `https` stream URL SHALL be shown dimmed with "Not playable (no HTTPS stream)" / "Nicht abspielbar (kein HTTPS-Stream)" and can't be added. Choosing a playable result SHALL create the station with its name as title, its stream URL and its logo URL. A result whose station is already in the library, by URL, SHALL show "Added" / "Hinzugefügt" and can't be added again. While searching, the dialog shows a loading state. When the directory can't be reached, it shows "The station directory isn't reachable right now. You can still enter a URL." / "Das Senderverzeichnis ist gerade nicht erreichbar. Du kannst trotzdem eine URL eingeben." Nothing is sent to radio-browser.info until the user types in the search field. Below the `sm` breakpoint, the add button of a result SHALL show only its icon; its accessible name still says "Add <station>" / "<station> hinzufügen".

#### Scenario: Search and add
- **WHEN** Ann types "fm4" and chooses the result "FM4" with an `https` stream
- **THEN** "FM4" is added at the end of her library with its stream URL and logo

#### Scenario: Station without HTTPS
- **WHEN** a result's only stream URL starts with `http://`
- **THEN** it is dimmed, shows "Not playable (no HTTPS stream)", and can't be chosen

#### Scenario: Already added
- **WHEN** Ann already has "FM4" with the same stream URL and searches "fm4"
- **THEN** the result "FM4" shows "Added" and can't be chosen

#### Scenario: Directory down
- **WHEN** radio-browser.info doesn't answer
- **THEN** the dialog shows that the directory isn't reachable and the "Enter URL" tab still works

#### Scenario: No request before typing
- **WHEN** Ann opens the dialog and doesn't type
- **THEN** the browser sends no request to radio-browser.info

#### Scenario: Add button on a phone
- **WHEN** Ann searches "fm4" on a screen narrower than the `sm` breakpoint
- **THEN** the add button of "FM4" shows only a plus icon and is announced as "Add FM4"
