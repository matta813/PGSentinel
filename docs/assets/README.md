# Product assets

The screenshots in this directory are captured from the real React interface with synthetic API responses. They contain no production database names, queries, hosts, credentials, or user data.

`pgsentinel-data-freshness.png` is reproducible with `node frontend/scripts/capture-data-freshness-screenshot.mjs`.

## Regenerate screenshots

From the repository root:

```bash
npm ci --prefix frontend
npx --prefix frontend playwright install chromium
npm --prefix frontend run screenshots
npm --prefix frontend run screenshot:routing
```

The capture scripts start a local Vite process, intercept `/api/v1` calls inside Playwright, and write deterministic dark-theme images here. Review every changed image before committing it. Update the corresponding synthetic fixture under `frontend/scripts/` when the UI intentionally changes.

The browser download is a maintainer tool only. It is not part of the PGSentinel application image or runtime dependency set.
