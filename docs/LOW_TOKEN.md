# Low-token, measured

Hera's thesis is **MCP-grade reach over a live Godot editor at a fraction of the
context tokens**. This page backs that claim with numbers — measured where we
can, estimated transparently where we can't — plus a reproducer so you can check
it yourself.

There are two separate costs to an agent's context budget. Hera wins on both.

---

## 1. Fixed surface cost — what the agent must "know" each turn

An MCP client normally exposes every server tool's **definition** (name +
description + JSON input schema) to the model in the request tool list. That
cost scales with the number of MCP tools, and real Godot MCP servers expose a
lot of them:

| Godot MCP addon sample | Publicly documented surface | Tool definitions resident **per turn** (est.) |
|------------------------|----------------------------:|----------------------------------------------:|
| Smaller sampled server | ~41 MCP tools / 120+ ops | ~4,100 – 8,200 tok |
| Larger sampled server | 155 MCP tools | ~15,500 – 31,000 tok |

> Estimate: tool counts are from public project docs. One sampled server uses
> roll-up MCP tools with an `op` enum, so the honest resident-schema count is
> its documented ~41 MCP tools, not its 120+ underlying operations. Per-tool
> schema is assumed at **100–200 tokens** (name, description, and JSON input
> schema). We did not run those servers — this is an order-of-magnitude figure,
> not billing.

Hera carries **zero tool schemas** in context. The agent just runs a shell
command. The surface it may load is cacheable markdown, not a per-turn tool
list. Those docs **do grow** as the command surface grows; they are still
loaded once (and prompt-cacheable), unlike MCP schemas that sit in every
request.

Sizes below are **UTF-8 byte counts** from this tree (`wc -c` / file length),
not tokenizer output. Approximate tokens as bytes/4.

| Hera surface (one-time, cacheable) | Bytes | ~tok |
|------------------------------------|------:|-----:|
| `plugins/hera-godot/skills/live-editor/SKILL.md` | 4,187 | ~1,047 |
| `AGENTS.md` (driver card) | 16,001 | ~4,000 |
| `docs/COMMANDS.md` (full reference) | 47,019 | ~11,755 |

So before any work is done, an MCP client in the sampled servers still pays
**~4k–31k resident tokens every turn** for tool schemas. Hera pays **nothing
per turn** for schemas. Loading the full command reference is no longer a ~1k
token footnote — that older claim is retired. Prefer the live-editor skill or
`AGENTS.md` for routine driving, and open `COMMANDS.md` when a flag or field
is in doubt.

---

## 2. Per-operation cost — the response to each action

Hera defaults to **compact JSON** (`--json` pretty-prints; `--ids` trims to
paths). Live sizes below were measured on Godot 4.7 except `status`, which uses
this tree's contract goldens (UTF-8 bytes, including the trailing newline).

| Operation | compact (default) | `--json` (pretty) |
|-----------|------------------:|------------------:|
| `status` (compact CLI) | 316 bytes (~79 tok, fixture) | — |
| `status --capabilities` | 843 bytes (~211 tok, fixture) | — |
| `scene tree` (1-node demo)|  116 bytes (~29 tok) | 170 bytes (~42 tok) |
| `node get .` (full Node2D property dump) | 744 bytes (~186 tok) | 932 bytes (~233 tok) |
| `node find` (all)         |   83 bytes (~20 tok) | 133 bytes (~33 tok) |

Compact is ~20–30% smaller than pretty on these calls, and `--ids` cuts a tree
or find result down to one path per line.

High-volume UI reads should be scoped. On the Memory Match prompt scene, the
full runtime Control tree was 7,858 chars (~1,965 tok), while this focused read:

```sh
./hera game ui tree --type Button --fields name,path,text,disabled
```

returned 2,803 chars (~701 tok). For one exact control, this is smaller still:

```sh
./hera game ui tree --text Restart --fields name,path,text,rect,disabled
```

Use the narrowest command that answers the question:

| Need | Prefer |
|------|--------|
| Confirm editor scene structure | `./hera --ids scene tree` |
| Inspect one editor property | `./hera node get . --prop script` |
| Inspect a few editor properties | `./hera node get . --props visible,position` |
| Find clickable runtime controls | `./hera game ui tree --type Button --fields name,path,text,disabled` |
| Inspect one runtime control | `./hera game ui tree --text Restart --fields name,path,text,rect,disabled` |
| Discover deterministic QA helpers | `./hera game qa discover` |
| Check layout/clipping visually | `./hera game screenshot --analyze` |

---

## Caveats (so the numbers stay honest)

- **Hera figures are measured** on a live Godot 4.7 editor; **MCP figures are
  estimated** from public tool counts × an assumed per-tool schema size. We did
  not run the MCP servers.
- **Tool count means MCP tools, not internal operations.** One sampled server
  documents 120+ operations across ~41 MCP tools; the table uses ~41 for its
  resident schema estimate.
- Token counts use a **`chars / 4`** approximation (no model tokenizer). It is
  fine for relative scale, not for exact billing. JSON tends to tokenize a little
  denser than 4 chars/token, so these slightly *under*count both sides.
- **Prompt caching** lowers the per-call cost of resident tokens for both sides
  on a cache hit. The structural point survives caching: MCP schemas grow with
  every tool *and* sit in each turn; Hera docs grow with the command surface
  but are optional, cacheable reads rather than a mandatory tool list.
- **`status` fixture sizes** are the contract goldens in this tree, including
  the trailing newline. They are not every project's live `status` length
  (`project_path` and similar fields vary).
- This is about **context economy, not a verdict on MCP.** MCP buys native,
  one-click client integration (Claude Code, Cursor, Codex). Hera trades that for
  the token budget and for working with anything that can run a shell command.

---

## Reproduce it

With a Godot 4.7 editor running the Hera Agent plugin and `hera` built
(`go build -o hera .`):

```sh
# per-operation response sizes (compact vs pretty)
for cmd in "status" "scene tree" "node get ." "node find" \
  "game ui tree" \
  "game ui tree --type Button --fields name,path,text,disabled" \
  "game qa discover"; do
  c=$(./hera $cmd | wc -c)
  p=$(./hera --json $cmd | wc -c)
  printf "%-16s compact=%4d chars  pretty=%4d chars\n" "$cmd" "$c" "$p"
done

# fixed surface doc size
wc -c AGENTS.md docs/COMMANDS.md
```

Approximate tokens by dividing chars by 4. The MCP side is arithmetic:
documented MCP tool count × ~100–200 tokens per tool schema.
