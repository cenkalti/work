# Image Generation MCP Spec

Small stdio MCP server that exposes OpenAI GPT Image 1.5 (`gpt-image-1`) to Claude Code as a tool with strict, typed arguments. Purpose: generate icons and UI assets from a text prompt plus rigid parameters (format, size, transparency, output path).

## Why GPT Image 1.5

- Native transparent PNG via `background: "transparent"` — first-class for icons.
- Best-in-class text rendering (~96% typography accuracy) — logos, signage, UI copy.
- Fixed, predictable sizes.
- Official Node SDK, stable REST contract.

Nano Banana 2 leads the LM Arena overall leaderboard but treats transparency less natively; GPT Image 1.5 is the better fit for the stated use case.

## Components

### 1. `generate_image` Tool

Calls `/v1/images/generations`, decodes each returned base64 image, writes to disk.

**Input**

- `prompt: string` — natural-language description.
- `output_path: string` — absolute path to write the image. Extension must match `format`.
- `format: "png" | "jpeg" | "webp"` — default `png`.
- `size: "1024x1024" | "1536x1024" | "1024x1536" | "auto"` — default `1024x1024`. These are the only values `gpt-image-1` accepts.
- `background: "transparent" | "opaque" | "auto"` — default `auto`. Honored only when `format` is `png` or `webp`.
- `quality: "low" | "medium" | "high" | "auto"` — default `high`. Transparency requires `medium` or `high`.
- `n: integer (1-10)` — default `1`. When `>1`, suffix `-1`, `-2`, … before the extension in `output_path`.

**Output**

```ts
interface GenerateImageResult {
  success: boolean;
  files: { path: string; sizeBytes: number }[];
  model: "gpt-image-1";
  usage: { input_tokens: number; output_tokens: number };
}
```

**Behavior**

- Pre-flight validation (fails before the API call):
  - Extension of `output_path` matches `format`.
  - `background === "transparent"` is incompatible with `format === "jpeg"`.
  - `output_path` is absolute (or resolved against `IMAGE_GEN_DEFAULT_DIR` if relative).
- Creates parent directory if missing (`fs.mkdir(..., { recursive: true })`).
- Decodes `response.data[i].b64_json` and writes bytes with `fs.writeFile`.
- Returns absolute paths of every file written.

---

### 2. `edit_image` Tool

Wraps `/v1/images/edits` for inpainting and variations of an existing asset.

**Input**

- `prompt: string`
- `input_path: string` — existing image on disk.
- `mask_path?: string` — optional PNG mask; alpha channel marks the editable region.
- `output_path: string`
- `format`, `size`, `background`, `quality` — same semantics and defaults as `generate_image`.

**Output** — same shape as `GenerateImageResult`.

**Behavior** — same validation and write rules as `generate_image`. `input_path` and (if provided) `mask_path` are streamed to the SDK as `fs.ReadStream`.

---

### 3. Configuration

- `OPENAI_API_KEY` — required.
- `IMAGE_GEN_DEFAULT_DIR` — optional. If set and `output_path` is relative, resolve against this directory. If unset, relative paths are rejected.

---

### 4. Runtime & Packaging

- **Language:** TypeScript, Node ≥20.
- **Dependencies:** `@modelcontextprotocol/sdk`, `openai`, `zod`.
- **Transport:** stdio.
- **Entry point:** `bin/image-gen-mcp.js` (compiled from `src/index.ts`).
- **Install into Claude Code** (user scope so every project sees it):
  ```
  claude mcp add --scope user --transport stdio image-gen \
    -- node /absolute/path/to/image-gen-mcp/bin/image-gen-mcp.js
  ```

---

### 5. OpenAI SDK Call Shape

```ts
import OpenAI from "openai";
import { promises as fs } from "node:fs";

const openai = new OpenAI();

const r = await openai.images.generate({
  model: "gpt-image-1",
  prompt,
  size,                   // "1024x1024" | "1536x1024" | "1024x1536" | "auto"
  background,             // "transparent" | "opaque" | "auto"
  output_format: format,  // "png" | "jpeg" | "webp"
  quality,                // "low" | "medium" | "high" | "auto"
  n,
});

for (const [i, item] of r.data.entries()) {
  const bytes = Buffer.from(item.b64_json!, "base64");
  await fs.writeFile(pathFor(i), bytes);
}
```

---

### 6. Error Handling

- OpenAI API errors surface as MCP tool errors with the API message and HTTP status.
- Input validation errors (bad extension, transparent+jpeg, relative path without default dir) fail fast with a clear message, before any network call.
- No retries in v1. Claude Code retries at the tool-call level if needed.

---

### 7. Verification

After implementation, run these end-to-end:

1. `npm run build && node bin/image-gen-mcp.js` — server starts cleanly on stdio.
2. Register with Claude Code (command above).
3. In a Claude Code session:
   - "Generate a transparent PNG icon of a blue fox head at 1024x1024, save to `/tmp/fox.png`" → verify file exists and alpha channel is present: `sips -g hasAlpha /tmp/fox.png`.
   - "Generate a 1536x1024 JPEG landscape hero to `/tmp/hero.jpg`" → `sips -g pixelWidth /tmp/hero.jpg` returns `1536`.
   - Ask for `format: "jpeg"` + `background: "transparent"` → tool returns validation error without making an API call.
4. Confirm `usage.input_tokens` / `usage.output_tokens` are non-zero in the tool result.

## Design Notes

- One model, one endpoint — keeps the surface area minimal and the tool schema strictly typed.
- Rigid enums on `size`, `format`, `background`, `quality` let Claude Code pass exact values and surface validation errors early.
- Output-path contract (absolute path in, absolute path back) keeps the MCP stateless; no temp-file indirection.
- `n > 1` suffix rule is deterministic so callers can predict the full file list.
