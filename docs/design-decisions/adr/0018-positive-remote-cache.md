# Positive remote branch evidence is cached conservatively

## Status

Accepted on 2026-09-09.

## Context

`ww list` and `ww clean --dry-run` already batch remote branch checks into one
`git ls-remote --heads` call per configured remote, but separate CLI processes
repeat those calls. Remote latency can dominate status evaluation in a
workspace with several repositories or remotes.

Remote branch absence is safety-sensitive: a false absence can classify an
active worktree as `stale` and make it cleanable. A false presence only delays
stale classification. The cache must therefore preserve the existing live
query as the authority whenever positive evidence is insufficient or unsafe to
reuse.

## Decision

- Persist only a complete positive branch set returned by a successful
  `ls-remote --heads <remote>` query.
- Give every entry a fixed 30-second TTL. An entry is a hit only when every
  currently requested candidate branch is in its unexpired positive set.
- If any candidate is absent, the entry is expired, the identity changes, or
  the cache is invalid, run the complete live query and replace the positive
  set only when that query succeeds.
- Never persist absence, a final `active`/`merged`/`stale` status, or a
  `cleanable` decision. Never use expired positive evidence after a live query
  fails; preserve the current error.
- Key entries by the canonical repository common directory, remote name, and a
  hash of all effective configured remote URL values. Persist the identity hash
  and branch evidence, but never raw URLs or credentials.
- Reuse the existing `os.UserCacheDir()/ww` store, private permissions, and
  atomic same-directory replacement. Cache failures remain silent and fail
  open to the existing uncached behavior.

## Consequences

- **Positive**: repeated list and clean status evaluations can avoid remote
  latency while retaining a bounded conservative safety property.
- **Positive**: a deleted remote branch can remain `active` temporarily, but
  cannot become prematurely `stale` or cleanable because of a cached absence.
- **Negative**: stale positive evidence can delay cleanup for at most 30
  seconds, and a candidate miss still pays for a complete live refresh.
- **Negative**: cache key hashing and validation add local work on cold or
  invalidated invocations, so cold performance budgets and warm benefit
  measurements remain separate.
