# Presentation Agent Spec

Convert any text input (Markdown, JSON, plain prose) into a polished, self-contained HTML document using an LLM agent with write/patch/validate tools.

## Components

### 1. `write-html-file` Tool

Writes an HTML file to disk, then validates HTML structure, inline JS syntax, and inline CSS syntax. Returns structured errors so the agent can fix issues without a full rewrite.

**Input**
- `filePath: string` — absolute path to write the `.html` file
- `html: string` — complete self-contained document (all CSS and JS inlined)

**Output**
- `success: boolean`
- `filePath: string`
- `sizeBytes: number`
- `validation: ValidationResult`

**Validation pipeline** (runs on every write):
1. **HTML** — `npx html-validate --preset=standard --formatter=json`
2. **JS** — extracts first `<script>` block, parses with `acorn` (ecmaVersion 2022)
3. **CSS** — extracts first `<style>` block, checks brace balance with an inline Node script

`ValidationResult`:
```ts
interface ValidationResult {
  valid: boolean;
  html: ValidationIssue[];
  js: ValidationIssue[];
  css: ValidationIssue[];
  summary: string; // e.g. "0 HTML error(s), 0 JS error(s), 0 CSS error(s)"
}

interface ValidationIssue {
  line?: number;
  col?: number;
  severity: 'error' | 'warning';
  message: string;
}
```

---

### 2. `patch-html-file` Tool

Surgically replaces an exact string in an existing HTML file, then re-validates. Use this to fix specific errors without rewriting the whole document.

**Input**
- `filePath: string`
- `search: string` — exact string to find (must be unique unless `replaceAll` is true)
- `replace: string` — replacement string
- `replaceAll?: boolean` — replace every occurrence (default: false)

**Output**
- `success: boolean`
- `filePath: string`
- `sizeBytes: number`
- `occurrencesReplaced: number`
- `validation: ValidationResult`

Throws if `search` is not found, or if there are multiple occurrences and `replaceAll` is not set.

---

### 3. Presentation Agent

**Model:** `anthropic/claude-haiku-4-5` (fast, cheap; good enough for layout/HTML generation)  
**Max steps:** 10  
**Tools:** `write-html-file`, `patch-html-file`

**System prompt summary:**
- Convert input to a beautiful, fully self-contained HTML file.
- Default output path: `/tmp/document-[title-slug].html`. Use user-specified path if provided (e.g. `Output: /path/to/file.html`).
- Output style: dark mode, no sticky header, sidebar table of contents, reading progress bar, responsive layout, print support, good typography.
- Call `write-html-file` once to save.
- If `validation.valid` is false, use `patch-html-file` to fix only the reported issues. Repeat until valid.
- Final response: file path, size in KB, estimated word count, validation summary, and `open [filePath]` command.

---

### 4. Presentation Workflow

Two-step linear workflow:

```
read-markdown → render-html
```

**Input schema:**
- `markdownPath: string` — absolute path to the source `.md` file
- `outputPath?: string` — absolute path for the output `.html` (optional)

**Step 1 — `read-markdown`:** reads the file from disk, passes `content` and `outputPath` forward.

**Step 2 — `render-html`:** calls the presentation agent with the prompt:
```
Convert the following markdown to a beautiful HTML document. [Output: <outputPath>]

<content>
```
Extracts the output file path from the agent's response text via regex (looks for "saved to", "written to", backtick-quoted `.html` paths, or `open <path>`). Falls back to `outputPath` or `/tmp/document.html`.

**Output schema:**
- `filePath: string`
- `sizeKb: number`
- `summary: string` — full agent response text

## Key Design Notes

- The agent self-heals: it validates after every write and patches until `valid === true`, up to the `maxSteps` limit.
- The workflow is stateless — no memory or thread context needed.
- The `patch-html-file` uniqueness check prevents accidental multi-occurrence replacements that could corrupt the document.
- CSS validation is intentionally lightweight (brace balance only) to avoid introducing a heavy dependency; HTML and JS get real parsers.
- `html-validate` is run via `npx --yes` so it requires no pre-installed dependency.
