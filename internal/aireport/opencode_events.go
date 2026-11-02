package aireport

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

// errNoFinalText reports that the stream carried no final assistant
// text for the last step.
var errNoFinalText = errors.New("opencode produced no final assistant text")

// maxAdapterErrorBytes bounds the error text lifted from a failed
// opencode run.
const maxAdapterErrorBytes = 512

// openCodeEvent is one line of the `opencode run --format json` event
// stream. Only the fields Gaze consumes are decoded; every other field
// is tolerated.
type openCodeEvent struct {
	Type  string          `json:"type"`
	Part  *openCodePart   `json:"part,omitempty"`
	Error json.RawMessage `json:"error,omitempty"`
}

// openCodePart is the payload an event may carry.
type openCodePart struct {
	ID        string `json:"id"`
	MessageID string `json:"messageID"`
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	Synthetic bool   `json:"synthetic,omitempty"`
	Ignored   bool   `json:"ignored,omitempty"`
	Tool      string `json:"tool,omitempty"`
	State     *struct {
		Status string `json:"status"`
	} `json:"state,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// openCodeError is the structured error an `error` event carries.
type openCodeError struct {
	Name    string `json:"name"`
	Message string `json:"message"`
	// Data carries the provider error body, which is where opencode puts
	// the human-readable message for API failures.
	Data struct {
		Message string `json:"message"`
	} `json:"data"`
}

// textPart is one assistant text part recorded from the stream.
type textPart struct {
	id        string
	messageID string
	text      string
}

// eventState accumulates the parts of the stream that decide the final
// answer.
type eventState struct {
	texts       []textPart
	index       map[string]int
	lastStepMsg string
}

// extractFinalText returns the final assistant text of an
// `opencode run --format json` event stream.
//
// Only the text belonging to the last completed step is returned, so
// narration emitted between tool calls can never reach the report. The
// function is pure and never falls back to the raw stream. The event
// shape and message selection follow the opcode-1.19.0 stream verified
// in research R1.
func extractFinalText(stdout []byte) (string, error) {
	events, err := parseEvents(stdout)
	if err != nil {
		return "", err
	}

	state := &eventState{}
	for i := range events {
		if err := state.handle(&events[i]); err != nil {
			return "", err
		}
	}

	return state.finalText()
}

// parseEvents decodes one event per non-blank line.
//
// Lines that are not JSON are skipped: the stream framing is
// line-oriented, so a stray log line must not discard the events around
// it. A stream with no decodable event is a failure.
func parseEvents(stdout []byte) ([]openCodeEvent, error) {
	lines := bytes.Split(stdout, []byte("\n"))
	events := make([]openCodeEvent, 0, len(lines))

	for _, line := range lines {
		line = bytes.TrimRight(line, "\r")
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var ev openCodeEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			continue
		}
		events = append(events, ev)
	}

	if len(events) == 0 {
		return nil, fmt.Errorf(
			"opencode produced no JSON events (expected --format json event stream; pinned opencode-ai 1.19.0)",
		)
	}

	return events, nil
}

// handle folds one event into the accumulated state.
func (s *eventState) handle(ev *openCodeEvent) error {
	switch ev.Type {
	case "error":
		return errorEvent(ev)
	case "text":
		s.textEvent(ev)
	case "tool_use":
		return toolUseEvent(ev)
	case "step_finish":
		if ev.Part != nil {
			s.lastStepMsg = ev.Part.MessageID
		}
	}

	return nil
}

// textEvent records an assistant text part, ignoring parts the runtime
// marks as synthetic or ignored. A repeated part id replaces the earlier
// text in place.
func (s *eventState) textEvent(ev *openCodeEvent) {
	if ev.Part == nil || ev.Part.Synthetic || ev.Part.Ignored {
		return
	}

	part := textPart{id: ev.Part.ID, messageID: ev.Part.MessageID, text: ev.Part.Text}
	if part.id == "" {
		s.texts = append(s.texts, part)
		return
	}

	if s.index == nil {
		s.index = make(map[string]int, len(s.texts)+1)
	}

	if at, ok := s.index[part.id]; ok {
		s.texts[at] = part
		return
	}

	s.index[part.id] = len(s.texts)
	s.texts = append(s.texts, part)
}

// toolUseEvent fails the run when the agent executed a tool.
//
// The gaze-headless agent denies every tool, so a completed tool call
// means the run did not follow its contract and its output cannot be
// trusted.
func toolUseEvent(ev *openCodeEvent) error {
	if ev.Part == nil || ev.Part.State == nil || ev.Part.State.Status != "completed" {
		return nil
	}

	return fmt.Errorf("opencode executed tool %q although the gaze-headless agent denies all tools", ev.Part.Tool)
}

// errorEvent converts a runtime error event into an error.
//
// The message is scrubbed of inherited secret values and bounded, so a
// failed run cannot leak an API key or an unbounded transcript.
func errorEvent(ev *openCodeEvent) error {
	name, message := "unknown", "no error detail"

	var payload openCodeError
	if len(ev.Error) > 0 && json.Unmarshal(ev.Error, &payload) == nil {
		if payload.Name != "" {
			name = payload.Name
		}
		if payload.Data.Message != "" {
			message = payload.Data.Message
		} else if payload.Message != "" {
			message = payload.Message
		}
	}

	detail := redactEnvValues(name+": "+message, os.Environ())
	return fmt.Errorf("opencode reported an error: %s", truncateBytes(detail, maxAdapterErrorBytes))
}

// finalText joins the text of the final step in first-seen order.
func (s *eventState) finalText() (string, error) {
	target := s.lastStepMsg
	if target == "" && len(s.texts) > 0 {
		target = s.texts[len(s.texts)-1].messageID
	}

	var b strings.Builder
	for _, part := range s.texts {
		if part.messageID != target {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(part.text)
	}

	text := strings.TrimSpace(b.String())
	if text == "" {
		return "", errNoFinalText
	}

	return text, nil
}

// truncateBytes bounds s to max bytes, cutting on a rune boundary.
func truncateBytes(s string, max int) string {
	if len(s) <= max {
		return s
	}

	cut := max
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}

	return s[:cut]
}
