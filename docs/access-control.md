# Access control

PGSentinel uses three deliberately small local roles. It does not depend on an external identity provider.

| Capability | Administrator | Operator | Viewer |
|---|---:|---:|---:|
| Inspect health, findings, incidents, metrics, and evidence | Yes | Yes | Yes |
| Acknowledge and reopen findings | Yes | Yes | No |
| Manage targets, notifications, routing, and operator controls | Yes | No | No |
| Inspect the audit log | Yes | No | No |
| Manage users and roles | Yes | No | No |

The account created from `PGSENTINEL_ADMIN_PASSWORD` during first startup is migrated to `administrator`. New accounts receive an administrator-selected initial password of at least 12 characters and must replace it on first login. Passwords use the existing Argon2id storage path and are never returned by the API.

Role checks occur in the Go API before handlers execute; hidden navigation is only a usability aid. Changing another user's role invalidates all of that user's in-memory sessions immediately. An administrator cannot demote the account backing their current session, which prevents removing the last active administrative path accidentally.

User listing and storage are bounded to 100 local accounts, enforced by both the API and SQLite. PGSentinel intentionally does not add groups, custom permissions, SSO, API tokens, or enterprise IAM policy in this release.

Migration `010_user_roles.sql` adds a constrained role column with `administrator` as the upgrade default. Downgrading the binary leaves the additive column unused; take the normal SQLite backup before attempting any manual schema rollback.
