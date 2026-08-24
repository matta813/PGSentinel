# Product assets

The screenshots in this directory are captured from the real React interface with synthetic API responses. They contain no production database names, queries, hosts, credentials, or user data.

## Regenerate screenshots

From the repository root:

```bash
npm ci --prefix frontend
npx --prefix frontend playwright install chromium
npm --prefix frontend run screenshots
```

The capture script starts a local Vite process, intercepts `/api/v1` calls inside Playwright, and writes deterministic dark-theme images here. Review every changed image before committing it. Update the synthetic fixture in `frontend/scripts/capture-product-screenshots.mjs` when the UI intentionally changes.

The browser download is a maintainer tool only. It is not part of the PGSentinel application image or runtime dependency set.
