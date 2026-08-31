# Diagnostic bundles

Administrators can export a bounded diagnostic ZIP from **Settings → Diagnostics** when opening a support request or providing analyzer feedback.

The bundle contains:

- build version, commit, generation time, and the redaction policy;
- server identity and operational status with host, username, password, and connection errors redacted;
- up to 1,000 findings, including lifecycle and bounded analyzer evidence;
- per-resource collection freshness and failure counts.

PGSentinel does not include notification destination configuration, authentication data, stored snapshots, metric history, or raw query text. Evidence fields labelled as SQL, statement, or query text are replaced with `[redacted]`. The export is recorded in the audit log without recording its contents.

The archive is intended to be safe by construction, but server names, tags, database names, relation names, finding summaries, and operational measurements remain visible because they are useful for diagnosis. Review the files before sharing them outside the operations team.

API clients can download the same archive with `GET /api/v1/diagnostic-bundle`. Only administrators are authorized because the endpoint exposes operational metadata.
