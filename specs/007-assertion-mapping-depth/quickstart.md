# Quickstart: Verifying Assertion Mapping Depth

**Date**: 2026-02-27
**Feature**: [spec.md](spec.md) | [plan.md](plan.md)

## How to Observe the Improvement

### 1. Run the Mapping Accuracy Test

The ratchet test measures mapping accuracy across all test fixtures:

```bash
go test -race -count=1 -run TestSC003_MappingAccuracy ./internal/quality/...
```

**Before this change**: ~73.8% (31/42 mapped), baseline floor 70%.
**After this change**: >= 85% mapped, baseline floor raised.

### 2. Run Quality on a Package with Selector Patterns

The `multilib` fixture has testify-style selector assertions
(`user.Name`, `user.Email`):

```bash
gaze quality --format=json ./internal/quality/testdata/src/multilib
```

Look for `contract_coverage.percentage` in the output. Tests like
`TestNewUser_Testify` should now show covered `ReturnValue` effects
because `user.Name` resolves to the `user` variable traced from
`user, err := NewUser(...)`.

### 3. Compare Before/After on Real Packages

Run `gaze quality` on packages known to have low contract coverage:

```bash
# crap package (currently 68.8%)
gaze quality ./internal/crap

# classify package (currently 66.7%)
gaze quality ./internal/classify

# docscan package (currently 58.3%)
gaze quality ./internal/docscan
```

After the change, these packages should show higher contract
coverage percentages because assertions on struct fields and
`len()` calls now map to their corresponding `ReturnValue` effects.

### 4. Verify No Regressions

Run the full test suite to confirm no existing tests break:

```bash
go test -race -count=1 -short ./...
```

All 11 packages should pass. Pay particular attention to the
`internal/quality` tests which include regression assertions for
existing mappings.

### 5. Check Confidence Values in JSON Output

Run quality analysis with JSON output and inspect the `confidence`
field on assertion mappings:

```bash
gaze quality --format=json ./internal/crap | python3 -c "
import json, sys
data = json.load(sys.stdin)
for r in (data.get('quality_reports') or []):
    for m in (r.get('contract_coverage', {}).get('mapped_assertions') or []):
        print(f'{m[\"confidence\"]:>3} {m[\"assertion_type\"]:>20} {m[\"assertion_location\"]}')
"
```

**Expected**: Direct identity matches show confidence `75`. Indirect
matches (through selector, index, or builtin) show confidence `65`.
