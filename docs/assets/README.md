# Product assets

The screenshots in this directory are captured from the real React interface with synthetic API responses. They contain no production database names, queries, hosts, credentials, or user data.

`pgsentinel-operator-controls.png` is reproducible with `node frontend/scripts/capture-operator-controls-screenshot.mjs`.

`pgsentinel-incident-timeline.png` is generated with `npm --prefix frontend run screenshot:incidents` and uses synthetic finding lifecycle data.

`pgsentinel-audit-log.png` is generated with `npm --prefix frontend run screenshot:audit` and contains only synthetic actors and resources.

## Regenerate screenshots

From the repository root:

```bash
npm ci --prefix frontend
npx --prefix frontend playwright install chromium
npm --prefix frontend run screenshots
npm --prefix frontend run screenshot:routing
npm --prefix frontend run screenshot:replication
npm --prefix frontend run screenshot:incidents
npm --prefix frontend run screenshot:audit
```

The capture scripts start a local Vite process, intercept `/api/v1` calls inside Playwright, and write deterministic dark-theme images here. Review every changed image before committing it. Update the corresponding synthetic fixture under `frontend/scripts/` when the UI intentionally changes.

The browser download is a maintainer tool only. It is not part of the PGSentinel application image or runtime dependency set.
