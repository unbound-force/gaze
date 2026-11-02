// Package scaffold embeds distributable OpenCode agent and command
// files, writes them to a target project directory, optionally
// generates the CI AI-report caller workflow for `gaze init --ci`,
// registers the divisor-gaze-testing persona in existing
// unbound-force .uf review configuration, and removes superseded
// gaze-scaffolded orphans.
package scaffold

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/mod/semver"
)

//go:embed assets
var assets embed.FS

// Options configures the scaffold operation.
type Options struct {
	// TargetDir is the root directory to scaffold into.
	// Defaults to the current working directory.
	TargetDir string

	// Force overwrites existing files when true.
	// When false, existing files are skipped.
	Force bool

	// Version is the gaze version string to embed in the
	// version marker comment. Set by ldflags at build time.
	// Defaults to "dev" for development builds.
	Version string

	// CI requests CI scaffolding (the AI-report caller workflow)
	// in addition to the .opencode/ assets. The zero value (false)
	// means no CI scaffolding: the gaze version is never validated
	// and no CI artifacts are written.
	CI bool

	// GazeVersion overrides the version used for CI scaffolding,
	// taking precedence over the build-time Version. The zero value
	// (empty) means Version is used. When CI scaffolding is
	// requested, the resolved value must be a gaze release tag >=
	// minWorkflowTag (see resolveGazeVersion); without CI it is
	// ignored.
	GazeVersion string

	// Stdout is the writer for summary output.
	// Defaults to os.Stdout.
	Stdout io.Writer
}

// Result reports what the scaffold operation did.
type Result struct {
	// Created lists files that were written for the first time.
	Created []string

	// Skipped lists files that already existed and were not
	// overwritten (Force was false).
	Skipped []string

	// Overwritten lists files that existed and were replaced
	// (Force was true).
	Overwritten []string

	// Updated lists tool-owned files that existed with different
	// content and were overwritten via overwrite-on-diff (Force
	// was false, but content differed from the embedded version).
	Updated []string

	// Registered lists .uf review-config files that the
	// divisor-gaze-testing persona registration was merged into.
	// Only projects that already use unbound-force (have a .uf
	// directory) are registered; gaze never creates .uf files.
	Registered []string

	// Removed lists legacy scaffold files deleted as superseded
	// orphans: gaze-marked .opencode/agents/reviewer-testing.md
	// copies replaced by agents/divisor-gaze-testing.md. Marker-less
	// (user-customized) copies are never removed.
	Removed []string
}

// isToolOwned reports whether the given relative path (under
// .opencode/) is a tool-owned file. Tool-owned files use
// overwrite-on-diff behavior: they are replaced when their
// content differs from the embedded version, even without --force.
// User-owned files (only commands/gaze.md today) retain
// skip-if-present behavior.
//
// Ownership is determined by an explicit list: all files under
// references/ are tool-owned by directory convention, and specific
// command files are tool-owned by exact match. This approach is
// necessary because the commands/ directory contains both
// user-owned files (gaze.md) and tool-owned files
// (speckit.testreview.md).
func isToolOwned(relPath string) bool {
	if strings.HasPrefix(relPath, "references/") {
		return true
	}
	switch relPath {
	case "commands/speckit.testreview.md",
		"agents/gaze-reporter.md",
		"agents/gaze-test-generator.md",
		"agents/divisor-gaze-testing.md",
		"commands/gaze-fix.md",
		"dcp.jsonc":
		return true
	}
	return false
}

// versionMarker returns the version marker comment to embed in
// scaffolded Markdown files. Non-Markdown files skip the marker
// (see processAssetFile).
func versionMarker(version string) string {
	if version == "" {
		version = "dev"
	}
	return fmt.Sprintf("<!-- scaffolded by gaze %s -->\n", version)
}

// isMarkerLine reports whether line is a gaze version-marker comment
// ("<!-- scaffolded by gaze ... -->"), ignoring surrounding
// whitespace. Marker detection (orphan cleanup) and marker removal
// (re-scaffolding) share this single predicate so the two cannot
// drift.
func isMarkerLine(line string) bool {
	const prefix = "<!-- scaffolded by gaze "
	const suffix = "-->"
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, prefix) && strings.HasSuffix(trimmed, suffix)
}

// removeExistingMarker strips any "<!-- scaffolded by gaze ... -->"
// line from content. This prevents marker duplication when
// re-scaffolding with a different version (issue #279).
func removeExistingMarker(content []byte) []byte {
	lines := strings.Split(string(content), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if isMarkerLine(line) {
			continue
		}
		out = append(out, line)
	}
	return []byte(strings.Join(out, "\n"))
}

// insertMarkerAfterFrontmatter inserts the version marker after
// the YAML frontmatter closing delimiter (---). If no frontmatter
// is found, the marker is appended to the end of the content.
//
// Any existing "<!-- scaffolded by gaze ... -->" marker is removed
// first to prevent duplication when re-scaffolding with a different
// version.
//
// YAML frontmatter must start with "---\n" on the very first line
// and end with "\n---\n". Prepending a marker before the opening
// "---" breaks frontmatter parsing in tools like OpenCode, which
// causes agent tool configurations and command delegation to be
// silently ignored.
//
// Constraint: YAML frontmatter content MUST NOT contain a line
// that is exactly "---". If it does, the first such line will be
// incorrectly treated as the closing delimiter. All current
// embedded assets satisfy this constraint.
func insertMarkerAfterFrontmatter(content []byte, marker string) []byte {
	// Strip any existing version marker to avoid duplication when
	// re-scaffolding with a different version (issue #279).
	content = removeExistingMarker(content)
	s := string(content)

	// Check for YAML frontmatter: must start with "---\n".
	if !strings.HasPrefix(s, "---\n") {
		// No frontmatter — append marker at the end.
		return append(content, []byte(marker)...)
	}

	// Find the closing "---" delimiter (skip the opening one).
	closeIdx := strings.Index(s[4:], "\n---\n")
	if closeIdx < 0 {
		// Malformed frontmatter — append marker at the end.
		return append(content, []byte(marker)...)
	}

	// Insert marker after the closing "---\n".
	// closeIdx is relative to s[4:], so the absolute position of
	// the closing "\n---\n" is closeIdx+4. The line ends at
	// closeIdx+4+len("\n---\n") = closeIdx+8.
	insertAt := closeIdx + 4 + len("\n---\n")
	out := make([]byte, 0, len(content)+len(marker))
	out = append(out, content[:insertAt]...)
	out = append(out, []byte(marker)...)
	out = append(out, content[insertAt:]...)
	return out
}

// applyDefaults sets zero-valued Options fields to their defaults.
// CI and GazeVersion deliberately receive no defaults: CI's zero
// value means "no CI scaffolding" (opt-in only), and an empty
// GazeVersion defers to Version during resolution (see
// resolveGazeVersion).
func applyDefaults(opts *Options) error {
	if opts.TargetDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
		opts.TargetDir = cwd
	}
	if opts.Version == "" {
		opts.Version = "dev"
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	return nil
}

// minWorkflowTag is the minimum gaze release tag accepted when
// validating the CI scaffolding version pin: the first release tag
// containing the reusable AI-report workflow
// (.github/workflows/reusable_gaze_ai_report.yml). Release tags are
// immutable, so the floor never moves — older tags cannot gain the
// file.
const minWorkflowTag = "v2.3.0"

// strictReleaseTagPattern is the strict-form gate layered on top of
// semver.IsValid: a full vMAJOR.MINOR.PATCH with an optional
// pre-release suffix and no build metadata. Bare semver.IsValid is
// insufficient here because golang.org/x/mod/semver accepts partial
// versions — semver.IsValid("v2.3") is true and
// semver.Compare("v2.3", "v2.3.0") == 0 (missing minor/patch
// components default to 0) — so a partial form would slip past the
// floor comparison and pin the generated `uses:` reference to a tag
// that does not exist (gaze release tags are always full
// major.minor.patch). The pre-release body's character class is
// deliberately permissive; semver.IsValid stays in the validation
// chain to reject malformed pre-release identifiers (e.g. empty
// dot-separated segments) that the class alone would let through.
// Build metadata is rejected because gaze release tags never carry
// it.
var strictReleaseTagPattern = regexp.MustCompile(`^v\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`)

// resolveGazeVersion resolves the gaze release tag for CI
// scaffolding — the GazeVersion override when set, otherwise the
// build-time Version — and validates it against minWorkflowTag. It
// is a pure function of opts (no filesystem access) so the
// version-resolution matrix can exercise it as a table-driven test.
//
// A version passes only when it clears all three layers: the strict
// release-tag form above, semver.IsValid, and
// semver.Compare(v, minWorkflowTag) >= 0. The floor is inclusive;
// pre-releases of the floor itself (e.g. v2.3.0-rc1) sort below it
// by semver precedence and are rejected, while pre-releases above
// the floor (e.g. v2.4.0-rc1) are accepted. Any invalid, pre-floor,
// dev, or empty resolution fails closed with an error naming the
// floor and directing the user to install gaze from a release tag
// or pass --gaze-version.
func resolveGazeVersion(opts Options) (string, error) {
	v := opts.Version
	if opts.GazeVersion != "" {
		v = opts.GazeVersion
	}
	if !strictReleaseTagPattern.MatchString(v) ||
		!semver.IsValid(v) ||
		semver.Compare(v, minWorkflowTag) < 0 {
		return "", fmt.Errorf(
			"gaze version %q cannot pin the CI workflow: the reusable AI-report workflow first ships in %s; "+
				"install gaze from a release tag (>= %s) or pass --gaze-version",
			v, minWorkflowTag, minWorkflowTag)
	}
	return v, nil
}

// resolveCIPin resolves the gaze release tag that pins the generated
// workflow's uses: reference (design D4): the Options.GazeVersion
// override when set, else the build-time Options.Version, validated
// by resolveGazeVersion against minWorkflowTag. It is explicitly
// gated on opts.CI — plain init runs never validate the version, so
// their behavior is unchanged by the CI feature. Run calls it before
// the go.mod existence check so a failed --ci invocation writes
// nothing, and the combined dev-build-without-go.mod case fails
// closed on the version error rather than taking the missing-go.mod
// skip path (design D6 check ordering). A plain run resolves to the
// empty string, which completeCIContext ignores.
func resolveCIPin(opts Options) (string, error) {
	if !opts.CI {
		return "", nil
	}
	return resolveGazeVersion(opts)
}

// defaultGoVersion is the Go toolchain version substituted into the
// generated CI workflow when the target's go.mod yields no usable `go`
// directive (absent, unreadable, or outside the accepted grammar). It
// matches the reusable AI-report workflow's own go-version input
// default, so a fallback workflow still runs the Go version the
// reusable workflow would have chosen for itself (design D5).
const defaultGoVersion = "1.25"

// goDirectivePattern is the grammar every Go version substituted into
// the generated CI workflow must match: major[.minor[.patch]] with an
// optional pre-release suffix (e.g. 1.25, 1.24, 1.25.0, 1.25.0-rc1)
// — the version forms actions/setup-go accepts. The go.mod directive
// value is validated against this pattern and never passed through
// verbatim: an unvalidated value could break the generated workflow —
// a value containing a double quote terminates the double-quoted
// go-version YAML scalar early, a raw CR from a CRLF go.mod undergoes
// YAML flow folding, and anything else surfaces as an opaque setup-go
// failure at CI run time, the delayed-failure class this feature
// exists to eliminate (design D5).
var goDirectivePattern = regexp.MustCompile(`^\d+(\.\d+)?(\.\d+)?(-[0-9A-Za-z.-]+)?$`)

// parseGoModVersion extracts the Go version from go.mod content with
// a minimal line scan for a line beginning "go " — deliberately not
// modfile.Parse, which would couple CI scaffolding to unrelated
// go.mod validity (a malformed replace directive must not break gaze
// init; design D5). It is a pure function of the content so the
// directive matrix can exercise it as a table-driven test; file I/O
// is kept separate in readGoModVersion.
//
// Trailing whitespace and a trailing \r (CRLF go.mod) are stripped
// before matching. The first line beginning "go " decides the
// outcome: when its value matches goDirectivePattern it is returned
// unchanged; when it does not, or no such line exists,
// defaultGoVersion is returned with fellBack set. The scan does not
// continue past a non-matching directive line — go.mod admits a
// single go directive, so a later duplicate cannot be authoritative,
// and preferring it would guess at file validity this scan
// deliberately avoids. The returned version always satisfies the
// grammar.
func parseGoModVersion(content string) (version string, fellBack bool) {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimRight(line, " \t\r")
		if !strings.HasPrefix(line, "go ") {
			continue
		}
		v := strings.TrimSpace(strings.TrimPrefix(line, "go "))
		if goDirectivePattern.MatchString(v) {
			return v, false
		}
		break
	}
	return defaultGoVersion, true
}

// readGoModVersion resolves the CI workflow's Go version from the
// go.mod at path, keeping file I/O out of the pure parseGoModVersion.
// A read failure is an unreadable directive, not an error: design D5
// falls back to defaultGoVersion — reported via fellBack so the
// caller can warn — rather than failing the run.
func readGoModVersion(path string) (version string, fellBack bool) {
	content, err := os.ReadFile(path)
	if err != nil {
		return defaultGoVersion, true
	}
	return parseGoModVersion(string(content))
}

// ciContext carries the resolved CI-scaffolding inputs from Run's
// pre-walk resolution phase to the workflow write path (design D7,
// task 1.4) and the console next-steps section (design D10, task
// 1.5). It is populated only when Options.CI is set; the zero value
// means no CI scaffolding was requested.
type ciContext struct {
	// gazeVersion is the validated gaze release tag for the
	// workflow's uses: pin and gaze-version input (design D4).
	gazeVersion string

	// goVersion is the grammar-validated Go toolchain version for the
	// workflow's setup-go step (design D5). It always matches
	// goDirectivePattern: defaultGoVersion whenever the go.mod
	// directive was absent, unreadable, or invalid.
	goVersion string

	// skipped is true when CI scaffolding was requested but skipped
	// because the target has no go.mod at the repository root (design
	// D6). The skip is non-fatal — the .opencode/ scaffolding
	// proceeds — and suppresses the CI write path and the next-steps
	// section.
	skipped bool
}

// printCISkipWarning prints the design-D6 warning for a --ci run in a
// repository without a Go module at the root. It names the
// requirement (the reusable workflow analyzes ./... from the
// repository root) and the remedy (create the module, re-run), and it
// stays distinct from the generic missing-go.mod warning that
// precedes it in Run's output.
func printCISkipWarning(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Warning: skipping CI scaffolding (--ci): the reusable gaze AI-report workflow analyzes ./... from the repository root and requires a Go module there.")
	_, _ = fmt.Fprintln(w, "Run `go mod init` at the repository root, then re-run `gaze init --ci` to generate the workflow.")
	_, _ = fmt.Fprintln(w, "The .opencode/ scaffolding is unaffected.")
}

// checkGoMod locates the target directory's go.mod, printing the
// general missing-go.mod warning when it is absent. The warning
// applies to every Run invocation, CI or plain; the returned absent
// flag also feeds the CI context, where a missing module skips CI
// scaffolding with a distinct, CI-specific warning (design D6, see
// completeCIContext). A stat error other than fs.ErrNotExist is not
// an absence — the caller proceeds as if the module exists.
func checkGoMod(opts Options) (path string, absent bool) {
	path = filepath.Join(opts.TargetDir, "go.mod")
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		_, _ = fmt.Fprintln(opts.Stdout, "Warning: no go.mod found in current directory.")
		_, _ = fmt.Fprintln(opts.Stdout, "Gaze works best in a Go module root.")
		_, _ = fmt.Fprintln(opts.Stdout)
		return path, true
	}
	return path, false
}

// completeCIContext resolves the remaining CI-scaffolding inputs
// (designs D5 and D6), gated on opts.CI so plain init output is
// unchanged; a plain run gets the zero ciContext, and the gazeVersion
// pin resolved by resolveCIPin is dropped with it. The reusable
// AI-report workflow analyzes ./... from the repository root, so a
// missing go.mod skips CI scaffolding with a distinct, non-fatal
// warning (printCISkipWarning) and the .opencode/ asset walk proceeds
// unchanged. With a go.mod present, the workflow's Go version is read
// from its `go` directive via readGoModVersion — validated before
// substitution, never passed through verbatim — falling back to
// defaultGoVersion with a warning when the directive is absent,
// unreadable, or outside the accepted grammar. The resolved context
// feeds the workflow write path (task 1.4) and the next-steps section
// (task 1.5).
func completeCIContext(opts Options, goModPath string, goModAbsent bool, gazeVersion string) ciContext {
	if !opts.CI {
		return ciContext{}
	}
	ci := ciContext{gazeVersion: gazeVersion}
	if goModAbsent {
		ci.skipped = true
		printCISkipWarning(opts.Stdout)
		return ci
	}
	version, fellBack := readGoModVersion(goModPath)
	if fellBack {
		_, _ = fmt.Fprintf(opts.Stdout,
			"Warning: no readable `go` directive in go.mod; the generated CI workflow will use Go %s (the reusable workflow's default).\n",
			defaultGoVersion)
		_, _ = fmt.Fprintln(opts.Stdout, "Fix the directive and re-run `gaze init --ci` to pin your module's Go version.")
	}
	ci.goVersion = version
	return ci
}

// ciWorkflowPath is the generated CI workflow's location relative to
// the target project root (design D1): a self-contained caller for
// the reusable AI-report workflow. It doubles as the display path in
// the scaffold Result so printSummary reports the workflow through
// the existing created/skipped/overwritten loops with no new plumbing
// (design D7).
const ciWorkflowPath = ".github/workflows/gaze-ai-report.yml"

// ciWorkflowTemplatePath is the embedded CI workflow template
// (design D9), read directly by the dedicated CI write path. It lives
// in the ci/ subtree that Run's asset walk and assetPaths exclude
// (design D8) — the template is consumed by writeCIWorkflow, never
// mirrored into .opencode/.
const ciWorkflowTemplatePath = "assets/ci/gaze-ai-report.yml"

// CI workflow template placeholders (design D9), substituted by
// renderCIWorkflowTemplate.
const (
	gazeVersionPlaceholder = "__GAZE_VERSION__"
	goVersionPlaceholder   = "__GO_VERSION__"
)

// isCIAsset reports whether relPath (relative to the embedded assets
// root) names an entry in the ci/ subtree. That subtree holds the CI
// workflow template, which is consumed by the dedicated CI write path
// rather than mirrored into .opencode/: both Run's asset walk and
// assetPaths skip ci/ entries so no .opencode/ci/ file is ever
// written and the embedded/source mirror comparison enumerates only
// the nine .opencode/ assets (design D8).
func isCIAsset(relPath string) bool {
	return strings.HasPrefix(relPath, "ci/")
}

// renderCIWorkflowTemplate substitutes the CI workflow template's two
// placeholders — __GAZE_VERSION__ (the validated gaze release tag,
// design D4) and __GO_VERSION__ (the grammar-validated Go toolchain
// version, design D5). It is a pure function of the template bytes so
// the generated workflow is golden-file testable (design D9); both
// substituted values are validated upstream (resolveGazeVersion,
// parseGoModVersion), so no escaping is needed here.
func renderCIWorkflowTemplate(template []byte, gazeVersion, goVersion string) []byte {
	rendered := strings.ReplaceAll(string(template), gazeVersionPlaceholder, gazeVersion)
	rendered = strings.ReplaceAll(rendered, goVersionPlaceholder, goVersion)
	return []byte(rendered)
}

// writeCIWorkflow generates the AI-report caller workflow at
// .github/workflows/gaze-ai-report.yml in the target directory
// (design D7). It is a dedicated write path, separate from
// processAssetFile: the workflow is not a mirrored .opencode/ asset,
// it carries its own `#` provenance header instead of the Markdown
// version marker (design D9 — insertMarkerAfterFrontmatter does not
// apply), and it is user-owned forever — it is deliberately never
// added to isToolOwned, so an existing file is skipped unless --force
// regenerates it. ci must carry the resolved gazeVersion and
// goVersion (Run guarantees both whenever the write path is reached).
// Outcomes reuse writeNewFile's created/overwritten actions plus the
// skipped case, reported by Run through the existing Result fields.
func writeCIWorkflow(opts Options, ci ciContext) (string, error) {
	outPath := filepath.Join(opts.TargetDir, filepath.FromSlash(ciWorkflowPath))

	_, statErr := os.Stat(outPath)
	if statErr != nil && !errors.Is(statErr, fs.ErrNotExist) {
		return "", fmt.Errorf("checking %s: %w", ciWorkflowPath, statErr)
	}
	exists := statErr == nil

	// User-owned, so pure skip-if-present. Unlike tool-owned .opencode/
	// assets there is no content comparison — the skip decision is
	// independent of the template, so it is made before reading it.
	if exists && !opts.Force {
		return "skipped", nil
	}

	template, err := assets.ReadFile(ciWorkflowTemplatePath)
	if err != nil {
		return "", fmt.Errorf("reading embedded CI workflow template: %w", err)
	}
	content := renderCIWorkflowTemplate(template, ci.gazeVersion, ci.goVersion)

	return writeNewFile(outPath, content, exists, ciWorkflowPath)
}

// handleToolOwnedFile compares the existing file at outPath with the
// new content. If they differ, the file is overwritten. Returns an
// action string ("updated" or "skipped") and any error.
func handleToolOwnedFile(outPath string, content []byte, displayPath string) (string, error) {
	existing, err := os.ReadFile(outPath)
	if err != nil {
		return "", fmt.Errorf("reading existing %s: %w", displayPath, err)
	}
	if bytes.Equal(existing, content) {
		return "skipped", nil
	}
	// Content differs — overwrite. Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return "", fmt.Errorf("creating directory for %s: %w", displayPath, err)
	}
	if err := os.WriteFile(outPath, content, 0o644); err != nil {
		return "", fmt.Errorf("updating %s: %w", displayPath, err)
	}
	return "updated", nil
}

// writeNewFile creates parent directories and writes content to
// outPath. Returns an action string ("overwritten" or "created")
// based on whether the file previously existed.
func writeNewFile(outPath string, content []byte, exists bool, displayPath string) (string, error) {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return "", fmt.Errorf("creating directory %s: %w", filepath.Dir(outPath), err)
	}
	if err := os.WriteFile(outPath, content, 0o644); err != nil {
		return "", fmt.Errorf("creating %s: %w", displayPath, err)
	}
	if exists {
		return "overwritten", nil
	}
	return "created", nil
}

// processAssetFile handles a single embedded asset: checks existence,
// reads content, inserts the version marker for Markdown files (non-
// Markdown files skip the marker), and writes or skips based on
// force/tool-ownership semantics. Returns the action taken
// ("created", "overwritten", "updated", "skipped") or an error.
func processAssetFile(embeddedPath, relPath string, opts Options, marker string) (string, error) {
	outPath := filepath.Join(opts.TargetDir, ".opencode", relPath)
	displayPath := filepath.Join(".opencode", relPath)

	_, statErr := os.Stat(outPath)
	if statErr != nil && !errors.Is(statErr, fs.ErrNotExist) {
		return "", fmt.Errorf("checking %s: %w", displayPath, statErr)
	}
	exists := statErr == nil

	content, err := assets.ReadFile(embeddedPath)
	if err != nil {
		return "", fmt.Errorf("reading embedded asset %s: %w", embeddedPath, err)
	}

	// Skip the HTML version marker for non-Markdown files.
	// The marker (<!-- scaffolded by gaze vX.Y.Z -->) is an HTML
	// comment that is invalid in formats like JSONC.
	var out []byte
	if strings.HasSuffix(relPath, ".md") {
		out = insertMarkerAfterFrontmatter(content, marker)
	} else {
		out = content
	}

	if exists && !opts.Force {
		if isToolOwned(relPath) {
			return handleToolOwnedFile(outPath, out, displayPath)
		}
		return "skipped", nil
	}

	return writeNewFile(outPath, out, exists, displayPath)
}

// appendActionResult records one asset-walk outcome on the scaffold
// result under the file's .opencode display path. The action strings
// mirror processAssetFile's returns.
func appendActionResult(result *Result, relPath, action string) {
	displayPath := filepath.Join(".opencode", relPath)
	switch action {
	case "created":
		result.Created = append(result.Created, displayPath)
	case "overwritten":
		result.Overwritten = append(result.Overwritten, displayPath)
	case "updated":
		result.Updated = append(result.Updated, displayPath)
	case "skipped":
		result.Skipped = append(result.Skipped, displayPath)
	}
}

// appendCIWorkflowResult generates the AI-report caller workflow
// (design D7) when CI scaffolding was requested and not skipped for a
// missing go.mod (design D6), recording the outcome on the scaffold
// result. The write runs after the .opencode/ walk so the core
// scaffolding completes even when the CI write fails. The workflow is
// user-owned and reported through the standard result fields, so
// printSummary needs no new plumbing.
func appendCIWorkflowResult(result *Result, opts Options, ci ciContext) error {
	if !opts.CI || ci.skipped {
		return nil
	}
	action, err := writeCIWorkflow(opts, ci)
	if err != nil {
		return err
	}
	switch action {
	case "created":
		result.Created = append(result.Created, ciWorkflowPath)
	case "overwritten":
		result.Overwritten = append(result.Overwritten, ciWorkflowPath)
	case "skipped":
		result.Skipped = append(result.Skipped, ciWorkflowPath)
	}
	return nil
}

// Run scaffolds OpenCode agent, command, reference, and configuration
// files into the target directory. It creates .opencode/agents/,
// .opencode/commands/, and .opencode/references/ subdirectories
// and writes the embedded quality-reporting files and DCP config.
//
// Markdown files are prepended with a version marker comment:
//
//	<!-- scaffolded by gaze vX.Y.Z -->
//
// Non-Markdown files (e.g., dcp.jsonc) skip the marker because the
// HTML comment syntax is invalid in those formats.
//
// Files are classified as user-owned or tool-owned via
// isToolOwned(). If a user-owned file already exists
// and opts.Force is false, the file is skipped. Tool-owned files
// use overwrite-on-diff: they are replaced when their content
// differs from the embedded version, even without --force. If
// opts.Force is true, all files are overwritten regardless of
// ownership.
//
// If opts.CI is set, Run resolves the gaze version — the
// Options.GazeVersion override when set, else the build-time
// Options.Version — and validates it against minWorkflowTag before
// touching the filesystem: an invalid, pre-floor, dev, or empty
// version fails closed with an error and nothing is written. A
// plain run (opts.CI false) skips version validation entirely.
//
// The CI workflow's Go toolchain version is resolved from the
// target's go.mod `go` directive via a minimal, validated line scan
// (design D5): a directive that is absent, unreadable, or outside
// the accepted grammar falls back to defaultGoVersion with a
// warning. When the target has no go.mod at the repository root, CI
// scaffolding is skipped entirely with a distinct warning naming the
// requirement (design D6) — non-fatal; the .opencode/ scaffolding
// proceeds.
//
// When CI scaffolding proceeds, Run generates the AI-report caller
// workflow at .github/workflows/gaze-ai-report.yml after the
// .opencode/ asset walk (design D7). The workflow is user-owned —
// skipped when it already exists, regenerated only with --force — and
// never joins the .opencode/ mirror or isToolOwned. The summary is
// then followed by a CI next-steps section (the secret-setup
// one-liner, the jq minify tip, the second-secret masking pointer, and
// a paste-able threshold-guidance prompt) whenever CI scaffolding ran
// and was not skipped for a missing go.mod (design D10).
//
// After the asset walk, Run registers the divisor-gaze-testing
// persona in an existing .uf review configuration (a project
// without a .uf directory gets a notice; .uf files are never
// created) and removes gaze-marked .opencode/agents/reviewer-testing.md
// orphans superseded by the renamed agent file.
//
// Run returns a Result summarizing what was created, skipped,
// overwritten, updated, registered, or removed.
func Run(opts Options) (*Result, error) {
	if err := applyDefaults(&opts); err != nil {
		return nil, err
	}

	// Resolve and validate the CI version pin (design D4) before any
	// filesystem access, so a failed --ci invocation writes nothing
	// and the combined dev-build-without-go.mod case fails closed on
	// the version error rather than the missing-go.mod skip (design
	// D6 check ordering). Plain runs (CI=false) skip validation
	// entirely.
	gazeVersion, err := resolveCIPin(opts)
	if err != nil {
		return nil, err
	}

	// Check for go.mod and warn if absent.
	goModPath, goModAbsent := checkGoMod(opts)

	// Resolve the remaining CI-scaffolding inputs (designs D5 and
	// D6), gated on opts.CI so plain init output is unchanged.
	ci := completeCIContext(opts, goModPath, goModAbsent, gazeVersion)

	result := &Result{}
	marker := versionMarker(opts.Version)

	err = fs.WalkDir(assets, "assets", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		relPath := strings.TrimPrefix(p, "assets/")
		// The ci/ subtree holds the CI workflow template, not a
		// mirrored .opencode/ asset (design D8); it is consumed only
		// by the dedicated CI write path below.
		if isCIAsset(relPath) {
			return nil
		}
		action, err := processAssetFile(p, relPath, opts, marker)
		if err != nil {
			return err
		}
		appendActionResult(result, relPath, action)
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Generate the AI-report caller workflow (design D7) after the
	// .opencode/ walk so the core scaffolding completes even when the
	// CI write fails.
	if err := appendCIWorkflowResult(result, opts, ci); err != nil {
		return nil, err
	}

	// Register the divisor-gaze-testing persona in an existing .uf
	// review configuration (projects already using unbound-force)
	// and clean up gaze-marked reviewer-testing.md orphans
	// superseded by the renamed agent file.
	result.Registered = append(result.Registered, syncUFReviewConfig(opts)...)
	result.Removed = append(result.Removed, cleanupOrphanedTestingReviewer(opts)...)

	printSummary(opts.Stdout, result)

	// Print the CI next-steps guidance (design D10) after the summary
	// when CI scaffolding ran: the section walks the adopter through
	// secret setup, log-masking hygiene, and threshold selection. It is
	// suppressed on the missing-go.mod skip (design D6), because
	// guidance for a workflow that was never written is not actionable.
	if opts.CI && !ci.skipped {
		printCINextSteps(opts.Stdout)
	}
	return result, nil
}

// printSummary writes a human-readable summary of the scaffold
// operation to w.
func printSummary(w io.Writer, r *Result) {
	if len(r.Created) > 0 || len(r.Overwritten) > 0 || len(r.Updated) > 0 ||
		len(r.Registered) > 0 || len(r.Removed) > 0 {
		_, _ = fmt.Fprintln(w, "Gaze OpenCode integration initialized:")
	} else {
		_, _ = fmt.Fprintln(w, "Gaze OpenCode integration already up to date:")
	}

	for _, f := range r.Created {
		_, _ = fmt.Fprintf(w, "  created: %s\n", f)
	}
	for _, f := range r.Skipped {
		_, _ = fmt.Fprintf(w, "  skipped: %s (already exists)\n", f)
	}
	for _, f := range r.Overwritten {
		_, _ = fmt.Fprintf(w, "  overwritten: %s\n", f)
	}
	for _, f := range r.Updated {
		_, _ = fmt.Fprintf(w, "  updated: %s (content changed)\n", f)
	}
	for _, f := range r.Registered {
		_, _ = fmt.Fprintf(w, "  registered: %s (divisor-gaze-testing)\n", f)
	}
	for _, f := range r.Removed {
		_, _ = fmt.Fprintf(w, "  removed: %s (superseded by agents/divisor-gaze-testing.md)\n", f)
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Run /gaze for quality reports and /speckit.testreview for testability analysis.")

	// Count only user-owned skipped files for the --force hint.
	// Tool-owned reference files that are skipped (identical content)
	// do not need --force — they auto-update when content changes.
	var userSkipped int
	for _, f := range r.Skipped {
		rel := strings.TrimPrefix(f, filepath.Join(".opencode")+string(filepath.Separator))
		if !isToolOwned(rel) {
			userSkipped++
		}
	}
	if userSkipped > 0 {
		word := "file"
		if userSkipped > 1 {
			word = "files"
		}
		_, _ = fmt.Fprintf(w, "%d %s skipped (use --force to overwrite).\n", userSkipped, word)
	}
}

// printCINextSteps prints the CI adoption guidance (design D10) that
// follows the scaffold summary when CI scaffolding ran: the
// OPENCODE_CONFIG_CONTENT secret-setup one-liner with its jq minify
// tip, the second-secret masking pointer, and a paste-able
// threshold-guidance prompt. Run calls it only when CI scaffolding was
// requested and not skipped for a missing go.mod: guidance for a
// workflow that was never written would not be actionable. The section
// prints identically whether the workflow was created, skipped as
// existing, or overwritten with --force — the next steps are the same.
func printCINextSteps(w io.Writer) {
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "CI next steps for %s:\n", ciWorkflowPath)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "1. Store your opencode configuration as the OPENCODE_CONFIG_CONTENT secret:")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "     gh secret set OPENCODE_CONFIG_CONTENT < opencode-config.json")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "   The config is single-line strict JSON holding the provider, model,")
	_, _ = fmt.Fprintln(w, "   permission denials, and the literal API key; see")
	_, _ = fmt.Fprintln(w, "   docs/guides/ci-integration.md for the full shape. Minify it first")
	_, _ = fmt.Fprintln(w, "   with `jq -c .`: GitHub's multiline secret masking is unreliable, and")
	_, _ = fmt.Fprintln(w, "   the single-line form costs nothing.")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "2. Register the bare API key as a second secret (e.g. OPENCODE_API_KEY)")
	_, _ = fmt.Fprintln(w, "   and reference it in an unused env var on a calling-job step, so")
	_, _ = fmt.Fprintln(w, "   GitHub masks key fragments echoed into logs. See \"Secret hygiene and")
	_, _ = fmt.Fprintln(w, "   stderr redaction\" in docs/guides/ci-integration.md for the pattern.")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "3. Choose thresholds: the scaffolded workflow is report-only until you")
	_, _ = fmt.Fprintln(w, "   uncomment its threshold inputs. Paste this prompt to your AI agent:")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "     Read %s, run\n", ciWorkflowPath)
	_, _ = fmt.Fprintln(w, "     `gaze report ./... --format=json`, and propose permissive initial")
	_, _ = fmt.Fprintln(w, "     max-crapload, max-gaze-crapload, and min-contract-coverage values")
	_, _ = fmt.Fprintln(w, "     from the summary metrics. Wait for my approval of the final values,")
	_, _ = fmt.Fprintln(w, "     then uncomment the threshold inputs in the workflow.")
	_, _ = fmt.Fprintln(w)
}

// assetPaths returns the relative paths of the embedded assets that
// mirror into .opencode/. Entries in the ci/ subtree are excluded
// (design D8): that subtree holds the CI workflow template consumed
// by the dedicated CI write path, not a mirrored asset. This is used
// by the drift detection test to enumerate expected files.
func assetPaths() ([]string, error) {
	var paths []string
	err := fs.WalkDir(assets, "assets", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		relPath := strings.TrimPrefix(path, "assets/")
		if isCIAsset(relPath) {
			return nil
		}
		paths = append(paths, relPath)
		return nil
	})
	return paths, err
}

// assetContent returns the raw content of an embedded asset by
// its relative path (e.g., "agents/gaze-reporter.md").
func assetContent(relPath string) ([]byte, error) {
	return assets.ReadFile(path.Join("assets", relPath))
}
