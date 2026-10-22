# Spec Delta

## ADDED Requirements

### Requirement: Delete folders used by connected apps
A delete SHALL revoke every active app token whose root folder is removed by it: the deleted folder itself, and with "delete contents" also every folder below it. With "move contents to the parent", only tokens whose root is the deleted folder itself are revoked; tokens rooted in moved folders keep working. Before such a delete, the confirmation dialog SHALL say how many connected apps use the removed folders and that they will be revoked. Revoked tokens stay in the user's token list, and their root is shown as "Deleted folder" / "Gelöschter Ordner". Expired and already revoked tokens are not counted in the warning.

#### Scenario: Warning in the dialog
- **WHEN** the folder "Reading" is the root of two active tokens and the user opens its delete dialog
- **THEN** the dialog says that 2 connected apps use this folder and will be revoked

#### Scenario: Delete revokes
- **WHEN** the user deletes "Reading" and confirms
- **THEN** both tokens are revoked, requests with them are rejected, and the "Connected apps" tab lists them as revoked with the root "Deleted folder"

#### Scenario: Token rooted in a subfolder, contents deleted
- **WHEN** a token's root is "Work / Tools" and the user deletes "Work" with its contents
- **THEN** the warning counts that token and the token is revoked

#### Scenario: Token rooted in a subfolder, contents moved
- **WHEN** a token's root is "Work / Tools" and the user deletes "Work" and keeps its contents
- **THEN** the token stays active and its root is now "Tools" in the parent of "Work"

#### Scenario: No tokens affected
- **WHEN** no active token uses the folder being deleted or any folder that the delete removes
- **THEN** the dialog shows no connected-apps warning
