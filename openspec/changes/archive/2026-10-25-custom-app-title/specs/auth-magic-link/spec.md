# Spec Delta

## MODIFIED Requirements

### Requirement: Localized sign-in email
The sign-in email SHALL be written in the requested locale. It SHALL contain the sign-in button and link, then, introduced as an alternative ("or use this one-time code"), the one-time code with a note that it is entered on the sign-in page where the email was requested. It SHALL also contain the 15-minute validity of link and code and a note that the recipient can ignore the email if they didn't request it. Wherever the subject and body name the product (subject, heading, text, button and signature), they SHALL use the configured app title (see "Configurable app title" in the `app-hosting` capability, "Boardly" by default). It SHALL be sent as plain text with an HTML alternative, both with the same content.

#### Scenario: English email
- **WHEN** a link is requested with locale `en`
- **THEN** the email subject and body are English and mention the 15-minute validity

#### Scenario: Both ways to sign in
- **WHEN** a sign-in email is sent
- **THEN** its HTML and plain-text parts both contain the link, followed by the alternative of the one-time code

#### Scenario: Default app title
- **WHEN** the server runs without `BOARDLY_APP_TITLE` and a link is requested with locale `en`
- **THEN** the email subject is "Your Boardly sign-in link"

#### Scenario: Custom app title
- **WHEN** the server runs with `BOARDLY_APP_TITLE="Boardly Staging"` and a link is requested with locale `de`
- **THEN** the email subject is "Dein Anmeldelink für Boardly Staging", and the heading, text, button and signature of both parts name "Boardly Staging" instead of "Boardly"
