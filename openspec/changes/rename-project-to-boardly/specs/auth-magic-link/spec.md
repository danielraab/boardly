## MODIFIED Requirements

### Requirement: Development mail fallback
When no SMTP host is configured, the server SHALL write each sign-in link and its one-time code to its log instead of sending email, and SHALL log a warning at startup that mail delivery is disabled.

#### Scenario: No SMTP configured
- **WHEN** the server runs without `BOARDLY_SMTP_HOST` and a link is requested
- **THEN** the full link and the one-time code appear in the server log and the request still responds 202
