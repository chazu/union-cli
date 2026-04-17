# union

Composable, versioned snippet management for AGENTS.md / CLAUDE.md files.

Snippets (**clauses**) live in a central git-backed store and compose into
registered projects (**shops**) via marked regions in each shop's
AGENTS.md (its **contract**). See `chat.md` for design discussion and
`docs/superpowers/specs/2026-04-16-union-mvp-design.md` for the full spec.

## Status

Scaffold. Module: `github.com/chazu/union`.

## Command sketch

```
union init                     # set up the hall
union new <path>               # author a clause (editor or stdin)
union clauses                  # list clauses in the store
union organize .               # register current dir as a shop
union ratify <path>            # add clause to this shop's contract
union strike <path>            # remove clause from this shop's contract
union contract                 # show clauses present in this shop's contract
union expel <path>             # remove clause from the store
```
