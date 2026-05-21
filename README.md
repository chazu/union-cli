<p align="center"><img src="site/logo.png" alt="union logo"></p>

# union

Composable, versioned snippet management for AGENTS.md / CLAUDE.md files.

Snippets (**clauses**) live in a central git-backed store at `$UNION_DIR`
(default `~/.union`) and are composed into registered projects (**shops**)
via marked regions in each shop's contract file (default `AGENTS.md`).
Edits to a clause propagate immediately into every shop that ratified it,
leaving the resulting diff uncommitted for the user to review.

Or, as the stochastic parrot put it:
> Helm for agent context, but less cursed and more explicit.

## Install

```
go install github.com/chazu/union/cmd/union@latest
```

## Quick start

```bash
# one-time setup — creates a 'default' store
union init

# or: name the initial store
# union init personal

# author a clause (qualified as <store>:<path>)
printf 'Be helpful and direct.\n' | union new default:base/identity

# register a project, ratify the clause into its AGENTS.md
cd ~/dev/my-project
union organize .
union ratify default:base/identity
union contract                      # → default:base/identity
cat AGENTS.md                       # shows the marked block

# edits propagate automatically
union edit default:base/identity    # opens $EDITOR; save → updates AGENTS.md here

# add a second store, ratify from both
union store add personal
union new personal:writing/voice -f voice.md
union ratify personal:writing/voice
```

## Command reference

| Command | Purpose |
|---|---|
| `union init [name]` | Create `$UNION_DIR` and an initial store (default name: `default`) |
| `union new <store:path> [-f FILE]` | Author a new clause (editor, stdin, or `-f FILE`; `-f -` reads stdin) |
| `union clauses [store:prefix]` | List clauses across stores; optional `store:prefix` filter |
| `union show <store:path>` | Print a clause |
| `union edit <store:path>` | Edit a clause in `$VISUAL`/`$EDITOR`; propagates to ratified shops |
| `union expel <store:path>` | Delete a clause; strikes it from ratified shops |
| `union organize [dir] [--contract NAME]` | Register a directory as a shop |
| `union shops` | List registered shops |
| `union disband <dir>` | Unregister a shop |
| `union ratify <store:path>` | Add a clause to this shop's contract |
| `union strike <store:path>` | Remove a clause from this shop's contract |
| `union contract` | Show clauses currently in this shop's contract |
| `union store add <name>` | Create a new store |
| `union store list` | List stores |
| `union store remove <name>` | Delete a store (refused if any shop still ratifies from it) |
| `union store remote add <store> <name> <url>` | Add a git remote to a store |
| `union store remote remove <store> <name>` | Remove a git remote |
| `union store remote list <store>` | List a store's remotes |
| `union store push <store> [remote] [branch]` | `git push` in a store |
| `union store pull <store> [remote] [branch]` | `git pull --rebase` in a store |
| `union store clone <url> [name]` | Clone a remote repo as a new store |
| `union store fetch <store> [remote]` | `git fetch` in a store |
| `union store status <store>` | `git status --short --branch` for a store |
| `union rename <old> <new>` | Rename clause within same store; rewrites all shop markers |
| `union verify` | Check all shops' contracts match their store clauses (CI-friendly) |
| `union sync` | Re-propagate all clauses to all shops (repair drift) |
| `union status` | Show global summary: stores, clauses, shops |
| `union search <pattern>` | Search clause bodies for a substring |
| `union log <store:path>` | Show git log for a clause |
| `union orphans` | List clauses not ratified by any shop |
| `union which` | Print union paths for debugging |
| `union completion bash\|zsh\|fish` | Generate shell completions |
| `union harness detect` | Auto-detect harnesses in current shop |
| `union harness list` | List configured/detected harnesses with capabilities |
| `union harness add <name>` | Explicitly declare a harness |
| `union harness remove <name>` | Remove a harness declaration |
| `union emit [--write] [--harness X]` | Render hook clauses into native harness configs |
| `union import hooks [--harness X]` | Import existing harness hooks into clauses |
| `union pointer sync` | Create guidance file pointers (e.g., CLAUDE.md → AGENTS.md) |
| `union pointer list` | Show existing pointer files |
| `union guide <topic>` | Print agent-oriented documentation |

## Cross-harness hooks

Union manages hooks across multiple AI coding harnesses from a single
source of truth. Hook clauses use normalized event names and degrade
gracefully when a harness doesn't support an event.

### Supported harnesses

| Harness | Config target | Supported events |
|---------|--------------|-----------------|
| claude | `.claude/settings.json` | SessionStart, PreToolUse, PostToolUse, UserPrompt, Stop |
| opencode | `.opencode/plugins/union-hooks.mjs` | SessionStart, PreToolUse, PostToolUse, UserPrompt, Stop, PreCommit |
| codex | `codex.toml` | PreCommit |
| jcode | `.jcode/settings.json` | SessionStart, PreToolUse, PostToolUse, Stop |

### Hook clause format

```markdown
---
type: hook
event: SessionStart
harnesses: [claude, opencode]
degrade: skip
---
echo "started in {{shop.dir}}" >> /tmp/{{shop.name}}.log
```

### Template variables

| Variable | Expands to |
|----------|-----------|
| `{{shop.dir}}` | Absolute shop path |
| `{{shop.name}}` | Directory basename |
| `{{user.email}}` | `git config user.email` |
| `{{harness.name}}` | Target adapter name |
| `{{union.dir}}` | Union root path |
| `{{env.NAME}}` | Environment variable |

### Workflow

```bash
# create a hook clause
union new default:hooks/session-start

# ratify it into your project
union ratify default:hooks/session-start

# preview what would be emitted
union emit

# write native configs
union emit --write

# or import existing hooks first
union import hooks --harness claude
```

## Project config (`union.toml`)

Optional per-project manifest. Convention over configuration — most
projects need no config file at all (harnesses are auto-detected).

```toml
[harnesses.claude]
settings = ".claude/settings.json"

[harnesses.opencode]
settings = "opencode.json"

[hooks]
ratified = ["default:hooks/session-start"]

[pointers]
targets = ["CLAUDE.md"]
```

## Guidance file pointers

When your contract is `AGENTS.md` but a harness expects `CLAUDE.md`,
union can create a pointer file:

```bash
union pointer sync
# → CLAUDE.md points to AGENTS.md
```

The pointer file contains `@AGENTS.md` — a convention telling agents
to read the referenced file.

## Contract markers

Ratified clauses are wrapped in HTML-comment markers that carry the full
`store:path`:

```markdown
<!-- BEGIN union:default:base/identity -->
...clause content...
<!-- END union:default:base/identity -->
```

Content outside markers is preserved untouched across rewrites.
