# PGSentinel branding

The original mark combines a database cylinder with a watchful eye. Keep its silhouette and clear space intact; do not substitute the PostgreSQL elephant.

- Canonical editable vectors: [mark](pgsentinel-mark.svg) and [wordmark](pgsentinel-logo.svg).
- Light accent: `#168a7a`; dark accent: `#42b9a4`. Standalone SVGs follow the system colour preference. The sidebar mark inherits the application theme through `currentColor`.
- Browser favicon: `frontend/public/favicon.svg`, referenced by `frontend/index.html`. After editing the canonical mark, copy it with `cp docs/assets/brand/pgsentinel-mark.svg frontend/public/favicon.svg`.
- The inline `frontend/src/components/BrandMark.tsx` uses the same geometry to follow the user's in-app theme. Update it when changing the canonical mark.

Assets are hand-authored SVGs, with no raster files or generation dependencies. The wordmark uses system fonts; the symbol has no font dependency and is intended for small application/icon sizes. Beside visible product text, the inline mark is decorative and hidden from assistive technology; standalone SVGs include an accessible title.
