# gaze schema

Print the JSON Schema (Draft 2020-12) that documents the structure of `gaze analyze --format=json` output. Useful for validating output programmatically or generating client types in other languages.

## Synopsis

```text
gaze schema [flags]
```

## Arguments

This command takes no positional arguments.

## Flags

This command has no flags beyond the standard `--help`.

| Flag | Short | Type | Description |
|------|-------|------|-------------|
| `--help` | `-h` | `bool` | Help for schema |

## Output

The command prints the complete JSON Schema to stdout. The schema is embedded in the Gaze binary at build time (from `internal/report/schema.go`).

The schema defines:

- **Top-level object**: `version` (string) and `results` (array of `AnalysisResult`)
- **AnalysisResult**: `target` (function metadata), `side_effects` (array), `metadata` (timing/version)
- **SideEffect**: `id`, `type` (one of 37 effect types), `tier` (P0–P4), `location`, `description`, `target`, and optional `classification`
- **Classification**: `label` (contractual/incidental/ambiguous), `confidence` (0–100), `signals` (array), `reasoning`

See [JSON Schemas](../json-schemas.md) for annotated field descriptions and example output.

## Examples

### Print the schema

```bash
gaze schema
```

### Validate analyze output against the schema

```bash
# Using ajv-cli (npm install -g ajv-cli)
gaze schema > /tmp/gaze-schema.json
gaze analyze ./internal/crap --format=json > /tmp/output.json
ajv validate -s /tmp/gaze-schema.json -d /tmp/output.json
```

### Generate TypeScript types

```bash
# Using json-schema-to-typescript (npm install -g json-schema-to-typescript)
gaze schema | json2ts > gaze-types.ts
```

## See Also

- [JSON Schemas](../json-schemas.md) — annotated schema reference for all JSON-producing commands
- [`gaze analyze`](analyze.md) — the command whose output this schema describes
