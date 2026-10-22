# i18n Specification

## Purpose

Makes the Boardly UI available in several languages, English and German for now, and lets each visitor choose and keep their language.

## Requirements

### Requirement: Supported languages
The web app SHALL provide every user-visible text in English (`en`) and German (`de`). English SHALL be the fallback language.

#### Scenario: All texts translated
- **WHEN** the translation catalogs are checked
- **THEN** the `en` and `de` catalogs contain exactly the same set of keys

### Requirement: Initial language detection
On a visitor's first visit, the web app SHALL pick a supported language from the browser's preferred languages. It SHALL fall back to English when none of them is supported.

#### Scenario: German browser
- **WHEN** a first-time visitor's browser prefers `de-AT`
- **THEN** the web app is shown in German

#### Scenario: Unsupported browser language
- **WHEN** a first-time visitor's browser prefers only `fr`
- **THEN** the web app is shown in English

### Requirement: Switching language
The web app SHALL let visitors switch the language at any time, signed in or not. All visible texts update right away, without a page reload.

#### Scenario: Switch to German
- **WHEN** a visitor viewing the English UI picks "Deutsch" in the language switcher
- **THEN** all visible texts change to German without a page reload

### Requirement: Language choice persists
The web app SHALL remember a visitor's explicit language choice in that browser and use it on later visits, in preference to browser detection.

#### Scenario: Return visit
- **WHEN** a visitor who picked German returns in the same browser with an English-preferring browser setting
- **THEN** the web app is shown in German

### Requirement: Document language attribute
The web app SHALL keep the HTML document's `lang` attribute set to the active language.

#### Scenario: Lang attribute follows switch
- **WHEN** the visitor switches to German
- **THEN** the document's `lang` attribute is `de`
