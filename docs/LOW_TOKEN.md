# Low-token, measured

Hera's thesis is **MCP-grade reach over a live Godot editor at a fraction of the
context tokens**. This page backs that claim with numbers — measured where we
can, estimated transparently where we can't — plus a reproducer so you can check
it yourself.

There are two separate costs to an agent's context budget. Hera wins on both.

---

## 1. Fixed surface cost — what the agent must "know" each turn

An MCP server advertises its tools by putting every tool's **definition**
(name + description + JSON input schema) in the `tools` array of *every* request.
That cost scales with the number of tools, and the real Godot MCP servers expose
a lot of them:

| Godot MCP addon | Tools | Tool definitions resident **per turn** (est.) |
|-----------------|------:|----------------------------------------------:|
| `godot-ai` (dlight)        | 120+ | ~12,000 – 24,000 tok |
| `godot-mcp-native`         | 155  | ~15,500 – 31,000 tok |

> Estimate: tool counts are from the assets' own store pages; per-tool schema is
> assumed at **100–200 tokens** (a name, a one-line description, and a small JSON
> input schema). We did not run those servers — this is an order-of-magnitude
> figure, not billing.

Hera carries **zero tool schemas** in context. The agent just runs a shell
command. The only surface it loads is one doc, read once and prompt-cacheable,
and it stays **flat no matter how many commands exist**:

| Hera surface (one-time, cacheable) | Size |
|------------------------------------|-----:|
| `AGENTS.md` (how to drive the CLI) | 4,182 chars (~1,045 tok) |
| `docs/COMMANDS.md` (full reference) | 3,813 chars (~953 tok) |

So before any work is done, an agent pays **~12k–31k resident tokens** to a
120–155-tool Godot MCP server, versus **~1k tokens** (one cacheable doc) for
Hera — and Hera's number does not grow as the command surface grows.

---

## 2. Per-operation cost — the response to each action

Hera defaults to **compact JSON** (`--json` pretty-prints; `--ids` trims to bare
node paths). Measured against a live Godot 4.7 editor:

Sizes are `wc -c` (so they match the reproducer below; that counts the one
trailing newline the CLI prints):

| Operation | compact (default) | `--json` (pretty) |
|-----------|------------------:|------------------:|
| `status`                  |  194 chars (~48 tok) | 215 chars (~54 tok) |
| `scene tree` (1-node demo)|  116 chars (~29 tok) | 170 chars (~42 tok) |
| `node get .` (full Node2D property dump) | 744 chars (~186 tok) | 932 chars (~233 tok) |
| `node find` (all)         |   83 chars (~20 tok) | 133 chars (~33 tok) |

Compact is ~20–30% smaller than pretty on these calls, and `--ids` cuts a tree
or find result down to one path per line.

---

## Caveats (so the numbers stay honest)

- **Hera figures are measured** on a live Godot 4.7 editor; **MCP figures are
  estimated** from public tool counts × an assumed per-tool schema size. We did
  not run the MCP servers.
- Token counts use a **`chars / 4`** approximation (no model tokenizer). It is
  fine for relative scale, not for exact billing. JSON tends to tokenize a little
  denser than 4 chars/token, so these slightly *under*count both sides.
- **Prompt caching** lowers the per-call cost of resident tokens for both sides
  on a cache hit. The structural point survives caching: the MCP surface grows
  with every tool added; Hera's is one flat doc plus compact payloads.
- This is about **context economy, not a verdict on MCP.** MCP buys native,
  one-click client integration (Claude Code, Cursor, Codex). Hera trades that for
  the token budget and for working with anything that can run a shell command.

---

## Reproduce it

With a Godot 4.7 editor running the Hera Agent plugin and `hera` built
(`go build -o hera .`):

```sh
# per-operation response sizes (compact vs pretty)
for cmd in "status" "scene tree" "node get ." "node find"; do
  c=$(./hera $cmd | wc -c)
  p=$(./hera --json $cmd | wc -c)
  printf "%-16s compact=%4d chars  pretty=%4d chars\n" "$cmd" "$c" "$p"
done

# fixed surface doc size
wc -c AGENTS.md docs/COMMANDS.md
```

Approximate tokens by dividing chars by 4. The MCP side is arithmetic: tool count
(from each addon's store page) × ~100–200 tokens per tool schema.
