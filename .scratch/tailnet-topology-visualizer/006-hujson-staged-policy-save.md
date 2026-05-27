# HuJSON staged policy save loop

Labels: ready-for-human
Type: AFK

## Status

Superseded by 008-scoped-acl-edit-save-loop.md for the first real save path.

## What to build

This ticket originally described the first safe ACL editing loop. That path now exists in 008-scoped-acl-edit-save-loop.md.

The broader staged commit experience is now tracked in 016-staged-commit-tray-and-hujson-diff.md. The new version should not be a raw HuJSON-first editor. It should be the review and save surface for graph-backed policy drafts.

## Acceptance criteria

- [x] The first safe save loop exists through 008-scoped-acl-edit-save-loop.md.
- [x] Follow-up staged review work is captured in 016-staged-commit-tray-and-hujson-diff.md.

## Blocked by

- 004-cloud-api-auth-policy-fetch.md
- 005-effective-access-edges.md
