# GitHub discoverability checklist

Repository files now provide a product-focused README, reproducible screenshots, a social-preview image, community links, and release announcement tooling. The remaining items are GitHub repository settings and must be applied by an owner.

## Recommended repository metadata

### Description

Keep the current concise description:

> PostgreSQL monitoring and health analysis that explains what is wrong, why it matters, and what to investigate next.

### Website

The current website points back to the repository and adds no navigation value. Until a maintained product site or public demo exists, set the website to the documentation entry point:

`https://github.com/matta813/PGSentinel/blob/main/docs/README.md`

A GitHub Pages site is not recommended yet. The README already serves the same discovery path, while a second site would duplicate content and create another surface to keep current.

### Topics

Use focused search terms that describe the product and deployment model:

- `postgresql`
- `postgres`
- `postgres-monitoring`
- `database-monitoring`
- `database-observability`
- `self-hosted`
- `performance-monitoring`
- `devops`
- `docker`
- `golang`

This keeps useful implementation and operations topics while replacing broad or secondary terms such as `database`, `react`, `sqlite`, and `health-check` with phrases prospective users are more likely to search.

### Social preview

Upload [`assets/pgsentinel-social-preview.png`](assets/pgsentinel-social-preview.png) under **Settings → General → Social preview**. GitHub requires this owner action; committing the asset alone does not change the preview.

### Community features

- Keep GitHub Discussions enabled and use it for support and early design questions.
- Keep Issues enabled with the existing bug, feature, and analyzer templates.
- Apply `good first issue`, `help wanted`, `documentation`, `enhancement`, and `bug` only to genuine work that matches the label. Do not create placeholder issues for activity.
- Pin one welcome/support Discussion and, when useful, one release Discussion. Avoid duplicating every release announcement across multiple categories.

## Future demo decision

The repository currently has no safe hosted demo environment. Screenshots with synthetic fixtures are the lowest-maintenance trustworthy product tour. A public demo becomes worthwhile only when there is isolated hosting, automatic synthetic-data resets, abuse controls, and an owner for patching it. Never connect a public demo to a real PostgreSQL environment.
