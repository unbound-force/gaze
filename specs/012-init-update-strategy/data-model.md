# Data Model: Init Update Strategy

**Branch**: `012-init-update-strategy` | **Date**: 2026-03-01

## Entities

### Options (existing, unchanged)

Configuration for the scaffold operation. No changes required.

| Field     | Type       | Description                                                |
|-----------|------------|------------------------------------------------------------|
| TargetDir | string     | Root directory to scaffold into. Defaults to cwd.          |
| Force     | bool       | Unconditionally overwrite all files when true.             |
| Version   | string     | Gaze version string for the version marker. Defaults "dev".|
| Stdout    | io.Writer  | Writer for summary output. Defaults to os.Stdout.          |

### Result (modified)

Reports what the scaffold operation did. The `Skipped` field is removed and replaced with `Updated`, `UpToDate`, and `UpdatedFrom`.

#### Current Schema

| Field       | Type       | Description                                    |
|-------------|------------|------------------------------------------------|
| Created     | []string   | Files written for the first time.              |
| Skipped     | []string   | Files that existed and were not overwritten.    |
| Overwritten | []string   | Files that existed and were replaced (--force). |

#### New Schema

| Field       | Type              | Description                                                        |
|-------------|-------------------|--------------------------------------------------------------------|
| Created     | []string          | Files written for the first time (did not exist on disk).          |
| Updated     | []string          | Files that existed with an outdated version marker and were replaced.|
| UpToDate    | []string          | Files that existed with a current version marker and were left untouched.|
| Overwritten | []string          | Files that existed and were unconditionally replaced (--force).    |
| UpdatedFrom | map[string]string | Maps relative path to previous version string for each updated file.|

#### State Transition Rules

For each file in the embedded asset manifest:

```
File does not exist on disk
  → Write file with current content and version marker
  → Classify as "Created"

File exists, --force is set
  → Overwrite file unconditionally
  → Classify as "Overwritten"

File exists, running version is "dev"
  → Read first line, extract old version (may be "dev" or any string)
  → Overwrite file with current content and "dev" marker
  → Classify as "Updated", record old version in UpdatedFrom

File exists, version marker matches running version
  → Leave file untouched
  → Classify as "UpToDate"

File exists, version marker differs from running version (or missing/unparseable)
  → Read first line, extract old version (or "(unknown)" if missing)
  → Overwrite file with current content and new version marker
  → Classify as "Updated", record old version in UpdatedFrom
```

### Version Marker (implicit entity)

Not a struct — a string convention embedded as the first line of each scaffolded file.

| Attribute | Value                                    |
|-----------|------------------------------------------|
| Format    | `<!-- scaffolded by gaze VERSION -->`    |
| Location  | First line of scaffolded file            |
| Prefix    | `<!-- scaffolded by gaze `               |
| Suffix    | ` -->`                                   |

#### extractVersion() Behavior

| Input (first line)                       | Output            |
|------------------------------------------|-------------------|
| `<!-- scaffolded by gaze v1.0.0 -->`     | `"v1.0.0"`        |
| `<!-- scaffolded by gaze dev -->`        | `"dev"`           |
| `<!-- scaffolded by gaze  -->`           | `""` (empty)      |
| `# Some other content`                  | `""` (empty)      |
| (empty file)                             | `""` (empty)      |

When `extractVersion()` returns empty string, the file is treated as having no recognizable marker (FR-006: treat as outdated).

## Relationships

```
Options --[configures]--> Run()
Run()   --[produces]----> Result
Result  --[contains]----> Created, Updated, UpToDate, Overwritten (file lists)
Result  --[contains]----> UpdatedFrom (version provenance per updated file)
```

No external entities, no database, no network resources.
