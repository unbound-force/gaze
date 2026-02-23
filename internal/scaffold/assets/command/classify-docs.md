---
description: >
  Document-enhanced classification for a Go package. Runs mechanical
  classification via `gaze analyze --classify --format=json`, scans
  project documentation via `gaze docscan`, and feeds both to the
  doc-classifier agent to produce an enhanced classification JSON.
agent: doc-classifier
---

# Command: /classify-docs

## Description

Enhance Gaze's mechanical side effect classifications with signals
extracted from project documentation. Produces the same JSON schema
as `gaze analyze --classify --format=json` but with additional
document and AI inference signals and recalculated confidence scores.

## Usage

```
/classify-docs <package-pattern>
```

Examples:

```
/classify-docs ./internal/store
/classify-docs github.com/myorg/myrepo/pkg/service
```

## Instructions

1. **Build the binary** (ensures it reflects the current code):
   ```bash
   go build -o /tmp/gaze-classify-docs ./cmd/gaze
   ```

2. **Run mechanical classification**:
   ```bash
   /tmp/gaze-classify-docs analyze --classify --format=json $ARGUMENTS
   ```
   Capture the JSON output as `MECHANICAL_JSON`.

3. **Scan project documentation**:
   ```bash
   /tmp/gaze-classify-docs docscan $ARGUMENTS
   ```
   Capture the JSON output (array of `{path, content, priority}`) as
   `DOCS_JSON`.

4. **Feed both to the doc-classifier agent**:
   Pass the following to the `doc-classifier` agent as its prompt:

   ```
   ## Mechanical Classification JSON
   <paste MECHANICAL_JSON here>

   ## Documentation Content
   <paste DOCS_JSON here>

   Enhance the mechanical classifications using the document signals.
   Output the full enhanced classification JSON.
   ```

5. **Output** the agent's enhanced classification JSON to stdout.

## Error Handling

- If `gaze analyze` fails (e.g., build errors, package not found),
  report the error and stop.
- If `gaze docscan` finds no documents, pass an empty array `[]` to
  the agent; the agent will return the mechanical-only results
  unchanged with a warning in `metadata.warnings`.
- If the agent produces invalid JSON, report the schema validation
  error and suggest running `gaze analyze --classify --format=json`
  directly for mechanical-only results.

## Notes

- This command requires `gaze` to be buildable from the current
  working directory (i.e., `go build ./cmd/gaze` must succeed).
- Document scanning respects `.gaze.yaml` exclude/include patterns
  if the file is present.
- The agent is read-only and does not modify any source files.
