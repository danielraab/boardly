# Spec Delta

## MODIFIED Requirements

### Requirement: Mini player bar
While a stream is loaded, the app shell SHALL show a bar fixed to the bottom of the viewport with the station's icon (or a radio icon), its title, a play/pause button ("Play" / "Abspielen", "Pause" / "Pause"), a volume control ("Volume" / "Lautstärke") with a mute button ("Mute" / "Stumm", "Unmute" / "Ton an"), and a close button ("Close player" / "Player schließen") that stops playback and hides the bar. When no stream is loaded, the bar is not shown. While the bar is shown, the page content and the global footer (see the `app-shell` capability) SHALL get enough space at the bottom that the bar covers none of them. On viewports narrower than 480 px, the bar SHALL leave out the volume control and keep the icon, title, play/pause and close. All controls work with the keyboard and have accessible names.

#### Scenario: Bar while playing
- **WHEN** Ann plays "FM4"
- **THEN** the bar at the bottom shows the FM4 icon, "FM4", a pause button, the volume control and the close button

#### Scenario: Pause and resume
- **WHEN** Ann activates "Pause" and then "Play"
- **THEN** the stream stops and then plays again from the live position

#### Scenario: Close
- **WHEN** Ann activates "Close player"
- **THEN** playback stops and the bar disappears

#### Scenario: Content not covered
- **WHEN** the bar is shown and Ann scrolls to the end of a long dashboard
- **THEN** the last widget is fully visible above the bar

#### Scenario: Footer not covered
- **WHEN** the bar is shown and Ann scrolls to the end of a page
- **THEN** the global footer with its links is fully visible above the bar
