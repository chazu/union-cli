package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newGuideCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "guide",
		Short: "Agent-oriented documentation for using union.",
		Long:  "Print guides explaining how to use union. Designed to be read by AI agents.",
	}
	cmd.AddCommand(
		newGuideOverviewCmd(),
		newGuideHooksCmd(),
		newGuideClausesCmd(),
		newGuideEmitCmd(),
		newGuideTemplatesCmd(),
	)
	return cmd
}

func newGuideOverviewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "overview",
		Short: "What union is and core workflow.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprint(cmd.OutOrStdout(), guideOverview)
		},
	}
}

func newGuideHooksCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hooks",
		Short: "How to manage cross-harness hooks.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprint(cmd.OutOrStdout(), guideHooks)
		},
	}
}

func newGuideClausesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clauses",
		Short: "How to create and manage clauses.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprint(cmd.OutOrStdout(), guideClauses)
		},
	}
}

func newGuideEmitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "emit",
		Short: "How emit translates hooks to native configs.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprint(cmd.OutOrStdout(), guideEmit)
		},
	}
}

func newGuideTemplatesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "templates",
		Short: "Template variables available in hook clauses.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprint(cmd.OutOrStdout(), guideTemplates)
		},
	}
}

const guideOverview = `# Union Overview

Union manages reusable markdown snippets (clauses) across AI-assisted projects.

## Core Model

- CLAUSE: A versioned markdown snippet in a git-backed store.
- STORE: Git repository at $UNION_DIR/stores/<name>/ containing clauses.
- SHOP: A registered project directory with a contract file.
- CONTRACT: The shop's AGENTS.md (or other file) with marked clause regions.

## Workflow

1. Create clauses: union new <store:path>
2. Register projects: union organize <dir>
3. Add clauses to projects: union ratify <store:path>
4. Edit clauses (propagates everywhere): union edit <store:path>
5. Remove from a project: union strike <store:path>

## Key Properties

- Edits to a clause auto-propagate to all shops that ratified it.
- Content outside union markers is never touched.
- Every store mutation auto-commits in git.
- Propagated changes are left uncommitted for user review.

## Qualified Paths

All clauses use the format: <store>:<path>
Examples: default:base/identity, personal:lang/go

## Files

- $UNION_DIR/stores/<name>/clauses/ — clause storage
- $UNION_DIR/shops.toml — shop registry
- <shop>/union.toml — per-project harness config (optional)
- <shop>/AGENTS.md — contract file (default)
`

const guideHooks = `# Cross-Harness Hook Management

Union can manage hooks across multiple AI coding harnesses (Claude Code,
OpenCode, Codex, JCode) from a single source of truth.

## Supported Harnesses

| Harness    | Config target                      | Events supported                                    |
|------------|------------------------------------|----------------------------------------------------|
| claude     | .claude/settings.json              | SessionStart, PreToolUse, PostToolUse, UserPrompt, Stop |
| opencode   | .opencode/plugins/union-hooks.mjs  | SessionStart, PreToolUse, PostToolUse, UserPrompt, Stop, PreCommit |
| codex      | codex.toml                         | PreCommit                                          |
| jcode      | .jcode/settings.json               | SessionStart, PreToolUse, PostToolUse, Stop        |

## Hook Clause Format

Hook clauses are regular clauses with YAML frontmatter:

    ---
    type: hook
    event: SessionStart
    harnesses: [claude, opencode]
    degrade: skip
    matcher: Bash
    timeout: 5000
    ---
    echo "session started in {{shop.dir}}"

## Frontmatter Fields

- type: Must be "hook"
- event: Normalized event name (see table above)
- harnesses: Optional list — emit only to these (default: all)
- degrade: What to do if harness doesn't support the event
  - skip: silently omit (default)
  - warn: emit warning, omit hook
  - error: fail the emit
- matcher: Tool/pattern filter (harness-specific)
- timeout: Milliseconds before hook times out

## Commands

- union harness detect — find harnesses in current shop
- union harness list — show configured/detected harnesses
- union harness add <name> — declare a harness
- union harness remove <name> — remove a harness
- union emit — preview what would be emitted (dry-run)
- union emit --write — write native configs
- union emit --harness <name> — emit to one harness only
- union import hooks — import existing hooks into clauses

## Graceful Degradation

When a hook targets an event that a harness doesn't support:
- degrade: skip → hook is silently omitted for that harness
- degrade: warn → warning printed, hook omitted
- degrade: error → emit fails with an error

This means a single hook clause can target all harnesses and will
automatically do the right thing for each one.

## Event Name Mapping

Union uses normalized event names. Each adapter maps them to native names:

| Union Event   | Claude Code       | OpenCode             | Codex       | JCode             |
|---------------|-------------------|----------------------|-------------|-------------------|
| SessionStart  | SessionStart      | session.created      | —           | on_session_start  |
| PreToolUse    | PreToolUse        | tool.execute.before  | —           | before_tool       |
| PostToolUse   | PostToolUse       | tool.execute.after   | —           | after_tool        |
| UserPrompt    | UserPromptSubmit  | message.updated      | —           | —                 |
| Stop          | Stop              | session.deleted      | —           | on_session_end    |
| PreCommit     | —                 | tool.before.bash     | pre_commit  | —                 |

## Union-Managed Prefix

Emitted hooks are prefixed with "[union] " in their command string.
This allows union emit to identify and replace its own hooks without
disturbing user-added hooks.
`

const guideClauses = `# Clause Management

## Creating Clauses

    # From editor
    union new default:hooks/session-start

    # From stdin
    echo "make test" | union new default:hooks/pre-commit

    # From file
    union new default:hooks/lint -f hook-lint.md

## Hook Clauses

Hook clauses require YAML frontmatter with type: hook.
See 'union guide hooks' for the full format.

## Listing and Searching

    union clauses                  # all clauses
    union clauses default:hooks    # filter by prefix
    union search "make test"       # search bodies
    union orphans                  # clauses ratified nowhere

## Editing

    union edit default:hooks/lint  # opens $EDITOR, auto-commits, propagates

## Ratifying Into Shops

    cd /path/to/project
    union organize .
    union ratify default:hooks/session-start
    union contract                 # see what's ratified here

## Lifecycle

    union new → union ratify → union edit → union strike → union expel
    create      add to shop     modify       remove from    delete
                                             shop           entirely
`

const guideEmit = `# How Emit Works

'union emit' translates hook clauses into native harness configuration.

## Process

1. Resolve current shop (from cwd)
2. Load union.toml or auto-detect harnesses
3. Read contract file, find all ratified hook clauses
4. For each harness:
   a. Check which hooks it supports (filter by capabilities)
   b. Apply graceful degradation (skip/warn/error)
   c. Expand template variables ({{shop.dir}}, {{env.X}}, etc.)
   d. Call adapter.Emit() to produce native config
5. Show diff (dry-run) or write files (--write)

## Idempotency

Running 'union emit --write' multiple times produces the same result.
Union-managed hooks are identified by the "[union] " prefix and replaced
each time. User-added hooks are preserved.

## Per-Harness Behavior

Claude Code:
  - Writes to .claude/settings.json
  - Hooks stored in hooks.<EventName> array
  - Each hook is a {matcher, hooks: [{type: "command", command: "..."}]}

OpenCode:
  - Generates .opencode/plugins/union-hooks.mjs
  - ES module plugin using execSync for shell commands
  - Degrade behavior handled via try/catch in generated JS

Codex:
  - Writes to codex.toml
  - Only supports PreCommit → hooks.pre_commit array

JCode:
  - Writes to .jcode/settings.json
  - Similar to Claude but with snake_case event names

## Safety

- Default is dry-run (no --write = preview only)
- Never removes user hooks (only replaces [union]-prefixed ones)
- Creates directories as needed (e.g., .claude/, .opencode/plugins/)
`

const guideTemplates = `# Template Variables

Hook clause commands can contain {{variable}} placeholders that expand
at emit time. The stored clause keeps templates intact — only the emitted
native config gets concrete values.

## Built-in Variables (zero config, always available)

| Variable          | Expands to                          | Example                |
|-------------------|-------------------------------------|------------------------|
| {{shop.dir}}      | Absolute shop directory path        | /home/user/myapp       |
| {{shop.name}}     | Shop directory basename             | myapp                  |
| {{user.email}}    | git config user.email               | dev@example.com        |
| {{harness.name}}  | Current adapter being emitted to    | claude                 |
| {{union.dir}}     | Union root directory                | /home/user/.union      |

## Environment Variables

| Variable          | Expands to                          |
|-------------------|-------------------------------------|
| {{env.NAME}}      | os.Getenv("NAME")                   |
| {{env.HOME}}      | /home/user                          |
| {{env.CI}}        | true (if set)                       |

Missing env vars expand to empty string.

## Example

Clause stored as:
    ---
    type: hook
    event: SessionStart
    ---
    echo "{{shop.name}} session on {{harness.name}}" >> {{env.HOME}}/logs/sessions.log

Emitted to Claude Code as:
    [union] echo "myapp session on claude" >> /home/user/logs/sessions.log

Emitted to OpenCode as:
    [union] echo "myapp session on opencode" >> /home/user/logs/sessions.log

## Notes

- Expansion happens per-adapter ({{harness.name}} differs for each target)
- Re-emit needed if variables change (e.g., moved shop directory)
- Unknown {{variables}} are left as-is (not expanded, not errored)
`
