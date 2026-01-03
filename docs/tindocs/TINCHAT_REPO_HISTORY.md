# Tinchat Parent Repo History

This file preserves the git history from the `tinchat` parent repository before it was removed.
The parent repo was used to track the `tinode-src` submodule and store documentation.
All important content has been moved to `tinode-src/docs/tindocs/`.

**Archived:** January 3, 2026

---

## Commit History

| Commit | Date | Message |
|--------|------|---------|
| 483ace9 | 2026-01-03 | feat: Add server-side edit/unsend message support (submodule update) |
| 045c1de | 2026-01-03 | docs: Update EDIT_UNSEND spec with constraints |
| 585ee9a | 2026-01-03 | WIP on feature/reply-quote: 9791724 docs: Add server deployment state documentation |
| f8144dc | 2026-01-03 | index on feature/reply-quote: 9791724 docs: Add server deployment state documentation |
| 9791724 | 2026-01-03 | docs: Add server deployment state documentation |
| af2e94b | 2026-01-02 | feat: Add reply/quote messages feature |
| 4c51aae | 2025-12-31 | Add emoji reactions documentation |
| 4770712 | 2025-12-31 | Fix: user ID is in meta.sub[].topic, verified with live server test |
| f04961a | 2025-12-31 | Correct user search procedure - public must be string, user ID in meta.sub[].user |
| 416460e | 2025-12-31 | Add user search documentation |
| a74e94c | 2025-12-31 | Add comprehensive Tinode custom server documentation |

---

## Branches

- `main` (formerly `master`)
- `feature/reply-quote`
- `feature/edit-unsend`

---

## What was in this repo

The `tinchat` repo was a wrapper that contained:
- `tinode-src/` - Git submodule pointing to `scalecode-solutions/chat` fork
- `docs/` - Feature specifications and documentation (now in `tinode-src/docs/tindocs/`)
- `FEATURE_ROADMAP.md` - Feature planning document
- `tinchat.code-workspace` - VS Code workspace file

All documentation has been migrated to `tinode-src/docs/tindocs/`.

---

## GitHub Repository

The parent repo was hosted at: `https://github.com/scalecode-solutions/tinchat`

This repo is no longer needed as all work is now done directly in the fork:
`https://github.com/scalecode-solutions/chat`
