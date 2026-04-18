# Recall

Search past Claude Code conversations and work logs to answer "what did I already discuss / decide / build around X?".

Usage (user types into chat):

```
/recall <query>                # free-form query, may include filter hints
/recall wezterm detach tab
/recall tctl kubernetes exec auth pod
/recall WORK_PROJECTS_DIR last week
/recall plan in this branch
```

## Where the data lives

**Claude Code transcripts** — one JSONL file per session:

```
~/.claude/projects/<encoded-cwd>/<session-uuid>.jsonl
~/.claude/projects/<encoded-cwd>/<session-uuid>/subagents/agent-*.jsonl
```

`<encoded-cwd>` = absolute path with `/` → `-`. Examples:

| cwd | encoded |
|---|---|
| `/Users/cenk/projects/work` | `-Users-cenk-projects-work` |
| `/Users/cenk/projects/work/.work/tree/memory` | `-Users-cenk-projects-work--work-tree-memory` |

**Work logs** — `.work/space/<branch>/log.md` under each project in `$WORK_PROJECTS_DIR` (default `~/projects`).

## JSONL structure

One JSON object per line. Key fields: `type` (`user`/`assistant`/`tool_use`/`progress`/`file-history-snapshot`), `message.content` (string for user; list of blocks for assistant — `text`, `thinking`, `tool_use`), `cwd`, `gitBranch`, `timestamp`, `sessionId`.

## Core pipeline — use this, not raw grep

**Raw grep over JSONL is unreliable** (tool-call blobs and file snapshots match by accident, regex OR is generic). Always pre-extract conversational text with `jq` first, then grep, then filter.

One-liner to extract clean text with metadata:

```bash
jq -r '
  select(.type=="user" or .type=="assistant") |
  .timestamp as $ts | .gitBranch as $br | (input_filename) as $f |
  if (.message.content | type) == "string" then [$ts, $br, $f, .message.content]|@tsv
  else (.message.content[]? | select(.type=="text" or .type=="thinking") | [$ts, $br, $f, (.text // .thinking)]|@tsv)
  end
' <file.jsonl>
```

Output columns: `timestamp \t branch \t file \t text`.

## Steps

### 1. Parse the query

Split query into **terms** and **filter hints**:

| Phrase | Action |
|---|---|
| `in <project>` | restrict to `~/.claude/projects/-Users-cenk-projects-<project>*/` |
| `in this project` | `pwd`, derive project root, encode |
| `in this branch` | `work id` → branch → post-filter on column 2 |
| `last week` / `last N days` / `yesterday` | prefilter with `find ... -mtime -N` |
| `since <date>` | post-filter column 1 |
| `subagents only` / `no subagents` | include/exclude `**/subagents/**` |

Remaining non-stopwords are **terms**. Treat multiple terms as **AND**, never OR (regex OR is noise).

### 2. Find candidate files (fast, cheap)

Do a rough raw `Grep` over the narrowest rare term to get a short candidate file list. Use `output_mode: files_with_matches`. If the query has no rare term, skip to step 3 over all files within the project/time filter.

Rule of thumb: if a single term returns >40 files, ask for a narrower term or a filter before extracting.

### 3. Extract + AND-filter

For each candidate file (or all files within filters), run the jq extractor, then chain `grep -i` per term:

```bash
jq -r '<extractor>' <f> | grep -i '<term1>' | grep -i '<term2>' | ...
```

Alias for convenience (in-session Bash):

```bash
recall_extract() { jq -r 'select(.type=="user" or .type=="assistant") |
  .timestamp as $ts | .gitBranch as $br | (input_filename) as $f |
  if (.message.content|type)=="string" then [$ts,$br,$f,.message.content]|@tsv
  else (.message.content[]?|select(.type=="text" or .type=="thinking")|[$ts,$br,$f,(.text//.thinking)]|@tsv) end' "$@"; }
```

### 4. Rank and dedupe

Collect all matching lines, then:

1. **Dedupe per session**: keep one row per `sessionId` (derivable from the file path — the UUID). Pick the row with the most term hits; on tie, the most recent.
2. **Sort**: descending by timestamp.
3. **Skip self-reference**: drop rows where the source file is a previous `/recall` conversation about the same terms (usually recognizable — the terms appear together with "recall" or inside a `~/.claude/recall-patterns.log` mention).
4. **Cap at 10 hits**. If more, surface the count and ask the user to narrow.

### 5. Present results

Compact, scannable, recency-first:

```
1. work/main                          2026-04-16  "agent jump: OSC 8 hyperlinks in agent inbox..."
   ~/.claude/projects/-Users-cenk-projects-work/2cfad352-e2ac-49b0-b488-99bacdbc9eae.jsonl

2. config-wezterm/HEAD                2026-04-17  "file:// links in Claude Code → spawn nvim in new tab"
   ~/.claude/projects/-Users-cenk--config-wezterm/951ccf27-5566-46b9-b9e1-6a90d1d8c7c8.jsonl
```

Rules:
- One line of excerpt per hit, trimmed to ~80 chars.
- Show `project/branch` and date.
- Include full path so the user can Cmd-click (wezterm OSC 8 handles it).
- **Never dump JSONL blobs.**
- Offer: "Read full context for hit #N?"

### 6. Log the pattern (feeds v2 CLI design)

After presenting, append one line to `~/.claude/recall-patterns.log`:

```
<ISO-timestamp>\t<raw-query>\t<filters-used>\t<pipeline>\t<hits-shown>/<candidate-files>\t<notes>
```

Notes field: anything surprising (e.g. "rare-term grep sufficient", "jq AND chain needed", "missed older phrasing").

## What NOT to do

- **Don't regex-OR generic terms** (`didn't work|still broken`): catastrophic noise.
- **Don't grep raw JSONL for conversational queries**: matches tool-use and snapshot blobs.
- **Don't Read whole sessions**: they can be 10k+ lines. Use `grep -n` for line numbers, then Read a ±20 line window only if the excerpt alone is insufficient.
- **Don't trust filename alone for branch**: the `gitBranch` field in each line is authoritative (worktrees move, branches rename).
- **Don't OR across terms**. Multiple query terms = AND.

## Tips

- **Rare identifier search is free and precise.** `WORK_PROJECTS_DIR`, `run_child_process`, a struct name — raw grep is fine for these.
- **Concept search needs synonym expansion.** "dot-separated branch naming" only finds sessions using that exact phrase. For concepts, try 2–3 phrasings and union the results.
- **Filters compose.** `in work last week` → time prefilter + path restriction.
