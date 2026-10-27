# Spec Delta

## ADDED Requirements

### Requirement: Note colour palette
Every theme SHALL define a background colour for each of the note colours `red` (Red / Rot), `orange` (Orange / Orange), `yellow` (Yellow / Gelb), `green` (Green / Grün), `teal` (Teal / Petrol), `blue` (Blue / Blau), `purple` (Purple / Lila), `pink` (Pink / Rosa) and `grey` (Grey / Grau). In light themes they are light tints, in dark themes dark shades, so that the same note colour fits every theme. In every theme, `ink` and `muted` SHALL have a contrast ratio of at least 4.5:1 and `accent` at least 3:1 against each note colour, and each note colour SHALL be distinguishable from `paper`. Notes and note categories store only the colour's name, never a colour value, so a note's colour follows the active theme.

#### Scenario: Contrast check for note colours
- **WHEN** the contrast ratios of `ink`, `muted` and `accent` against every note colour are computed for each of the 7 themes
- **THEN** every pair meets its minimum

#### Scenario: Colour follows the theme
- **WHEN** a note has the colour `blue` and Ann switches from the theme `white` to `black`
- **THEN** the note's card changes from a light blue to a dark blue background without reload, and its text stays readable
