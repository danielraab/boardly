# Spec Delta

## ADDED Requirements

### Requirement: Registration switch
The server SHALL read from `BOARDLY_ALLOW_REGISTRATION` whether new accounts may be created. The value is `true` or `false`, and an unset or empty value gives the default `true`. Any other value SHALL stop startup with an error message that names `BOARDLY_ALLOW_REGISTRATION`. How sign-in behaves when registration is not allowed is described in the `auth-magic-link` capability.

#### Scenario: Default allows registration
- **WHEN** the server starts without `BOARDLY_ALLOW_REGISTRATION`
- **THEN** registration is allowed

#### Scenario: Registration disabled
- **WHEN** the server starts with `BOARDLY_ALLOW_REGISTRATION=false`
- **THEN** registration is not allowed

#### Scenario: Invalid value
- **WHEN** the server starts with `BOARDLY_ALLOW_REGISTRATION=maybe`
- **THEN** it exits with a non-zero status and an error message that names `BOARDLY_ALLOW_REGISTRATION`
