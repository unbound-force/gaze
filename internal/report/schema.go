package report

// Schema is the JSON Schema (Draft 2020-12) for the Gaze analysis
// JSON output. It documents the structure returned by WriteJSON.
const Schema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://github.com/jflowers/gaze/analysis-report.schema.json",
  "title": "Gaze Analysis Report",
  "description": "Output schema for gaze analyze --format=json",
  "type": "object",
  "required": ["version", "results"],
  "properties": {
    "version": {
      "type": "string",
      "description": "Schema version (semver)"
    },
    "results": {
      "type": "array",
      "items": { "$ref": "#/$defs/AnalysisResult" }
    }
  },
  "$defs": {
    "AnalysisResult": {
      "type": "object",
      "required": ["target", "side_effects", "metadata"],
      "properties": {
        "target": { "$ref": "#/$defs/FunctionTarget" },
        "side_effects": {
          "type": "array",
          "items": { "$ref": "#/$defs/SideEffect" }
        },
        "metadata": { "$ref": "#/$defs/Metadata" }
      }
    },
    "FunctionTarget": {
      "type": "object",
      "required": ["package", "function", "signature", "location"],
      "properties": {
        "package": {
          "type": "string",
          "description": "Full import path"
        },
        "function": {
          "type": "string",
          "description": "Function or method name"
        },
        "receiver": {
          "type": "string",
          "description": "Receiver type for methods (e.g., '*Store')"
        },
        "signature": {
          "type": "string",
          "description": "Full function signature"
        },
        "location": {
          "type": "string",
          "description": "Source position (file:line:col)"
        }
      }
    },
    "SideEffect": {
      "type": "object",
      "required": ["id", "type", "tier", "location", "description", "target"],
      "properties": {
        "id": {
          "type": "string",
          "description": "Stable identifier (se-XXXXXXXX)"
        },
        "type": {
          "type": "string",
          "description": "Side effect type from taxonomy",
          "enum": [
            "ReturnValue", "ErrorReturn", "SentinelError",
            "ReceiverMutation", "PointerArgMutation",
            "SliceMutation", "MapMutation", "GlobalMutation",
            "WriterOutput", "HttpResponseWrite",
            "ChannelSend", "ChannelClose", "DeferredReturnMutation",
            "FileSystemWrite", "FileSystemDelete", "FileSystemMeta",
            "DatabaseWrite", "DatabaseTransaction",
            "GoroutineSpawn", "Panic", "CallbackInvocation",
            "LogWrite", "ContextCancellation",
            "StdoutWrite", "StderrWrite", "EnvVarMutation",
            "MutexOp", "WaitGroupOp", "AtomicOp",
            "TimeDependency", "ProcessExit", "RecoverBehavior",
            "ReflectionMutation", "UnsafeMutation", "CgoCall",
            "FinalizerRegistration", "SyncPoolOp",
            "ClosureCaptureMutation"
          ]
        },
        "tier": {
          "type": "string",
          "enum": ["P0", "P1", "P2", "P3", "P4"],
          "description": "Priority tier"
        },
        "location": {
          "type": "string",
          "description": "Source position"
        },
        "description": {
          "type": "string",
          "description": "Human-readable explanation"
        },
        "target": {
          "type": "string",
          "description": "Affected entity (field, variable, type, etc.)"
        }
      }
    },
    "Metadata": {
      "type": "object",
      "required": ["gaze_version", "go_version", "duration_ms"],
      "properties": {
        "gaze_version": { "type": "string" },
        "go_version": { "type": "string" },
        "duration_ms": {
          "type": "integer",
          "description": "Analysis duration in milliseconds"
        },
        "warnings": {
          "oneOf": [
            { "type": "array", "items": { "type": "string" } },
            { "type": "null" }
          ],
          "description": "Analysis warnings, if any"
        }
      }
    }
  }
}`
