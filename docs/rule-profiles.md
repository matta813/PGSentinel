# Rule profiles

Rule profiles are reusable, reviewed sets of analyzer threshold values. Administrators manage them under **Settings → Rule profiles**.

Profiles use JSON with `name`, optional `description`, and `entries`. Each entry contains a supported `ruleId` and numeric `value`. Import rejects unknown rules, duplicate entries, non-finite numbers, unsafe ranges, empty profiles, and profiles with more than 50 entries.

Export produces `pgsentinel-rule-profile/v1` JSON without credentials or server-specific scope. Applying a profile is a separate operation: select global, server, or tag scope, review the conflict preview, and explicitly approve replacement when overrides already exist. Application is transactional, so a partial profile is never activated. Creation, deletion, and application are audited.

Profiles do not suppress findings and do not modify monitored PostgreSQL servers.
