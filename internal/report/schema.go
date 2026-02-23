// Package report provides output formatters for Gaze analysis results.
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
          "description": "Function or method name. The value '<package>' indicates package-level declarations (e.g., sentinel errors) not associated with a specific function."
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
            "WriterOutput", "HTTPResponseWrite",
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
        },
        "classification": {
          "$ref": "#/$defs/Classification",
          "description": "Contractual classification (only present when --classify is used)"
        }
      }
    },
    "Classification": {
      "type": "object",
      "required": ["label", "confidence", "signals"],
      "properties": {
        "label": {
          "type": "string",
          "enum": ["contractual", "incidental", "ambiguous"],
          "description": "Classification result"
        },
        "confidence": {
          "type": "integer",
          "minimum": 0,
          "maximum": 100,
          "description": "Confidence score (0-100)"
        },
        "signals": {
          "type": "array",
          "items": { "$ref": "#/$defs/Signal" },
          "description": "Evidence signals that contributed to the score"
        },
        "reasoning": {
          "type": "string",
          "description": "Human-readable summary of the classification"
        }
      }
    },
    "Signal": {
      "type": "object",
      "required": ["source", "weight"],
      "properties": {
        "source": {
          "type": "string",
          "description": "Signal source (e.g., 'interface', 'caller', 'naming', 'godoc', 'readme')"
        },
        "weight": {
          "type": "integer",
          "description": "Numeric contribution to confidence score (can be negative)"
        },
        "source_file": {
          "type": "string",
          "description": "File path that provided this signal (verbose mode only)"
        },
        "excerpt": {
          "type": "string",
          "description": "Short quote from the source (verbose mode only)"
        },
        "reasoning": {
          "type": "string",
          "description": "Explanation of why this signal was applied (verbose mode only)"
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
        "timestamp": {
          "type": "string",
          "format": "date-time",
          "description": "ISO 8601 timestamp of when the analysis was run"
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

// QualitySchema is the JSON Schema (Draft 2020-12) for the Gaze
// quality analysis JSON output. It documents the structure returned
// by quality.WriteJSON.
const QualitySchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://github.com/jflowers/gaze/quality-report.schema.json",
  "title": "Gaze Quality Report",
  "description": "Output schema for gaze quality --format=json",
  "type": "object",
  "required": ["quality_reports", "quality_summary"],
  "properties": {
    "quality_reports": {
      "type": "array",
      "items": { "$ref": "#/$defs/QualityReport" }
    },
    "quality_summary": { "$ref": "#/$defs/PackageSummary" }
  },
  "$defs": {
    "QualityReport": {
      "type": "object",
      "required": [
        "test_function", "test_location", "target_function",
        "contract_coverage", "over_specification",
        "assertion_detection_confidence", "metadata"
      ],
      "properties": {
        "test_function": {
          "type": "string",
          "description": "Name of the test function"
        },
        "test_location": {
          "type": "string",
          "description": "Source position (file:line)"
        },
        "target_function": { "$ref": "#/$defs/FunctionTarget" },
        "contract_coverage": { "$ref": "#/$defs/ContractCoverage" },
        "over_specification": { "$ref": "#/$defs/OverSpecificationScore" },
        "ambiguous_effects": {
          "oneOf": [
            { "type": "array", "items": { "$ref": "#/$defs/SideEffectRef" } },
            { "type": "null" }
          ],
          "description": "Effects excluded from metrics due to ambiguous classification"
        },
        "unmapped_assertions": {
          "oneOf": [
            { "type": "array", "items": { "$ref": "#/$defs/AssertionMapping" } },
            { "type": "null" }
          ],
          "description": "Assertions that could not be linked to any side effect"
        },
        "assertion_detection_confidence": {
          "type": "integer",
          "minimum": 0,
          "maximum": 100,
          "description": "Fraction of test assertions successfully pattern-matched (0-100)"
        },
        "metadata": { "$ref": "#/$defs/Metadata" }
      }
    },
    "FunctionTarget": {
      "type": "object",
      "required": ["package", "function", "signature", "location"],
      "properties": {
        "package": { "type": "string" },
        "function": { "type": "string" },
        "receiver": { "type": "string" },
        "signature": { "type": "string" },
        "location": { "type": "string" }
      }
    },
    "ContractCoverage": {
      "type": "object",
      "required": ["percentage", "covered_count", "total_contractual"],
      "properties": {
        "percentage": {
          "type": "number",
          "minimum": 0,
          "maximum": 100,
          "description": "Coverage ratio (0-100)"
        },
        "covered_count": {
          "type": "integer",
          "description": "Number of contractual effects asserted on"
        },
        "total_contractual": {
          "type": "integer",
          "description": "Total number of contractual effects"
        },
        "gaps": {
          "oneOf": [
            { "type": "array", "items": { "$ref": "#/$defs/SideEffectRef" } },
            { "type": "null" }
          ],
          "description": "Contractual effects NOT asserted on"
        }
      }
    },
    "OverSpecificationScore": {
      "type": "object",
      "required": ["count", "ratio"],
      "properties": {
        "count": {
          "type": "integer",
          "description": "Number of incidental side effects asserted on"
        },
        "ratio": {
          "type": "number",
          "minimum": 0,
          "maximum": 1,
          "description": "Incidental assertions / total assertions"
        },
        "incidental_assertions": {
          "oneOf": [
            { "type": "array", "items": { "$ref": "#/$defs/AssertionMapping" } },
            { "type": "null" }
          ]
        },
        "suggestions": {
          "oneOf": [
            { "type": "array", "items": { "type": "string" } },
            { "type": "null" }
          ],
          "description": "Actionable advice per incidental assertion"
        }
      }
    },
    "AssertionMapping": {
      "type": "object",
      "required": ["assertion_location", "assertion_type", "confidence"],
      "properties": {
        "assertion_location": {
          "type": "string",
          "description": "Source position (file:line)"
        },
        "assertion_type": {
          "type": "string",
          "enum": ["equality", "error_check", "diff_check", "custom"],
          "description": "Kind of assertion"
        },
        "side_effect_id": {
          "type": "string",
          "description": "Stable ID of the mapped side effect"
        },
        "confidence": {
          "type": "integer",
          "minimum": 0,
          "maximum": 100,
          "description": "Mapping confidence (0-100)"
        }
      }
    },
    "SideEffectRef": {
      "type": "object",
      "required": ["id", "type", "tier", "description"],
      "properties": {
        "id": { "type": "string" },
        "type": { "type": "string" },
        "tier": { "type": "string" },
        "location": { "type": "string" },
        "description": { "type": "string" },
        "target": { "type": "string" }
      }
    },
    "PackageSummary": {
      "type": "object",
      "required": [
        "total_tests", "average_contract_coverage",
        "total_over_specifications", "assertion_detection_confidence"
      ],
      "properties": {
        "total_tests": { "type": "integer" },
        "average_contract_coverage": {
          "type": "number",
          "description": "Mean coverage across tests (0-100)"
        },
        "total_over_specifications": { "type": "integer" },
        "worst_coverage_tests": {
          "oneOf": [
            { "type": "array", "items": { "$ref": "#/$defs/QualityReport" } },
            { "type": "null" }
          ],
          "description": "Bottom 5 tests by coverage"
        },
        "assertion_detection_confidence": { "type": "integer" }
      }
    },
    "Metadata": {
      "type": "object",
      "required": ["gaze_version", "go_version", "duration_ms"],
      "properties": {
        "gaze_version": { "type": "string" },
        "go_version": { "type": "string" },
        "duration_ms": { "type": "integer" },
        "timestamp": {
          "type": "string",
          "format": "date-time"
        },
        "warnings": {
          "oneOf": [
            { "type": "array", "items": { "type": "string" } },
            { "type": "null" }
          ]
        }
      }
    }
  }
}`
