# Deployment and configuration change history

PGSentinel correlates query latency regressions with operational changes that occurred during the two anomalous intervals. Correlation is temporal evidence only; it does not claim that a deployment or setting change caused the regression.

Administrators can record a deployment under **Settings → Change history** by selecting a server, entering a short release summary, and providing its occurrence time. Deployment markers can cover events from the previous year and can be deleted. Creation and deletion are recorded in the audit log.

The metadata collector automatically compares the monitored PostgreSQL settings with the previous successful configuration snapshot. Changed setting names and their old and new values are stored as a configuration event. The initial snapshot does not create an event, and unchanged collections do not create duplicates.

Change history is retained for one year and capped at 10,000 events per server.

When a persistent query regression is detected, up to five deployment or configuration events within its anomalous window are added to the finding evidence. The finding remains at medium confidence because time overlap is not proof of causation.

API endpoints:

- `GET /api/v1/change-events?serverId=<uuid>` lists up to 100 recent events by default.
- `POST /api/v1/deployments` records an administrator-supplied deployment marker.
- `DELETE /api/v1/deployments/{id}` deletes a deployment marker. Automatically detected configuration events cannot be deleted.
