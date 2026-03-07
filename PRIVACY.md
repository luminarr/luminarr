# Privacy Policy

Luminarr does not collect, store, or transmit any personal data or usage information
to any third party controlled by this project.

## Outbound Network Connections

Luminarr makes outbound network connections **only** to services that you explicitly
configure. The complete list:

| Destination | When | Controlled by |
|---|---|---|
| **The Movie Database (TMDB)** | Movie search and metadata fetch | Built-in |
| **Configured indexers** | RSS sync and manual release search | Indexer settings in the UI/API |
| **Configured download clients** | Sending grabs, polling queue status | Download client settings |
| **Configured notification targets** | Webhooks, Discord, email, etc. | Notification settings |

## What Luminarr Never Does

- **No telemetry.** No usage data, events, or analytics are sent anywhere.
- **No crash reporting.** Errors are logged locally only.
- **No update checks.** Luminarr never contacts any server to check for updates.
- **No account or registration required.** There is no Luminarr service to sign up for.

## Credentials and Secrets

- API keys and passwords are stored in your local `config.yaml` file only.
- They are **never** transmitted to any server not directly associated with the
  service they authenticate (e.g., your TMDB key is only sent to TMDB).
- They are **never** written to logs. The codebase uses a `Secret` type that
  renders as `***` in all log output and JSON serialization.

## Outbound Request Logging

Every outbound HTTP request is logged locally with: method, URL (with auth
parameters stripped), and status code. This allows you to audit exactly what the
application communicates externally.

## Local Data

All data — movie records, history, quality profiles, plugin configurations — is
stored in your local database file (`luminarr.db` or your configured Postgres
instance). Luminarr has no access to this data beyond your own machine.

## Third-Party Services

When you configure a third-party service (TMDB, an indexer, a download client),
you are subject to that service's own privacy policy. Luminarr's privacy commitments
apply only to the Luminarr software itself.

---

*This document describes what the Luminarr software does, not a legal agreement.
The source code is the authoritative reference for behaviour.*
