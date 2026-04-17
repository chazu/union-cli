# union

Composable, versioned snippet management for AGENTS.md / CLAUDE.md files.

Snippets (**clauses**) live in a central git-backed store at `$UNION_DIR`
(default `~/.union`) and are composed into registered projects (**shops**)
via marked regions in each shop's contract file (default `AGENTS.md`).
Edits to a clause propagate immediately into every shop that ratified it,
leaving the resulting diff uncommitted for the user to review.

See `docs/superpowers/specs/2026-04-16-union-mvp-design.md` for the full
design.

## Install

```
go install github.com/chazu/union/cmd/union@latest
```

## Quick start

```bash
# one-time setup
union init

# author a clause
printf 'Be helpful and direct.\n' | union new base/identity

# register a project, ratify the clause into its AGENTS.md
cd ~/dev/my-project
union organize .
union ratify base/identity
union contract                      # → base/identity
cat AGENTS.md                       # shows the marked block

# edits propagate automatically
union edit base/identity            # opens $EDITOR; save → updates AGENTS.md here
```

## Command reference

| Command | Purpose |
|---|---|
| `union init` | Create `$UNION_DIR` and init its git repo |
| `union new <path>` | Author a new clause (editor, stdin, or `-f FILE`) |
| `union clauses [prefix]` | List clauses in the store |
| `union show <path>` | Print a clause |
| `union edit <path>` | Edit a clause; propagates to ratified shops |
| `union expel <path>` | Delete a clause; strikes it from ratified shops |
| `union organize [dir]` | Register a directory as a shop |
| `union shops` | List registered shops |
| `union disband <dir>` | Unregister a shop |
| `union ratify <path>` | Add a clause to this shop's contract |
| `union strike <path>` | Remove a clause from this shop's contract |
| `union contract` | Show clauses currently in this shop's contract |

## Contract markers

Ratified clauses are wrapped in HTML-comment markers:

```markdown
<!-- BEGIN union:base/identity -->
...clause content...
<!-- END union:base/identity -->
```

Content outside markers is preserved untouched across rewrites.
