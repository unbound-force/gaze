package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/unbound-force/gaze/internal/config"
	"github.com/unbound-force/gaze/internal/crap"
	"github.com/unbound-force/gaze/internal/protocol"
)

// Providers holds the constructed provider adapters ready to be
// passed to crap.Options.
type Providers struct {
	// Complexity is the external complexity provider.
	Complexity crap.ComplexityProvider

	// LineCoverage is the external line coverage provider.
	LineCoverage crap.LineCoverageProvider

	// ContractCoverage is the external contract coverage provider.
	// Nil when test_mapping capability is false.
	ContractCoverage crap.ContractCoverageProvider

	// SideEffects provides access to the side effect analyzer for
	// commands (like gaze quality) that need direct access to
	// per-function analysis results beyond what the contract
	// coverage provider exposes.
	SideEffects *ExternalSideEffectAnalyzer

	// Capabilities is the analyzer's declared capabilities from
	// the initialize handshake.
	Capabilities protocol.Capabilities

	// AnalyzerName is the human-readable analyzer name.
	AnalyzerName string

	// Language is the primary language the analyzer targets.
	Language string

	// LanguageVersion is the runtime/compiler version for the
	// target language.
	LanguageVersion string
}

// Session manages the full protocol lifecycle with an external
// analyzer: spawn binary, initialize (get capabilities), construct
// provider adapters, and shutdown.
//
// Design decision D2: Protocol lifecycle matches Issue #95.
type Session struct {
	client   *protocol.Client
	binary   string
	args     []string
	rootDir  string
	patterns []string
	stderr   io.Writer
	config   *config.GazeConfig

	// caps is populated after Initialize.
	caps     protocol.Capabilities
	language string
	initDone bool
}

// NewSession creates a new session for the given analyzer binary.
// The binary is not spawned until Initialize is called. The cfg
// parameter provides classification thresholds for ComputeScore;
// when nil, DefaultConfig is used.
func NewSession(binary string, args []string, rootDir string, patterns []string, stderr io.Writer, cfg *config.GazeConfig) *Session {
	return &Session{
		binary:   binary,
		args:     args,
		rootDir:  rootDir,
		patterns: patterns,
		stderr:   stderr,
		config:   cfg,
	}
}

// Initialize spawns the analyzer binary and performs the initialize
// handshake. Returns the Providers struct with all adapters ready
// for use with crap.Options.
//
// The caller must call Close when done to shut down the analyzer.
func (s *Session) Initialize() (*Providers, error) {
	client, err := protocol.NewClient(s.binary, s.args...)
	if err != nil {
		return nil, fmt.Errorf("spawning analyzer %s: %w", s.binary, err)
	}
	s.client = client

	// Initialize handshake with short timeout (D10).
	ctx, cancel := context.WithTimeout(context.Background(), protocol.ShortTimeout)
	defer cancel()

	resp, err := s.client.Call(ctx, protocol.MethodInitialize, protocol.InitializeParams{
		RootPath: s.rootDir,
	})
	if err != nil {
		_ = s.client.Close()
		return nil, fmt.Errorf("initialize handshake: %w", err)
	}
	if resp.Error != nil {
		_ = s.client.Close()
		return nil, fmt.Errorf("initialize error: %s (code %d)", resp.Error.Message, resp.Error.Code)
	}

	var initResult protocol.InitializeResult
	if err := json.Unmarshal(resp.Result, &initResult); err != nil {
		_ = s.client.Close()
		return nil, fmt.Errorf("parsing initialize result: %w", err)
	}

	s.caps = initResult.Capabilities
	s.language = initResult.Language
	s.initDone = true

	// Construct provider adapters.
	complexityProvider := NewExternalComplexityProvider(s.client)
	coverageProvider := NewExternalLineCoverageProvider(s.client)

	sideEffectAnalyzer := NewExternalSideEffectAnalyzer(
		s.client, s.caps, s.rootDir, s.patterns, s.stderr, s.config,
	)

	var contractProvider crap.ContractCoverageProvider
	if s.caps.TestMapping {
		contractProvider = NewExternalContractCoverageProvider(
			s.client, s.caps, sideEffectAnalyzer,
			s.rootDir, s.patterns, s.stderr,
		)
	}

	return &Providers{
		Complexity:       complexityProvider,
		LineCoverage:     coverageProvider,
		ContractCoverage: contractProvider,
		SideEffects:      sideEffectAnalyzer,
		Capabilities:     initResult.Capabilities,
		AnalyzerName:     initResult.AnalyzerName,
		Language:         initResult.Language,
		LanguageVersion:  initResult.LanguageVersion,
	}, nil
}

// DocCoverage calls the doc_coverage protocol method on the external
// analyzer and returns the result. Returns nil when the analyzer does
// not support doc_coverage (capabilities check) or when the session
// has not been initialized. Uses AnalysisTimeout for the call context.
func (s *Session) DocCoverage(ctx context.Context, params protocol.DocCoverageParams) (*protocol.DocCoverageResult, error) {
	if !s.initDone {
		return nil, nil
	}
	if !s.caps.DocCoverage {
		return nil, nil
	}

	// Respect caller-provided deadline; otherwise apply AnalysisTimeout.
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, protocol.AnalysisTimeout)
		defer cancel()
	}

	result, err := callAndUnmarshal[protocol.DocCoverageResult](ctx, s.client, protocol.MethodDocCoverage, params)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// Analyze calls the analyze protocol method on the external analyzer
// and returns the result. Returns nil when the session has not been
// initialized. Uses AnalysisTimeout when the caller-provided context
// has no deadline.
func (s *Session) Analyze(ctx context.Context, params protocol.AnalyzeParams) (*protocol.AnalyzeResult, error) {
	if !s.initDone {
		return nil, nil
	}

	// Respect caller-provided deadline; otherwise apply AnalysisTimeout.
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, protocol.AnalysisTimeout)
		defer cancel()
	}

	result, err := callAndUnmarshal[protocol.AnalyzeResult](ctx, s.client, protocol.MethodAnalyze, params)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// Language returns the target language declared by the analyzer during
// initialization. Returns empty string if the session has not been
// initialized.
func (s *Session) Language() string {
	if !s.initDone {
		return ""
	}
	return s.language
}

// Close sends a shutdown request to the analyzer and waits for the
// subprocess to exit. Safe to call even if Initialize was not called
// or failed.
func (s *Session) Close() error {
	if s.client == nil {
		return nil
	}
	return s.client.Close()
}
