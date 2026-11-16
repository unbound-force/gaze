# Document Annotations

For [external-analyzer projects](../porting/contracts.md) — projects analyzed through a language adapter via `--analyzer` rather than Gaze's native Go engine — Gaze can derive [classification](../concepts/classification.md) signals directly from your documentation. When an external analyzer reports an effect as `ambiguous`, annotations in your Markdown docs or a sidecar file can push it toward `contractual` or `incidental`.

This is **opt-in and external-analyzer-only**: annotations are only read when an external analyzer is in use. Go-native projects use the built-in GoDoc analyzer and are unaffected.

## Why Annotate Docs?

External analyzers don't have access to Go-specific signals like interface satisfaction or GoDoc comments. Design docs often state intent explicitly ("`Store.Save` persists state"), but that intent never reaches classification. Document annotations bridge that gap: they let you declare, in a deterministic and reviewable way, which effects are part of a function's contract and which are implementation details.

## Annotation Grammar

There are two syntax forms, both keyed on the analyzer-emitted effect type name (e.g. `ContainerMutation`, `StreamOutput`) and a `contractual` or `incidental` label.

### 1. Inline HTML Comments

HTML comments are invisible in every Markdown renderer — your docs stay clean while carrying machine-readable declarations:

```markdown
<!-- gaze:contractual ContainerMutation -->
<!-- gaze:incidental CallbackInvocation -->
```

The form is `<!-- gaze:<label> <TypeName> -->`. Label matching is case-insensitive; the type name is captured verbatim.

### 2. Fenced Block Callouts

Fenced block callouts render as styled blocks in GitHub/GitLab, making contract declarations visible and reviewable in pull requests:

```markdown
> [!gaze-contract]
> - `ContainerMutation` — contractual (public API mutation)
> - `CallbackInvocation` — incidental (internal wiring)
```

Each entry is ``> - `TypeName` — contractual|incidental``, optionally followed by a human-readable note after the label. Entries are separated by newlines; a line starting with `>` that is not an entry keeps the block open, while a non-`>` line ends the block.

## Sidecar File

For per-function precision — grammar annotations are type-level wildcards that fan out to every function with that effect — use a sidecar file at `.uf/gaze/contracts.yaml` (or `.uf/gaze/contracts.json` for tool-generated sidecars).

```yaml
# .uf/gaze/contracts.yaml
version: 1
contracts:
  - package: "mypackage"
    function: "process_items"
    side_effect_type: "ContainerMutation"
    label: contractual
    reasoning: "public API mutation per design doc"  # optional
  - package: "mypackage"
    function: "process_items"
    side_effect_type: "StreamOutput"
    label: incidental
```

The JSON format is structurally equivalent:

```json
{
  "version": 1,
  "contracts": [
    {
      "package": "mypackage",
      "function": "process_items",
      "side_effect_type": "ContainerMutation",
      "label": "contractual",
      "reasoning": "public API mutation per design doc"
    }
  ]
}
```

**Fields**:

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `version` | `int` | No | Schema version (reserved) |
| `package` | `string` | Yes | Package the function belongs to |
| `function` | `string` | Yes | The analyzer's verbatim function name (bare name, not receiver-qualified) |
| `side_effect_type` | `string` | Yes | A known effect type name (must be in the [taxonomy](../porting/taxonomy-reference.md)) |
| `label` | `string` | Yes | `contractual` or `incidental` (case-insensitive) |
| `reasoning` | `string` | No | Human-readable justification; defaults to a generated message when omitted |

Entries that fail semantic validation — an unknown `label`, an empty `package` or `function`, or an unknown `side_effect_type` — are skipped with a warning to stderr. The remaining valid entries are still loaded.

## Source Classification Rules

Every derived signal carries a source token and a signed weight. The source and weight are determined by where the annotation was found:

| Source | Token | Weight | Applies To |
|--------|-------|--------|------------|
| Architecture doc | `architecture_doc` | ±25 | Non-README Markdown documents (design docs, ADRs, etc.) |
| README | `readme` | ±15 | Documents whose basename starts with `README` (case-insensitive) |
| Sidecar | `sidecar` | ±30 | Entries in `.uf/gaze/contracts.yaml` / `.json` |

A `contractual` label produces a **positive** weight; an `incidental` label produces a **negative** weight. For example, an `incidental` annotation in a design doc carries `-25`, while a `contractual` sidecar entry carries `+30`.

Non-README docs carry higher weight than READMEs because design docs are typically more authoritative about intent. Sidecar weight is highest, reflecting explicit, intentional declarations.

## Precedence

Two rules govern which declaration wins when multiple sources annotate the same type:

1. **Sidecar overrides grammar** — a sidecar entry for a `(package, function, side_effect_type)` tuple discards the grammar-derived signal for that tuple. Sidecar declarations are explicit and intentional, so they carry the highest authority.
2. **Root-level docs win over nested docs** — when grammar annotations conflict for the same type name, a root-level document (`PriorityModuleRoot`) wins over a nested document (`PriorityOther`).

Note that precedence is per-tuple: a sidecar entry overrides grammar *only* for the specific function it names. Grammar annotations continue to apply to all other functions with the same effect type.

## Scope and Opt-In Behavior

- **External analyzers only**: doc-signal derivation runs only in the external-analyzer path (`--analyzer`). Go-native analysis is unaffected.
- **Opt-in**: unannotated docs and an absent sidecar file produce no signals. Classification works without them — it simply produces more `ambiguous` results.
- **Graceful degradation**: a missing sidecar, malformed sidecar, or malformed annotation never blocks analysis. Failing steps degrade to whatever valid signals remain.

## See Also

- [Improving Scores](improving-scores.md) — how classification signals affect CRAP and GazeCRAP scores
- [Classification](../concepts/classification.md) — how signals contribute to confidence scoring
- [Configuration Reference](../reference/configuration.md) — `.uf/gaze/contracts.yaml` schema and precedence
