# Organic growth

PGSentinel should grow through a useful product, clear onboarding, technically accurate content, and responsive community work. Automation prepares information for maintainers; it does not manufacture engagement or publish promotional content without review.

## Objectives

- Help more PostgreSQL operators discover and successfully try PGSentinel.
- Earn GitHub stars by being useful and easy to understand.
- Convert users with concrete feedback into contributors where appropriate.
- Make bug reports, analyzer feedback, and operational questions more actionable.
- Learn which onboarding and product areas cause recurring friction.

Stars, forks, and traffic are signals rather than goals in isolation. Issue response, setup success, useful feedback, repeat contributors, and release adoption provide important context.

## GitHub-native architecture

```text
                    GitHub Repository
                           │
             ┌─────────────┼─────────────┐
             │             │             │
             ▼             ▼             ▼
          Releases       Schedule       Community
             │             │             │
             ▼             ▼             ▼
         Release       Weekly Stats    Issues / PRs
         metadata          │             │
             ▼             ▼             ▼
       Announcement     Growth Report   Contributor
          drafts                          support
             │
             ▼
        Maintainer review
             │
             ▼
      Manual publishing
```

The key principle is: **automate preparation, not spam**.

## Release workflow

```text
Develop
  → merge features normally after review
  → prepare a dedicated release pull request
  → publish through the existing protected Release workflow
  → GitHub prepares an announcement artifact
  → maintainer reviews and edits the content
  → maintainer manually posts it where appropriate
```

After the existing `Release` workflow succeeds, [`release-announcement.yml`](../.github/workflows/release-announcement.yml) reads that published GitHub Release and runs `scripts/generate-release-announcement.py`. The resulting Markdown contains short, community, and project Discussion drafts plus key changes and verified release links. It appears in the Actions job summary and remains available as a 30-day artifact.

The workflow runs after `Release` rather than relying only on the `release.published` event. Releases created with GitHub's built-in workflow token do not start another workflow from that event. A manual dispatch can regenerate a draft for an existing tag.

The announcement workflow has only `contents: read`. It cannot create releases, Discussions, issues, comments, commits, or external posts.

## Weekly report

[`weekly-growth-report.yml`](../.github/workflows/weekly-growth-report.yml) runs each Monday and can also be started manually. It queries the GitHub API and produces:

- a concise Markdown report in the job summary;
- the same report as a 90-day artifact;
- a JSON snapshot used by the next successful run for week-over-week deltas.

The report includes stars, forks, watchers, open issues and pull requests, contributor count, recent commits, merged pull requests, releases, and newly opened issues. It tries the repository traffic endpoints for views and clones. If the built-in token lacks access, those fields are reported as `Unavailable` and the workflow continues; it never estimates a missing value.

Historical state is read from the previous successful run's retained artifact. The workflow does not commit analytics into the repository and does not create or update a tracking issue. If artifact retention expires or a previous snapshot is missing, absolute values still appear and deltas are omitted.

Permissions are limited to:

- `actions: read` to retrieve the prior snapshot artifact;
- `contents: read` for checkout and public repository metadata;
- `issues: read` and `pull-requests: read` for counts and activity.

No write permission, PAT, API key, or external secret is required.

## Community and contributor onboarding

- Use [GitHub Discussions](https://github.com/matta813/PGSentinel/discussions) for installation help, troubleshooting, and early design questions.
- Use issue forms for reproducible bugs, analyzer feedback, and focused feature proposals.
- Use the security advisory link for vulnerabilities; never request sensitive reports in public.
- Point contributors to [CONTRIBUTING.md](../CONTRIBUTING.md) and [development.md](development.md) for setup and checks.
- Apply `good first issue` and `help wanted` only to genuine, scoped work with enough context to begin.
- Respond personally to first contributions. A comment bot is intentionally omitted because the existing templates and GitHub interface already provide adequate guidance without adding write permissions or boilerplate.

## Suitable communities

When a release or technical lesson is genuinely relevant, maintainers may consider communities centered on:

- PostgreSQL operation and administration;
- self-hosted software;
- homelab infrastructure;
- DevOps and SRE practice;
- database engineering and performance.

Participation should follow each community's rules and add technical value. Do not automate submissions, cross-post indiscriminately, or join a community only to drop a project link.

## Content ideas

- Explain one PostgreSQL health rule, its evidence, and its uncertainty.
- Write a troubleshooting lesson learned while reproducing a real, redacted bug.
- Show a new finding or workflow with a synthetic screenshot.
- Explain how Query Impact Score combines workload signals and where it can mislead.
- Document PostgreSQL behavior behind locks, vacuum pressure, long transactions, or index findings.
- Share a development retrospective about safety boundaries, release engineering, or observability tradeoffs.
- Compare before/after onboarding steps when documentation improves.

Useful content should stand on its own even if the reader never installs PGSentinel.

## Maintainer review checklist

Before sharing generated copy:

1. Confirm every listed change exists in the linked release notes.
2. Remove items irrelevant to the intended audience.
3. Add operational context or limitations the generated draft cannot know.
4. Recheck installation and upgrade instructions against the released version.
5. Post manually from a maintainer-controlled account only where it is welcome.
6. Record useful feedback as a Discussion or genuine issue, not as a vanity metric.

Never implement fake stars, accounts, downloads, clone traffic, unsolicited comments, mass messages, follow exchanges, or engagement bots.
