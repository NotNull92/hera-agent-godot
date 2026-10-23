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

Sizes below are **UTF-8 byte counts** after the Wave 2 review follow-ups
(2026-10-23, Git LF content), not tokenizer output. Approximate tokens as bytes/4.

| Hera surface (one-time, cacheable) | Bytes | ~tok |
|------------------------------------|------:|-----:|
| `plugins/hera-godot/skills/live-editor/SKILL.md` | 5,308 | ~1,327 |
| `AGENTS.md` (driver card) | 4,496 | ~1,124 |
| `docs/COMMANDS.md` (full reference) | 77,481 | ~19,370 |
| `docs/AGENT_GUIDE.md` (detailed operating rules) | 21,114 | ~5,278 |

So before any work is done, an MCP client in the sampled servers still pays
**~4k–31k resident tokens every turn** for tool schemas. Hera pays **nothing
per turn** for schemas. Loading the full command reference is no longer a ~1k
token footnote — that older claim is retired. Prefer the live-editor skill or
`AGENTS.md` for routine driving, and open `COMMANDS.md` when a flag or field
is in doubt.

---

## 2. Per-operation cost — the response to each action

Hera defaults to **compact JSON** (`--json` pretty-prints; `--ids` trims to
paths). Sizes below use the updated 2.1 contract goldens in this tree: UTF-8
bytes with LF and the trailing newline. Pretty sizes use two-space indentation
of the same payload; status also has a dedicated pretty golden. These are fixed
fixtures, not a claim about every project's response size.

| Operation | compact (default) | `--json` (pretty) |
|-----------|------------------:|------------------:|
| `status` | 199 bytes (~50 tok) | 228 bytes (~57 tok) |
| `status --capabilities` | 809 bytes (~202 tok) | 964 bytes (~241 tok) |
| `scene tree` (1-node demo) | 88 bytes (~22 tok) | 134 bytes (~34 tok) |
| `node get .` (full Node2D dump) | 744 bytes (~186 tok) | 912 bytes (~228 tok) |
| `node find` | 55 bytes (~14 tok) | 97 bytes (~24 tok) |

`--ids` reduces tree/find output to one path per line. The 2.1 omissions apply
to both JSON modes; see [the migration table](MIGRATING_TO_V2_1.md).

High-volume UI reads should be scoped. On the Memory Match prompt scene, the
full runtime Control tree was 7,858 chars (~1,965 tok), while this focused read:

```sh
./hera-godot game ui tree --type Button --fields name,path,text,disabled
```

returned 2,803 chars (~701 tok). For one exact control, this is smaller still:

```sh
./hera-godot game ui tree --text Restart --fields name,path,text,rect,disabled
```

Use the narrowest command that answers the question:

| Need | Prefer |
|------|--------|
| Confirm editor scene structure | `./hera-godot --ids scene tree` |
| Inspect one editor property | `./hera-godot node get . --prop script` |
| Inspect a few editor properties | `./hera-godot node get . --props visible,position` |
| Find clickable runtime controls | `./hera-godot game ui tree --type Button --fields name,path,text,disabled` |
| Inspect one runtime control | `./hera-godot game ui tree --text Restart --fields name,path,text,rect,disabled` |
| Discover deterministic QA helpers | `./hera-godot game qa discover` |
| Check layout/clipping visually | `./hera-godot game screenshot --analyze` |

---

## Caveats (so the numbers stay honest)

- **Hera per-call table uses contract fixtures**; historical UI examples below
  were measured on a live Godot 4.7 editor; **MCP figures are
  estimated** from public tool counts × an assumed per-tool schema size. We did
  not run the MCP servers.
- **Tool count means MCP tools, not internal operations.** One sampled server
  documents 120+ operations across ~41 MCP tools; the table uses ~41 for its
  resident schema estimate.
- Token counts use a **`bytes / 4`** approximation (no model tokenizer). It is
  fine for relative scale, not for exact billing. JSON tends to tokenize a little
  denser than 4 chars/token, so these slightly *under*count both sides.
- **Prompt caching** lowers the per-call cost of resident tokens for both sides
  on a cache hit. The structural point survives caching: MCP schemas grow with
  every tool *and* sit in each turn; Hera docs grow with the command surface
  but are optional, cacheable reads rather than a mandatory tool list.
- **Per-call fixture sizes** are the contract goldens in this tree, including
  the trailing newline. They are not every project's live `status` length
  (`scene` and `project_name` vary).
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
  c=$(./hera-godot $cmd | wc -c)
  p=$(./hera-godot --json $cmd | wc -c)
  printf "%-16s compact=%4d chars  pretty=%4d chars\n" "$cmd" "$c" "$p"
done

# fixed surface doc size
wc -c AGENTS.md docs/COMMANDS.md
```

Approximate tokens by dividing chars by 4. The MCP side is arithmetic:
documented MCP tool count × ~100–200 tokens per tool schema.
