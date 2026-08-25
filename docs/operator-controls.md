# Operator controls

Operator controls reduce noise during known conditions without rewriting PostgreSQL evidence or finding history. They affect only PGSentinel analysis thresholds, visible suppression state, and notification queueing. They never change a monitored PostgreSQL server.

## Maintenance windows

A window has a start, end, reason, and one or more scopes: server, server tag, finding category, or rule ID. All configured scopes must match. Windows may last at most 30 days and start no more than one year ahead; a global unscoped window is rejected. The API and Settings UI show `upcoming`, `active`, and `expired` states.

During an active matching window, findings remain active with all evidence and lifecycle timestamps. They are visibly marked as suppressed and new lifecycle notifications are not queued. Ending or deleting a window does not manufacture a lifecycle transition.

## Suppressions

An individual finding can be suppressed directly from its problem detail for one hour, four hours, one day, or seven days. Rule suppressions require a server or tag scope, a reason, and an expiry within 30 days. Suppression never acknowledges, resolves, or deletes a finding. Removing a suppression simply restores its normal visible state and future notification behavior.

## Threshold overrides

Only the allowlisted numeric analyzer thresholds shown in Settings can be overridden. Each value has a hard safe range; values outside it are rejected by both the UI and API. This prevents an arbitrary value from effectively disabling the analyzer. Resolution is deterministic:

1. server override;
2. matching tag override (lexically first tag scope if several match);
3. global override;
4. built-in default.

Overrides are evaluated at the start of each analyzer cycle. Existing findings resolve only when a complete collection no longer meets the effective threshold. The reason is retained with every control.

## API and retention

Authenticated endpoints under `/api/v1` expose maintenance windows, suppressions, and threshold overrides. Each control type is capped at 200 records and list responses are fixed and bounded. Delete endpoints require UUID identifiers. Inputs reject unknown fields, invalid server/finding references, unknown rules, unsafe durations, non-finite numbers, and out-of-range thresholds.

Migration 006 adds three append-only control tables and preserves all existing findings, snapshots, notification state, and target configuration. A pre-migration binary ignores these tables; automatic downgrade or table deletion is intentionally not performed.

Creating or deleting a maintenance window, suppression, or threshold override adds a fixed-summary event to the durable audit log. Operator-entered descriptions and reasons are deliberately excluded from audit summaries; the control record remains the authoritative location for that context.
