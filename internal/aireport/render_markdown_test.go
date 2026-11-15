package aireport

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"unicode"

	"github.com/unbound-force/gaze/v2/internal/crap"
)

// renderFixture loads a payload fixture, optionally restricts its
// sections, and renders it.
func renderFixture(t *testing.T, name string, outcome AIOutcome, sections ...string) string {
	t.Helper()
	p := loadPayloadFixture(t, name)
	if len(sections) > 0 {
		set, err := ParseSections(sections)
		if err != nil {
			t.Fatalf("ParseSections(%v): %v", sections, err)
		}
		p.Metadata.Sections = set.List()
	}
	out, err := RenderMarkdown(p, outcome)
	if err != nil {
		t.Fatalf("RenderMarkdown: %v", err)
	}
	return out
}

// validAnswer is a valid AIAnswer used by slot tests.
func validAnswer() *AIAnswer {
	return &AIAnswer{
		Summary:         "The project is in fair shape.",
		Recommendations: []string{"Decompose `runQuality`.", "Add tests for `parseConfig`."},
	}
}

func TestRenderMarkdown_NilPayload(t *testing.T) {
	if _, err := RenderMarkdown(nil, AIOutcome{}); err == nil {
		t.Fatal("RenderMarkdown(nil) = nil error, want error")
	}
}

func TestRenderMarkdown_Titles(t *testing.T) {
	tests := []struct {
		name     string
		sections []string
		want     string
	}{
		{"all sections", []string{"crap", "quality", "classification", "health"}, "# 🔍 Gaze Full Quality Report"},
		{"crap only", []string{"crap"}, "# 🔍 Gaze CRAP Report"},
		{"quality only", []string{"quality"}, "# 🔍 Gaze Quality Report"},
		{"crap and quality", []string{"quality", "crap"}, "# 🔍 Gaze Quality Report (sections: crap, quality)"},
		{"classification and health", []string{"health", "classification"}, "# 🔍 Gaze Quality Report (sections: classification, health)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := renderFixture(t, "payload-full.json", AIOutcome{}, tt.sections...)
			if !strings.HasPrefix(out, tt.want+"\n") {
				t.Errorf("first line = %q, want %q", firstLine(out), tt.want)
			}
		})
	}
}

// firstLine returns the first line of s.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func TestRenderMarkdown_MetadataBranchOmitted(t *testing.T) {
	p := loadPayloadFixture(t, "payload-full.json")
	p.Metadata.Branch = ""
	out, err := RenderMarkdown(p, AIOutcome{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "Branch:") || strings.Contains(out, " · \n") {
		t.Errorf("empty Branch must be omitted with its separator; got header %q", headerOf(out))
	}
	if !strings.Contains(out, "Project: github.com/example/project\n") {
		t.Errorf("project line malformed; got header %q", headerOf(out))
	}
}

// headerOf returns the metadata header (the first four lines).
func headerOf(s string) string {
	lines := strings.SplitN(s, "\n", 5)
	if len(lines) < 3 {
		return s
	}
	return strings.Join(lines[:3], "\n")
}

func TestRenderMarkdown_MetadataEscaped(t *testing.T) {
	p := loadPayloadFixture(t, "payload-full.json")
	p.Metadata.Branch = "ev[il](https://x) <!--"
	p.Metadata.ModulePath = "github.com/example/<script>"
	out, err := RenderMarkdown(p, AIOutcome{})
	if err != nil {
		t.Fatal(err)
	}
	header := headerOf(out)
	want := `Project: github.com/example/\<script\> · Branch: ev\[il\](https\[:\]//x) \<!--`
	if !strings.Contains(header, want) {
		t.Errorf("header %q does not contain escaped values %q", header, want)
	}
	for _, raw := range []string{"[", "]", "<", ">"} {
		for _, idx := range allIndexes(header, raw) {
			if idx == 0 || header[idx-1] != '\\' {
				t.Errorf("header %q contains unescaped %q at byte %d", header, raw, idx)
			}
		}
	}
}

// allIndexes returns every byte offset of sub in s.
func allIndexes(s, sub string) []int {
	var out []int
	for i := 0; i < len(s); {
		j := strings.Index(s[i:], sub)
		if j < 0 {
			break
		}
		out = append(out, i+j)
		i += j + 1
	}
	return out
}

func TestRenderMarkdown_UnselectedSectionsOmitted(t *testing.T) {
	out := renderFixture(t, "payload-full.json", AIOutcome{}, "crap")
	for _, heading := range []string{"## 🧪 Quality Summary", "## 🏷️ Classification Summary", "## 🏥 Overall Health Assessment"} {
		if strings.Contains(out, heading) {
			t.Errorf("crap-only report contains %q", heading)
		}
	}
	if !strings.Contains(out, "## 📊 CRAP Summary") {
		t.Error("crap-only report missing the CRAP section")
	}
}

func TestRenderMarkdown_SectionSeparators(t *testing.T) {
	out := renderFixture(t, "payload-full.json", AIOutcome{})
	if got, want := strings.Count(out, "\n---\n"), 4; got != want {
		t.Errorf("separator count = %d, want %d", got, want)
	}
}

func TestRenderMarkdown_TrailingNewline(t *testing.T) {
	for _, sections := range [][]string{
		{"crap", "quality", "classification", "health"},
		{"crap"},
	} {
		out := renderFixture(t, "payload-full.json", AIOutcome{Answer: validAnswer()}, sections...)
		if !strings.HasSuffix(out, "\n") || strings.HasSuffix(out, "\n\n") {
			t.Errorf("sections %v: report must end with exactly one newline", sections)
		}
	}
}

func TestRenderMarkdown_CellEscaping(t *testing.T) {
	p := loadPayloadFixture(t, "payload-full.json")
	out, err := RenderMarkdown(p, AIOutcome{})
	if err != nil {
		t.Fatal(err)
	}

	longName := strings.Repeat("processGeneratedProtocolBufferMessageForTheExtremelyLongServiceName", 3)[:200]
	if !strings.Contains(out, codeSpan(longName)) {
		t.Error("200-rune function name was truncated")
	}
	if !strings.Contains(out, "``parse`raw`Literal``") {
		t.Error("backtick name is not in a double-backtick span")
	}
	if !strings.Contains(out, `parse\|Pipe`) {
		t.Error("pipe inside a function name was not escaped")
	}
	if !strings.Contains(out, "(*队列处理器).处理消息") {
		t.Error("CJK function name missing")
	}

	var report struct {
		Summary struct {
			Worst []struct {
				Function string `json:"function"`
			} `json:"worst_crap"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(p.CRAP, &report); err != nil {
		t.Fatal(err)
	}
	for _, row := range report.Summary.Worst {
		if strings.Contains(row.Function, "\n") {
			t.Errorf("fixture function %q unexpectedly contains a newline", row.Function)
		}
	}
}

func TestRenderMarkdown_NewlineInCell(t *testing.T) {
	if got, want := escapeCell("a\nb\rc\r\nd"), "a b c d"; got != want {
		t.Errorf("escapeCell = %q, want %q", got, want)
	}
	if got, want := escapeCell("a|b"), `a\|b`; got != want {
		t.Errorf("escapeCell = %q, want %q", got, want)
	}
}

// TestEscapeCell_Markup pins the bracket, angle-bracket, and ampersand
// escaping that keeps analysis-derived names from becoming links, HTML, or
// entity-decoded autolink triggers inside a table cell. The ampersand cases
// pin the plain-cell entity escape: a backslash-escaped "&" cannot open an
// HTML character reference, so cmark-gfm cannot decode entity-encoded
// autolink triggers after byte-level defanging. Without these cases the
// escaping branches can be deleted silently.
func TestEscapeCell_Markup(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"link", "a[b](https://evil.example)", `a\[b\](https\[:\]//evil.example)`},
		{"html tag", "<script>alert(1)</script>", `\<script\>alert(1)\</script\>`},
		{"html comment", "<!-- hidden -->", `\<!-- hidden --\>`},
		{"pipe", "a|b", `a\|b`},
		{"line breaks", "a\r\nb\nc\rd", "a b c d"},
		{"all together", "x|[y](z)<w>", `x\|\[y\](z)\<w\>`},
		{"benign ampersand", "R&D", `R\&D`},
		// Hostile: "&#64;" carries no literal "@" for neutralizeAutolinks
		// to defang, and cmark-gfm would decode the reference into the
		// email autolink trigger; the backslash-escaped "&" keeps the
		// reference from ever opening.
		{"ampersand entity", "user&#64;host.example", `user\&#64;host.example`},
		{"www entity prefix", "&#119;ww.evil.example", `\&#119;ww.evil.example`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := escapeCell(tc.in); got != tc.want {
				t.Errorf("escapeCell(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestSanitizeProse_Markup pins prose escaping for callouts and
// recommendation text, which are not table cells. The ampersand cases
// pin the entity-escape pair: a backslash-escaped "&" cannot open an
// HTML character reference, so cmark-gfm cannot decode entity-encoded
// autolink triggers after byte-level defanging.
func TestSanitizeProse_Markup(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"link", "[docs](https://evil.example)", `\[docs\](https\[:\]//evil.example)`},
		// ">" is intentionally left alone: it cannot open a tag, and
		// escaping it would corrupt ordinary prose such as "(1201 > 1200)".
		{"html tag", "<img src=x>", `\<img src=x>`},
		{"benign ampersand", "R&D", `R\&D`},
		// Hostile: "&#64;" carries no literal "@" for neutralizeAutolinks
		// to defang, and cmark-gfm would decode the reference into the
		// email autolink trigger; the backslash-escaped "&" keeps the
		// reference from ever opening.
		{"ampersand entity", "user&#64;host.example", `user\&#64;host.example`},
		{"line breaks", "one\ntwo\r\nthree\rfour", "one two three four"},
		{"plain prose unchanged", "Decompose (a > b) at 50% coverage.", "Decompose (a > b) at 50% coverage."},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sanitizeProse(tc.in); got != tc.want {
				t.Errorf("sanitizeProse(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestSanitizeHeaderValue_Pairs pins every headerEscaper pair exactly,
// including the \r\n-before-\r/\n replacer ordering: reordered pairs would
// flatten "a\r\nb" to two spaces instead of one. The ampersand cases pin
// the entity-escape pair the same way: a backslash-escaped "&" cannot
// open an HTML character reference, so cmark-gfm cannot decode
// entity-encoded autolink triggers after byte-level defanging.
func TestSanitizeHeaderValue_Pairs(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"backslash", `a\b`, `a\\b`},
		{"backtick", "a`b", "a\\`b"},
		{"asterisk", "*a*", `\*a\*`},
		{"underscore", "_a_", `\_a\_`},
		{"open bracket", "a[b", `a\[b`},
		{"close bracket", "a]b", `a\]b`},
		{"open angle", "a<b", `a\<b`},
		{"close angle", "a>b", `a\>b`},
		{"pipe", "a|b", `a\|b`},
		{"ampersand", "a&b", `a\&b`},
		{"benign ampersand", "R&D", `R\&D`},
		// One replacer pass: the backslash inserted before "&" is not
		// itself re-escaped, and the entity text survives verbatim, so
		// the new pair leaves every existing pair's behavior intact.
		{"ampersand entity", "a&amp;b", `a\&amp;b`},
		{"newline flattens", "a\nb", "a b"},
		{"carriage return flattens", "a\rb", "a b"},
		{"crlf flattens to one space", "a\r\nb", "a b"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sanitizeHeaderValue(tc.in); got != tc.want {
				t.Errorf("sanitizeHeaderValue(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// entityFixture is one cell of the sink × trigger × encoding-family
// matrix: a hostile value embedding one encoding of one autolink
// trigger character, plus the exact bytes every non-code-span sink
// must emit for it. The inputs deliberately avoid every other byte
// the sinks escape (backslash, backtick, asterisk, underscore,
// brackets, angle brackets, pipes, line breaks) and avoid literal
// autolink triggers, so one want string pins all three sinks at
// once — the only transform these inputs undergo is the universal
// "&" escape, and the single interplay case's defanged brackets
// escape identically ("\[", "\]") in all three escapers.
type entityFixture struct {
	name string
	in   string
	want string
}

// entityFixtures enumerates the complete matrix cell set from the
// delta spec (Requirement 2, epic #337 rule 2): every trigger
// character (`w`, `.`, `@`, `:`, `/`) crossed with every encoding
// family — decimal numeric; hexadecimal numeric with lowercase `x`,
// uppercase `X`, and mixed-case digits; named references (the `w`
// cell is structurally impossible: no HTML5 named reference encodes
// `w`); and hybrid literal-plus-entity strings. Each input embeds
// the encoded trigger in the autolink context it completes when
// cmark-gfm decodes it.
var entityFixtures = []entityFixture{
	// Decimal numeric: all five triggers.
	{"decimal w", "&#119;ww.evil.example", `\&#119;ww.evil.example`},
	{"decimal dot", "www&#46;evil.example", `www\&#46;evil.example`},
	{"decimal at", "user&#64;host.example", `user\&#64;host.example`},
	{"decimal colon", "https&#58;//evil.example", `https\&#58;//evil.example`},
	{"decimal slash", "https:&#47;/evil.example", `https:\&#47;/evil.example`},
	// Hexadecimal numeric: all five triggers, with lowercase `x`,
	// uppercase `X`, and mixed-case digit forms.
	{"hex w lowercase x", "&#x77;ww.evil.example", `\&#x77;ww.evil.example`},
	{"hex w uppercase X", "&#X77;ww.evil.example", `\&#X77;ww.evil.example`},
	{"hex dot lowercase x uppercase digit", "www&#x2E;evil.example", `www\&#x2E;evil.example`},
	{"hex dot uppercase x lowercase digit", "www&#X2e;evil.example", `www\&#X2e;evil.example`},
	{"hex at lowercase x", "user&#x40;host.example", `user\&#x40;host.example`},
	{"hex at uppercase X", "user&#X40;host.example", `user\&#X40;host.example`},
	{"hex colon lowercase x uppercase digit", "https&#x3A;//evil.example", `https\&#x3A;//evil.example`},
	{"hex colon uppercase X", "https&#X3A;//evil.example", `https\&#X3A;//evil.example`},
	{"hex colon lowercase x lowercase digit", "https&#x3a;//evil.example", `https\&#x3a;//evil.example`},
	{"hex slash lowercase x uppercase digit", "https:&#x2F;/evil.example", `https:\&#x2F;/evil.example`},
	{"hex slash uppercase x lowercase digit", "https:&#X2f;/evil.example", `https:\&#X2f;/evil.example`},
	// Named references: `.`, `@`, `:`, `/` — `w` is structurally
	// impossible in this family (no HTML5 named reference encodes it).
	{"named dot", "www&period;evil.example", `www\&period;evil.example`},
	{"named at", "user&commat;host.example", `user\&commat;host.example`},
	{"named colon", "https&colon;//evil.example", `https\&colon;//evil.example`},
	{"named slash", "https:&sol;/evil.example", `https:\&sol;/evil.example`},
	// Hybrid literal-plus-entity strings.
	{"hybrid decimal w plus named dots", "&#119;ww&period;evil&period;example", `\&#119;ww\&period;evil\&period;example`},
	{"hybrid named email", "user&commat;host&period;example", `user\&commat;host\&period;example`},
	{"hybrid named scheme", "https&colon;&sol;&sol;evil.example", `https\&colon;\&sol;\&sol;evil.example`},
	{"hybrid decimal dots", "www&#46;evil&#46;example", `www\&#46;evil\&#46;example`},
	// Interplay: the literal "www." trigger IS defanged by
	// neutralizeAutolinks before the "&" escape runs, so the output
	// shows both layers at once. The defanged brackets escape
	// identically ("\[", "\]") in all three sinks.
	{"hybrid literal www plus entity dot", "www.evil&#46;example", `www\[.\]evil\&#46;example`},
}

// TestNeutralizeEntityEncodedAutolinks is the spec-mandated entity
// fixture matrix (delta spec Requirement 2, task 2.1): every
// non-code-span sink crossed with every trigger character and every
// encoding family, asserting exact escaped output bytes (TC-009).
// cmark-gfm resolves HTML character references into text nodes
// before the autolink extension post-processes the inline tree, so a
// value like "&#119;ww.evil.example" — which carries no literal
// trigger byte for neutralizeAutolinks to defang — reconstructs its
// trigger after byte-level defanging and would linkify. The
// universal "&" escape closes every family at once: a
// backslash-escaped "&" cannot open a reference, so no encoding,
// known or future, can decode at a non-code-span position.
func TestNeutralizeEntityEncodedAutolinks(t *testing.T) {
	sinks := []struct {
		name string
		fn   func(string) string
	}{
		{"sanitizeHeaderValue", sanitizeHeaderValue},
		{"sanitizeProse", sanitizeProse},
		{"escapeCell", escapeCell},
	}
	for _, sink := range sinks {
		for _, fx := range entityFixtures {
			t.Run(sink.name+"/"+fx.name, func(t *testing.T) {
				got := sink.fn(fx.in)
				if got != fx.want {
					t.Errorf("%s(%q) = %q, want %q", sink.name, fx.in, got, fx.want)
				}
				// Exact-byte equality already implies it; assert the
				// invariant by name so a want-string typo cannot mask
				// a bare reference: every "&" in the emitted bytes
				// must carry a preceding backslash.
				for _, idx := range allIndexes(got, "&") {
					if idx == 0 || got[idx-1] != '\\' {
						t.Errorf("%s(%q) = %q: bare & at byte %d", sink.name, fx.in, got, idx)
					}
				}
			})
		}
	}
}

// TestRenderMarkdown_AutolinkTriggersDefanged is the negative guard for the
// escaper invariant: GFM extended autolinking linkifies bare "www." domains,
// "scheme://" URLs, and dotted email addresses even when brackets and angle
// brackets are escaped, so every sanitizer must defang all three trigger
// classes before escaping. Assertions run on the emitted bytes.
func TestRenderMarkdown_AutolinkTriggersDefanged(t *testing.T) {
	// Render level: a crafted branch name and module path must not survive
	// as bare autolink triggers in the metadata header.
	p := loadPayloadFixture(t, "payload-full.json")
	p.Metadata.Branch = "www.evil.example"
	p.Metadata.ModulePath = "user@host.example"
	out, err := RenderMarkdown(p, AIOutcome{})
	if err != nil {
		t.Fatal(err)
	}
	header := headerOf(out)
	for _, want := range []string{`www\[.\]evil.example`, `user\[@\]host.example`} {
		if !strings.Contains(header, want) {
			t.Errorf("header %q does not contain defanged value %q", header, want)
		}
	}
	for _, bare := range []string{"www.evil.example", "user@host.example", "://"} {
		if strings.Contains(header, bare) {
			t.Errorf("header %q contains bare autolink trigger %q", header, bare)
		}
	}

	// Render level: scheme URLs defang in the header too.
	p = loadPayloadFixture(t, "payload-full.json")
	p.Metadata.Branch = "https://evil.example"
	out, err = RenderMarkdown(p, AIOutcome{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(headerOf(out), `https\[:\]//evil.example`) {
		t.Errorf("header %q does not contain defanged scheme URL", headerOf(out))
	}

	// Unit level: every sink defangs every trigger class, case-insensitively.
	if got, want := sanitizeHeaderValue("branch WWW.EVIL.EXAMPLE"), `branch www\[.\]EVIL.EXAMPLE`; got != want {
		t.Errorf("sanitizeHeaderValue = %q, want %q", got, want)
	}
	if got, want := sanitizeProse("mail user@host.example now"), `mail user\[@\]host.example now`; got != want {
		t.Errorf("sanitizeProse = %q, want %q", got, want)
	}
	if got, want := escapeCell("see www.evil.example and user@host.example"), `see www\[.\]evil.example and user\[@\]host.example`; got != want {
		t.Errorf("escapeCell = %q, want %q", got, want)
	}

	// Code-span immunity: a function name renders verbatim inside its code
	// span, which GFM never autolinks, so names stay readable.
	p = loadPayloadFixture(t, "payload-full.json")
	var report crap.Report
	if err := json.Unmarshal(p.CRAP, &report); err != nil {
		t.Fatalf("decode crap: %v", err)
	}
	report.Scores[0].Function = "www.evil.example"
	report.Summary.WorstCRAP[0].Function = "www.evil.example"
	p.CRAP = mustMarshalReport(t, report)
	out, err = RenderMarkdown(p, AIOutcome{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "`www.evil.example`") {
		t.Error("code-span function name was not rendered verbatim")
	}

	// Code-span immunity extends to entity references: "&#64;" inside a
	// function name must stay verbatim in its backtick span. Code spans
	// never interpret character references and the autolink extension
	// never fires there, so escaping the "&" would only display the
	// backslash.
	report.Scores[0].Function = "parse&#64;Input"
	report.Summary.WorstCRAP[0].Function = "parse&#64;Input"
	p.CRAP = mustMarshalReport(t, report)
	out, err = RenderMarkdown(p, AIOutcome{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "`parse&#64;Input`") {
		t.Error("entity-bearing function name was not rendered verbatim in its code span")
	}
	if strings.Contains(out, "`parse\\&#64;Input`") {
		t.Error("code-span function name must not escape the ampersand; the backslash would display")
	}

	// A dotless address never linkifies under GFM, so it must pass through
	// untouched rather than be mangled.
	if got, want := sanitizeProse("ping user@localhost now"), "ping user@localhost now"; got != want {
		t.Errorf("sanitizeProse = %q, want %q", got, want)
	}
}

// scanUnescapedAmpsOutsideCodeSpans reports every byte offset of an
// "&" that is neither backslash-escaped nor inside code-span content.
//
// Scanner rule (pinned in task 2.2): code spans are identified by a
// backtick toggle over the emitted bytes. The toggle is sound in
// these tests for three reasons — the first by construction, the
// other two asserted as preconditions by assertScannerPreconditions
// rather than assumed: (1) the hostile payloads carry no backtick
// bytes, so no attacker byte shifts the toggle; (2) the fixture data
// contains no "&" byte, so a misaligned toggle window has nothing to
// mask or flag; and (3) the one backtick-bearing fixture value
// ("parse`raw`Literal") embeds an even number of backticks inside
// its double-backtick code span, so the toggle resynchronizes after
// that cell even though it misclassifies the span's interior
// boundaries. Hostile values that DO contain backticks are pinned by
// the byte-exact sink assertions (TestNeutralizeEntityEncodedAutolinks,
// TestRenderMarkdown_AutolinkTriggersDefanged), not by this scanner:
// a stray attacker backtick would shift a toggle's boundaries and
// could mask or misplace a span.
func scanUnescapedAmpsOutsideCodeSpans(out string) []int {
	var bad []int
	inSpan := false
	for i := 0; i < len(out); i++ {
		switch out[i] {
		case '`':
			inSpan = !inSpan
		case '&':
			if !inSpan && (i == 0 || out[i-1] != '\\') {
				bad = append(bad, i)
			}
		}
	}
	return bad
}

// assertScannerPreconditions checks the scanner's soundness premises
// against reality instead of assuming them in a comment: the raw
// fixture bytes must contain no "&" (so a misaligned toggle window
// has nothing to mask or flag), and the rendered output must carry
// an even number of backticks (so the toggle ends resynchronized —
// an odd count would invert it for the remainder of the report).
// Golden fixtures are routinely regenerated and edited, so these
// premises can silently stop holding.
func assertScannerPreconditions(t *testing.T, fixture, out string) {
	t.Helper()
	data, err := os.ReadFile(goldenPath(fixture))
	if err != nil {
		t.Fatalf("reading fixture %s: %v", fixture, err)
	}
	if strings.Contains(string(data), "&") {
		t.Fatalf("scanner premise broken: fixture %s contains an & byte", fixture)
	}
	if n := strings.Count(out, "`"); n%2 != 0 {
		t.Fatalf("scanner premise broken: rendered output carries an odd backtick count (%d); the toggle would not resynchronize", n)
	}
}

// TestRenderMarkdown_EntityEncodedAutolinksDefanged is the
// render-level half of the entity-bypass fix (delta spec Requirement
// 1 scenarios, task 2.2): hostile values in the metadata branch,
// module path, quality-step error string, and coverage-gap
// description — carrying encodings from every family — must render
// with the reference neutralized, and no "&" byte outside code-span
// content may be unescaped anywhere in the emitted report. Two
// renders are required because a failed quality step renders its
// error string as the step-unavailable callout and suppresses the
// gaps table, so the coverage-gap description cell needs a render
// with quality data present.
func TestRenderMarkdown_EntityEncodedAutolinksDefanged(t *testing.T) {
	// Render A — metadata header and callout positions. The hostile
	// values carry no backtick bytes, keeping the scanner's toggle
	// boundaries exactly the code spans tableName emitted.
	p := loadPayloadFixture(t, "payload-full.json")
	p.Metadata.Branch = "&#119;ww.evil.example"                                                    // decimal w
	p.Metadata.ModulePath = "mod&#x2E;ule&sol;repo"                                                // hex dot + named slash
	qualityErr := "mail user&#x40;host.example via &#X3A;//relay and www&#46;evil&#46;example now" // hex at, hex colon, hybrid dots
	p.Errors.Quality = &qualityErr

	out, err := RenderMarkdown(p, AIOutcome{})
	if err != nil {
		t.Fatal(err)
	}
	if want := `Project: mod\&#x2E;ule\&sol;repo · Branch: \&#119;ww.evil.example`; !strings.Contains(headerOf(out), want) {
		t.Errorf("header %q does not contain escaped hostile values %q", headerOf(out), want)
	}
	if want := `> ⚠️ Quality analysis unavailable: mail user\&#x40;host.example via \&#X3A;//relay and www\&#46;evil\&#46;example now`; !strings.Contains(out, want) {
		t.Errorf("output does not contain escaped callout %q", want)
	}
	assertScannerPreconditions(t, "payload-full.json", out)
	if bad := scanUnescapedAmpsOutsideCodeSpans(out); len(bad) > 0 {
		t.Errorf("render A: unescaped & outside code spans at byte offsets %v", bad)
	}

	// Render B — the coverage-gap description cell, with quality
	// data present so the gaps table renders.
	p = loadPayloadFixture(t, "payload-full.json")
	p.Metadata.Branch = "&#119;ww&period;evil&period;example" // hybrid decimal w + named dots
	p.Metadata.ModulePath = "mod&#X2e;ule&commat;repo"        // hex mixed-case dot + named at

	var quality qualityOutput
	if err := json.Unmarshal(p.Quality, &quality); err != nil {
		t.Fatalf("decode quality: %v", err)
	}
	quality.Summary.WorstCoverageTests[0].ContractCoverage.Gaps[0].Description = "see https&colon;//evil.example and user&#64;host.example"
	raw, err := json.Marshal(quality)
	if err != nil {
		t.Fatalf("marshal quality: %v", err)
	}
	p.Quality = raw

	out, err = RenderMarkdown(p, AIOutcome{})
	if err != nil {
		t.Fatal(err)
	}
	if want := `Project: mod\&#X2e;ule\&commat;repo · Branch: \&#119;ww\&period;evil\&period;example`; !strings.Contains(headerOf(out), want) {
		t.Errorf("header %q does not contain escaped hostile values %q", headerOf(out), want)
	}
	if want := `see https\&colon;//evil.example and user\&#64;host.example`; !strings.Contains(out, want) {
		t.Errorf("output does not contain escaped gap description %q", want)
	}
	assertScannerPreconditions(t, "payload-full.json", out)
	if bad := scanUnescapedAmpsOutsideCodeSpans(out); len(bad) > 0 {
		t.Errorf("render B: unescaped & outside code spans at byte offsets %v", bad)
	}

	// Render C — the AI-slot wiring: a validated answer whose summary
	// and recommendations carry entity-encoded triggers must render
	// escaped. The AI summary and AI-recommendations slots are
	// distinct sanitizeProse call sites from the callout path pinned
	// by render A, so a wiring regression there (dropping
	// sanitizeProse from a slot) would otherwise be caught by no
	// entity-bearing render test.
	p = loadPayloadFixture(t, "payload-full.json")
	answer := &AIAnswer{
		Summary:         "see &#119;ww.evil.example for details",
		Recommendations: []string{"Check user&commat;host.example next."},
	}
	out, err = RenderMarkdown(p, AIOutcome{Answer: answer})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`see \&#119;ww.evil.example for details`,
		`Check user\&commat;host.example next.`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output does not contain escaped AI-slot value %q", want)
		}
	}
	assertScannerPreconditions(t, "payload-full.json", out)
	if bad := scanUnescapedAmpsOutsideCodeSpans(out); len(bad) > 0 {
		t.Errorf("render C: unescaped & outside code spans at byte offsets %v", bad)
	}
}

// TestRenderMarkdown_BenignAmpersandParity pins design D4's
// trade-off (delta spec Requirement 4, task 3.1): a benign "&" that
// cannot continue a character reference renders identically to
// before ("R\&D" displays as "R&D" under cmark-gfm), while a value
// that IS a valid reference now renders verbatim ("a\&amp;b"
// displays as "a&amp;b"; the pre-change renderer silently decoded it
// to "a&b" inside an identifier). Both spec scenarios are pinned at
// sink level (every non-code-span sink) and at render level.
func TestRenderMarkdown_BenignAmpersandParity(t *testing.T) {
	// Sink level: R&D and a&amp;b through every non-code-span sink.
	cases := []struct{ in, want string }{
		{"R&D", `R\&D`},
		{"a&amp;b", `a\&amp;b`},
	}
	sinks := []struct {
		name string
		fn   func(string) string
	}{
		{"sanitizeHeaderValue", sanitizeHeaderValue},
		{"sanitizeProse", sanitizeProse},
		{"escapeCell", escapeCell},
	}
	for _, sink := range sinks {
		for _, tc := range cases {
			if got := sink.fn(tc.in); got != tc.want {
				t.Errorf("%s(%q) = %q, want %q", sink.name, tc.in, got, tc.want)
			}
		}
	}

	// Render level: the metadata header positions from the spec
	// scenarios — module path R&D, branch a&amp;b.
	p := loadPayloadFixture(t, "payload-full.json")
	p.Metadata.ModulePath = "R&D"
	p.Metadata.Branch = "a&amp;b"
	out, err := RenderMarkdown(p, AIOutcome{})
	if err != nil {
		t.Fatal(err)
	}
	if want := `Project: R\&D · Branch: a\&amp;b`; !strings.Contains(headerOf(out), want) {
		t.Errorf("header %q does not contain benign-parity values %q", headerOf(out), want)
	}
}

// TestRenderMarkdown_MetadataNewlinesFlattened pins the header invariant's
// new-line clause at render level: every line-break flavor in a branch name
// flattens to single spaces in the metadata header.
func TestRenderMarkdown_MetadataNewlinesFlattened(t *testing.T) {
	p := loadPayloadFixture(t, "payload-full.json")
	p.Metadata.Branch = "line1\nline2\r\nline3\rline4"
	out, err := RenderMarkdown(p, AIOutcome{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(headerOf(out), "Branch: line1 line2 line3 line4") {
		t.Errorf("header %q does not flatten multi-line branch to spaces", headerOf(out))
	}
}

// TestRenderMarkdown_HostileDataEscaped is the render-level guard: hostile
// analysis data must not produce a link or an HTML tag in a table cell, a
// callout, or a recommendation line.
func TestRenderMarkdown_HostileDataEscaped(t *testing.T) {
	const hostile = "<script>[x](https://evil.example)"

	p := loadPayloadFixture(t, "payload-full.json")

	var report crap.Report
	if err := json.Unmarshal(p.CRAP, &report); err != nil {
		t.Fatalf("decode crap: %v", err)
	}
	report.Scores[0].Function = hostile
	report.Summary.WorstCRAP[0].Function = hostile
	p.CRAP = mustMarshalReport(t, report)

	callout := hostile + "\nsecond line"
	p.Errors.Docscan = &callout

	assessment := Assess(p)
	if len(assessment.Recommendations) > 0 {
		assessment.Recommendations[0].Text = "Add tests for " + hostile + "."
	}
	p.Assessment = &assessment

	// Render the mutated payload directly: renderFixture reloads the
	// fixture, which would discard the hostile data.
	sections, err := ParseSections([]string{"crap", "health"})
	if err != nil {
		t.Fatalf("ParseSections: %v", err)
	}
	p.Metadata.Sections = sections.List()

	out, err := RenderMarkdown(p, AIOutcome{})
	if err != nil {
		t.Fatalf("RenderMarkdown: %v", err)
	}

	// The escaped form keeps the text visible (the renderer never drops
	// data), but every bracket and angle bracket must be backslash-escaped
	// so it can neither open a link nor start an HTML tag.
	for _, r := range []string{"[", "]", "<"} {
		for i := 0; i < len(out); i++ {
			if string(out[i]) != r {
				continue
			}
			if i == 0 || out[i-1] != '\\' {
				t.Errorf("unescaped %q at offset %d in:\n%s", r, i, out)
				break
			}
		}
	}

	if !strings.Contains(out, "script") {
		t.Error("expected the hostile name to survive escaping, not be dropped")
	}
}

// mustMarshalReport marshals a crap.Report for a renderer test.
func mustMarshalReport(t *testing.T, report crap.Report) json.RawMessage {
	t.Helper()

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal crap report: %v", err)
	}

	return data
}

func TestRenderMarkdown_FailedStepCallout(t *testing.T) {
	out := renderFixture(t, "payload-partial.json", AIOutcome{})
	if !strings.Contains(out, "## 🧪 Quality Summary") {
		t.Error("failed quality section must keep its heading")
	}
	want := "> ⚠️ Quality analysis unavailable: resolving packages for quality: go list ./...: exit status 1"
	if !strings.Contains(out, want) {
		t.Errorf("missing failed-step callout %q", want)
	}
}

func TestRenderMarkdown_CalloutBounded(t *testing.T) {
	p := loadPayloadFixture(t, "payload-full.json")
	detail := "first line\rsecond line"
	long := strings.Repeat("é", 200)
	errorText := detail
	p.Errors.CRAP = &errorText
	out, err := RenderMarkdown(p, AIOutcome{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "> ⚠️ CRAP analysis unavailable: first line\n") {
		t.Error("callout did not take only the first line")
	}

	errorText = long
	out, err = RenderMarkdown(p, AIOutcome{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "> ⚠️ CRAP analysis unavailable: "+strings.Repeat("é", 100)) {
		t.Error("callout did not truncate to 200 bytes on a rune boundary")
	}
}

func TestRenderMarkdown_GazeCRAPloadUnavailable(t *testing.T) {
	out := renderFixture(t, "payload-partial.json", AIOutcome{}, "crap")
	if !strings.Contains(out, "| GazeCRAPload | ⚪ N/A |") {
		t.Error("absent GazeCRAPload must render ⚪ N/A")
	}
}

func TestRenderMarkdown_ZeroTestsCallout(t *testing.T) {
	p := loadPayloadFixture(t, "payload-full.json")
	var quality struct {
		Reports []json.RawMessage `json:"quality_reports"`
		Summary map[string]any    `json:"quality_summary"`
	}
	if err := json.Unmarshal(p.Quality, &quality); err != nil {
		t.Fatal(err)
	}
	quality.Summary["total_tests"] = 0
	quality.Summary["worst_coverage_tests"] = []any{}
	raw, err := json.Marshal(quality)
	if err != nil {
		t.Fatal(err)
	}
	p.Quality = raw
	p.Metadata.Sections = sectionSetOf(SectionQuality).List()

	out, err := RenderMarkdown(p, AIOutcome{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "> ⚠️ Quality analysis found 0 tests for the selected packages.") {
		t.Error("missing zero-tests callout")
	}
	if strings.Contains(out, "### Worst Tests by Contract Coverage") {
		t.Error("zero-tests report must not render the worst-tests table")
	}
}

func TestRenderMarkdown_DocumentationNoFiles(t *testing.T) {
	p := loadPayloadFixture(t, "payload-full.json")
	p.Docscan = json.RawMessage(`{"documents":[],"api_coverage":null}`)
	p.Metadata.Sections = sectionSetOf(SectionHealth).List()
	out, err := RenderMarkdown(p, AIOutcome{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Documentation: no files found.\n") {
		t.Error("missing no-documents line")
	}
}

func TestRenderMarkdown_ClosedVocabulary(t *testing.T) {
	allowed := map[rune]bool{
		'🔍': true, '📊': true, '🧪': true, '🏷': true, '🏥': true,
		'🟢': true, '🟡': true, '🔴': true, '⚪': true, '⚠': true,
	}
	out := stripAISlots(renderFixture(t, "payload-full.json", AIOutcome{Answer: validAnswer()}))
	for _, r := range out {
		if unicode.Is(unicode.So, r) && !allowed[r] {
			t.Errorf("report contains emoji %q outside the closed vocabulary", r)
		}
	}
}

func TestRenderMarkdown_AISlotPlacement(t *testing.T) {
	out := renderFixture(t, "payload-full.json", AIOutcome{Answer: validAnswer()})
	summarySlot := strings.Index(out, "<!-- gaze:ai-summary -->")
	crapHeading := strings.Index(out, "## 📊 CRAP Summary")
	recsSlot := strings.Index(out, "<!-- gaze:ai-recommendations -->")
	deterministicRecs := strings.Index(out, "### Recommendations")

	if summarySlot < 0 || crapHeading < 0 || recsSlot < 0 || deterministicRecs < 0 {
		t.Fatalf("missing markers: summary=%d crap=%d recs=%d deterministic=%d", summarySlot, crapHeading, recsSlot, deterministicRecs)
	}
	if summarySlot > crapHeading {
		t.Error("summary slot must precede the first section")
	}
	if recsSlot < deterministicRecs {
		t.Error("recommendations slot must follow the deterministic recommendations")
	}
	if !strings.Contains(out, "1. Decompose `runQuality`.\n2. Add tests for `parseConfig`.\n") {
		t.Error("AI recommendations are not rendered as a numbered list")
	}
}

func TestRenderMarkdown_AIRecommendationsSlotWhenHealthNotSelected(t *testing.T) {
	out := renderFixture(t, "payload-full.json", AIOutcome{Answer: validAnswer()}, "crap")
	if !strings.HasSuffix(strings.TrimRight(out, "\n"), "<!-- /gaze:ai-recommendations -->") {
		t.Error("recommendations slot must sit at the end when health is not selected")
	}
	if strings.Contains(out, "### Recommendations") {
		t.Error("deterministic recommendations must not render without the health section")
	}
}

func TestRenderMarkdown_ZeroOutcomeOmitsSlots(t *testing.T) {
	out := renderFixture(t, "payload-full.json", AIOutcome{})
	for _, marker := range []string{"gaze:ai-summary", "gaze:ai-recommendations", "AI-Prioritized"} {
		if strings.Contains(out, marker) {
			t.Errorf("zero outcome must omit AI slots; found %q", marker)
		}
	}
}

func TestRenderMarkdown_RejectedOutcome(t *testing.T) {
	outcome := AIOutcome{FailureNote: "AI narrative rejected: summary exceeds length bound (1342 > 1200 characters)"}
	out := renderFixture(t, "payload-full.json", outcome)
	if !strings.Contains(out, "> ⚠️ AI narrative rejected: summary exceeds length bound (1342 > 1200 characters)") {
		t.Error("missing rejection callout")
	}
	if strings.Contains(out, "gaze:ai-recommendations") {
		t.Error("rejection must not render a recommendations slot")
	}
	if strings.Contains(out, "### Summary\n\nThe project") {
		t.Error("rejection must not render an answer body")
	}
}

func TestRenderMarkdown_UnavailableOutcome(t *testing.T) {
	outcome := AIOutcome{FailureNote: "AI narrative unavailable: adapter exited with status 1"}
	out := renderFixture(t, "payload-full.json", outcome)
	if !strings.Contains(out, "> ⚠️ AI narrative unavailable: adapter exited with status 1") {
		t.Error("missing unavailable callout")
	}
	if strings.Contains(out, "gaze:ai-recommendations") {
		t.Error("unavailable outcome must not render a recommendations slot")
	}
}

func TestRenderMarkdown_NoRecommendations(t *testing.T) {
	p := loadPayloadFixture(t, "payload-full.json")
	p.Summary = ReportSummary{TotalFunctions: 1}
	p.CRAP = json.RawMessage(`{"scores":[],"summary":{"total_functions":1,"avg_complexity":1,"avg_line_coverage":100,"avg_crap":1,"crapload":0,"crap_threshold":15,"quadrant_counts":{},"fix_strategy_counts":{},"worst_crap":[],"worst_gaze_crap":[]}}`)
	p.Assessment = nil
	out, err := RenderMarkdown(p, AIOutcome{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No recommendations — no functions at or above the CRAP threshold.") {
		t.Error("missing empty-recommendations line")
	}
}
