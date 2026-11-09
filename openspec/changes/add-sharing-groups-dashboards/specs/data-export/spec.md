# Spec Delta

## MODIFIED Requirements

### Requirement: Data left out of the export
The export SHALL NOT contain passkeys, app tokens (neither their secrets nor their names), sessions, sign-in links or codes, favicons, background jobs, the state of a scheduled account deletion, feed items without a flag of the user, the refresh state of feeds, groups, group memberships, shares, invitations, or any data of other users. A dashboard's `isPublic` value is exported as before.

#### Scenario: Passkeys and app tokens
- **WHEN** Ann has two passkeys and a Karakeep app token and exports her data
- **THEN** the document contains nothing about her passkeys or app tokens

#### Scenario: Favicons
- **WHEN** Ann's links show favicons and she exports with files
- **THEN** the archive contains no favicon files and the document lists no favicons

#### Scenario: Shares and groups
- **WHEN** Ann shared "Work" with Bob, belongs to the group "Family" and exports her data
- **THEN** the document contains nothing about the share or the group, and "Work" is exported with its `isPublic` value
