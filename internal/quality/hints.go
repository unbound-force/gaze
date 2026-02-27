package quality

import (
	"fmt"

	"github.com/unbound-force/gaze/internal/taxonomy"
)

// hintForEffect returns a Go code snippet suggesting how to write an
// assertion for the given side effect. The hint is a short, pasteable
// example that uses the effect's type and target to be as specific as
// possible. P0 and P1 effect types receive tailored hints; P2-P4 types
// receive a generic template.
func hintForEffect(e taxonomy.SideEffect) string {
	target := e.Target

	switch e.Type {
	// P0 — Must Detect.
	case taxonomy.ReturnValue:
		return "got := target(); // assert got == expected"

	case taxonomy.ErrorReturn:
		return "if err != nil { t.Fatal(err) }"

	case taxonomy.SentinelError:
		return "if !errors.Is(err, ExpectedErr) { t.Errorf(\"got %v, want %v\", err, ExpectedErr) }"

	case taxonomy.ReceiverMutation:
		if target != "" {
			return fmt.Sprintf("// assert receiver.%s after calling target()", target)
		}
		return "// assert receiver state after calling target()"

	case taxonomy.PointerArgMutation:
		if target != "" {
			return fmt.Sprintf("// assert *%s after calling target()", target)
		}
		return "// assert pointer argument state after calling target()"

	// P1 — High Value.
	case taxonomy.SliceMutation:
		return "// assert slice contents after calling target()"

	case taxonomy.MapMutation:
		return "// assert map contents after calling target()"

	case taxonomy.GlobalMutation:
		if target != "" {
			return fmt.Sprintf("// assert %s (global) after calling target()", target)
		}
		return "// assert global state after calling target()"

	case taxonomy.WriterOutput:
		if target != "" {
			return fmt.Sprintf("// assert bytes written to %s after calling target()", target)
		}
		return "// assert bytes written to writer after calling target()"

	case taxonomy.HTTPResponseWrite:
		return "// assert HTTP response status and body after calling target()"

	case taxonomy.ChannelSend:
		if target != "" {
			return fmt.Sprintf("// assert value sent on %s after calling target()", target)
		}
		return "// assert value sent on channel after calling target()"

	case taxonomy.ChannelClose:
		if target != "" {
			return fmt.Sprintf("// assert %s is closed after calling target()", target)
		}
		return "// assert channel is closed after calling target()"

	case taxonomy.DeferredReturnMutation:
		return "// assert named return value after calling target() (check via defer or named returns)"

	// P2-P4 — Generic fallback for less common effect types.
	default:
		return fmt.Sprintf("// assert %s side effect of target()", e.Type)
	}
}
