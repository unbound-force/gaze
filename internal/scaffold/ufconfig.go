// ufconfig.go registers the divisor-gaze-testing persona in existing
// unbound-force .uf review configuration and cleans up superseded
// gaze-scaffolded orphans. All operations are best-effort
// intelligence layered on top of the asset walk: malformed input
// warns and skips; gaze init never fails here and never creates .uf
// files (design decisions D5/D6 of the divisor-gaze-testing change).

package scaffold

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// ufGazeTestingAgent is the gaze-aware test-quality reviewer
	// occupying a new unique divisor slot.
	ufGazeTestingAgent = "divisor-gaze-testing"

	// ufGenericTestingAgent is the uf-owned generic test-quality
	// reviewer that the advisor lists swap away from. Its
	// capabilities entry is deliberately preserved (coexistence);
	// it is swapped out of all four advisor lists, triage included
	// (openspec/changes/gaze-init-triage-advisor).
	ufGenericTestingAgent = "divisor-testing"

	// ufCapabilitiesRelPath is the repo-relative display path of the
	// reviewer capabilities manifest inside a target project.
	ufCapabilitiesRelPath = ".uf/reviewer-capabilities.yaml"

	// ufMatrixRelPath is the repo-relative display path of the
	// review dispatch matrix inside a target project.
	ufMatrixRelPath = ".uf/review-matrix.yaml"

	// legacyTestingReviewerRelPath is the pre-rename scaffold path of
	// the testing persona (spec 017), superseded by
	// agents/divisor-gaze-testing.md.
	legacyTestingReviewerRelPath = ".opencode/agents/reviewer-testing.md"
)

// syncUFReviewConfig registers the divisor-gaze-testing persona in a
// target project's existing .uf review configuration and returns the
// display paths of the config files it rewrote. A project without a
// .uf directory gets a notice on opts.Stdout and nothing is created:
// a minimal capabilities file missing uf's canonical nine reviewers
// would fail uf's own manifest enforcement, so manufacturing one is
// worse than skipping (Minimal Assumptions). The two config files
// are handled independently — a malformed or missing file warns and
// is skipped without failing init (graceful degradation, mirroring
// the go.mod warning pattern).
func syncUFReviewConfig(opts Options) []string {
	ufDir := filepath.Join(opts.TargetDir, ".uf")
	info, err := os.Stat(ufDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			_, _ = fmt.Fprintln(opts.Stdout, "Notice: no .uf directory found; skipping Divisor review-config registration.")
			_, _ = fmt.Fprintln(opts.Stdout, "Gaze never creates .uf files; unbound-force projects register the persona automatically.")
		} else {
			_, _ = fmt.Fprintf(opts.Stdout, "Warning: checking %s: %v\n", ufDir, err)
		}
		return nil
	}
	if !info.IsDir() {
		_, _ = fmt.Fprintf(opts.Stdout, "Warning: %s is not a directory; skipping Divisor review-config registration.\n", ufDir)
		return nil
	}

	var registered []string
	if syncReviewerCapabilities(filepath.Join(ufDir, "reviewer-capabilities.yaml"), opts.Stdout) {
		registered = append(registered, ufCapabilitiesRelPath)
	}
	if syncReviewMatrix(filepath.Join(ufDir, "review-matrix.yaml"), opts.Stdout) {
		registered = append(registered, ufMatrixRelPath)
	}
	return registered
}

// syncReviewerCapabilities appends the divisor-gaze-testing reviewer
// entry to the reviewers sequence of a reviewer-capabilities.yaml,
// preserving comments and unknown fields via the yaml.Node
// round-trip. It reports whether the file was rewritten; an
// already-registered file is an idempotent no-op that is never
// rewritten.
func syncReviewerCapabilities(path string, stdout io.Writer) bool {
	data, doc, ok := loadYAMLDocument(path, stdout)
	if !ok {
		return false
	}
	root := documentRoot(&doc)
	if root == nil {
		warnSkip(stdout, path, "no document root")
		return false
	}
	reviewers := mappingValue(root, "reviewers")
	if reviewers == nil || reviewers.Kind != yaml.SequenceNode {
		warnSkip(stdout, path, "missing or invalid 'reviewers' sequence")
		return false
	}
	if reviewerEntryExists(reviewers, ufGazeTestingAgent) {
		return false
	}
	reviewers.Content = append(reviewers.Content, gazeTestingReviewerEntry())
	return writeYAMLIfChanged(path, data, &doc, stdout)
}

// syncReviewMatrix swaps the generic divisor-testing advisor for
// divisor-gaze-testing in the code, spec, triage, and feedback advisor
// lists of a review-matrix.yaml. A missing or non-sequence list in any
// of the four modes skips the whole file with a warning (no partial
// registration). Always overrides and every other key pass through the
// Node round-trip untouched. It reports whether the file was
// rewritten; a file already in the end-state is an idempotent no-op
// that is never rewritten.
func syncReviewMatrix(path string, stdout io.Writer) bool {
	data, doc, ok := loadYAMLDocument(path, stdout)
	if !ok {
		return false
	}
	root := documentRoot(&doc)
	if root == nil {
		warnSkip(stdout, path, "no document root")
		return false
	}
	advisor := mappingValue(root, "advisor")
	if advisor == nil || advisor.Kind != yaml.MappingNode {
		warnSkip(stdout, path, "missing or invalid 'advisor' mapping")
		return false
	}
	changed := false
	for _, mode := range []string{"code", "spec", "triage", "feedback"} {
		list := mappingValue(advisor, mode)
		if list == nil || list.Kind != yaml.SequenceNode {
			// Missing expected key: skip the whole file rather than
			// applying a partial registration.
			warnSkip(stdout, path, fmt.Sprintf("missing or invalid 'advisor.%s' sequence", mode))
			return false
		}
		if seqRemoveString(list, ufGenericTestingAgent) {
			changed = true
		}
		if !seqContainsString(list, ufGazeTestingAgent) {
			list.Content = append(list.Content, scalarString(ufGazeTestingAgent))
			changed = true
		}
	}
	if !changed {
		return false
	}
	return writeYAMLIfChanged(path, data, &doc, stdout)
}

// cleanupOrphanedTestingReviewer removes a legacy
// .opencode/agents/reviewer-testing.md (superseded by
// agents/divisor-gaze-testing.md) when its content carries the gaze
// version marker, returning the display path of the removed file.
// The marker is the provenance signal: marker-less copies are
// user-authored or customized and stay untouched. An absent file is
// a silent no-op. This cleanup runs regardless of .uf presence —
// any gaze-scaffolded project can carry the orphan.
func cleanupOrphanedTestingReviewer(opts Options) []string {
	legacyPath := filepath.Join(opts.TargetDir, ".opencode", "agents", "reviewer-testing.md")
	content, err := os.ReadFile(legacyPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			_, _ = fmt.Fprintf(opts.Stdout, "Warning: reading %s: %v\n", legacyPath, err)
		}
		return nil
	}
	if !contentHasMarker(content) {
		return nil
	}
	if err := os.Remove(legacyPath); err != nil {
		_, _ = fmt.Fprintf(opts.Stdout, "Warning: removing %s: %v\n", legacyPath, err)
		return nil
	}
	return []string{legacyTestingReviewerRelPath}
}

// contentHasMarker reports whether any line of content is a gaze
// version-marker comment.
func contentHasMarker(content []byte) bool {
	for _, line := range strings.Split(string(content), "\n") {
		if isMarkerLine(line) {
			return true
		}
	}
	return false
}

// loadYAMLDocument reads and parses a YAML file into a yaml.Node
// document tree. It reports ok=false, after warning on stdout, when
// the file is absent, unreadable, or malformed; callers skip such
// files without failing init.
func loadYAMLDocument(path string, stdout io.Writer) (data []byte, doc yaml.Node, ok bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			_, _ = fmt.Fprintf(stdout, "Warning: %s not found; skipping.\n", path)
		} else {
			_, _ = fmt.Fprintf(stdout, "Warning: reading %s: %v\n", path, err)
		}
		return nil, yaml.Node{}, false
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		_, _ = fmt.Fprintf(stdout, "Warning: parsing %s: %v; skipping.\n", path, err)
		return nil, yaml.Node{}, false
	}
	return data, doc, true
}

// documentRoot returns the root content node of a parsed YAML
// document (the mapping or sequence node beneath the document node),
// or nil when the document is empty.
func documentRoot(doc *yaml.Node) *yaml.Node {
	if doc == nil || doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil
	}
	return doc.Content[0]
}

// mappingValue returns the value node for key in a mapping node, or
// nil when node is not a mapping or the key is absent.
func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Kind == yaml.ScalarNode && node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

// reviewerEntryExists reports whether a reviewers sequence already
// carries an entry with the given agent name.
func reviewerEntryExists(reviewers *yaml.Node, agent string) bool {
	for _, entry := range reviewers.Content {
		if name := mappingValue(entry, "agent"); name != nil && name.Value == agent {
			return true
		}
	}
	return false
}

// gazeTestingReviewerEntry builds the reviewer-capabilities entry
// for the gaze test-quality persona. The block-style rendering
// matches the surrounding file convention (two-space indent, one
// scopes item per line).
func gazeTestingReviewerEntry() *yaml.Node {
	scopes := &yaml.Node{
		Kind:    yaml.SequenceNode,
		Tag:     "!!seq",
		Content: []*yaml.Node{scalarString("test-quality")},
	}
	return &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
		Content: []*yaml.Node{
			scalarString("agent"), scalarString(ufGazeTestingAgent),
			scalarString("capability"), scalarString("review"),
			scalarString("scopes"), scopes,
		},
	}
}

// seqContainsString reports whether a sequence node contains a
// scalar item with the given value.
func seqContainsString(seq *yaml.Node, value string) bool {
	if seq == nil || seq.Kind != yaml.SequenceNode {
		return false
	}
	for _, item := range seq.Content {
		if item.Kind == yaml.ScalarNode && item.Value == value {
			return true
		}
	}
	return false
}

// seqRemoveString removes scalar items with the given value from a
// sequence node, reporting whether anything was removed.
func seqRemoveString(seq *yaml.Node, value string) bool {
	removed := false
	kept := seq.Content[:0]
	for _, item := range seq.Content {
		if item.Kind == yaml.ScalarNode && item.Value == value {
			removed = true
			continue
		}
		kept = append(kept, item)
	}
	seq.Content = kept
	return removed
}

// scalarString builds a plain string scalar node.
func scalarString(v string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v}
}

// writeYAMLIfChanged re-encodes a mutated document with two-space
// indentation (matching the existing .uf file style) and rewrites
// path only when the encoding differs from the bytes already on
// disk. It reports whether the file was rewritten.
func writeYAMLIfChanged(path string, original []byte, doc *yaml.Node, stdout io.Writer) bool {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		_, _ = fmt.Fprintf(stdout, "Warning: encoding %s: %v; skipping.\n", path, err)
		return false
	}
	if err := enc.Close(); err != nil {
		_, _ = fmt.Fprintf(stdout, "Warning: encoding %s: %v; skipping.\n", path, err)
		return false
	}
	if bytes.Equal(buf.Bytes(), original) {
		return false
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		_, _ = fmt.Fprintf(stdout, "Warning: writing %s: %v; skipping.\n", path, err)
		return false
	}
	return true
}

// warnSkip prints a skip warning for a .uf config file that could
// not be merged.
func warnSkip(stdout io.Writer, path, reason string) {
	_, _ = fmt.Fprintf(stdout, "Warning: %s: %s; skipping.\n", path, reason)
}
