# Multiline test output assertions should use raw strings or structured normalization

## Summary

Several tests currently validate multiline text output by chaining multiple
`strings.Contains(...)` checks instead of asserting against a single raw-string
fixture or another normalized representation.

Concrete examples:

- [`integration_test.go`](/home/yoske/src/github.com/yoskeoka/vibe-coding-workspace/ww/integration_test.go#L154): `TestInteractiveHelpHidesJSONFlag`
  checks the `ww i --help` output via five separate `strings.Contains(...)`
  assertions plus one negative assertion.
- [`internal/interactive/interactive_test.go`](/home/yoske/src/github.com/yoskeoka/vibe-coding-workspace/ww/internal/interactive/interactive_test.go#L137):
  `TestRunnerDispatchesActionsAndPrintsOverviewOnce` checks the rendered
  overview by combining `strings.Count(...)` with multiple `strings.Contains(...)`
  assertions.
- [`integration_test.go`](/home/yoske/src/github.com/yoskeoka/vibe-coding-workspace/ww/integration_test.go#L1592):
  workspace list output tests use grouped `strings.Contains(...)` checks for
  table-shaped output instead of asserting an expected fragment as one unit.

This style works, but it has a few downsides:

- it spreads one expected output shape across many assertions
- it makes intended formatting harder to read during review
- it encourages partial matching even when the test is really about one
  cohesive multiline contract
- failure messages tell us which fragment was missing, but not what the full
  intended output shape was

## Proposed Solution

Adopt a test-style rule for multiline human-readable output:

- when the contract is a cohesive multiline block, prefer a single raw-string
  expected fragment and compare against that fragment directly
- for outputs that contain dynamic values, normalize or assemble a single
  expected block first, then assert once against that block
- keep multiple `strings.Contains(...)` assertions for truly independent facts,
  not for one logical output layout

Suggested follow-up scope:

1. Convert `ww i --help` assertions to one raw-string expectation.
2. Convert interactive overview rendering assertions to a raw-string or
   normalized expected block.
3. Review other table/help-output tests for the same pattern and migrate the
   ones that are validating one coherent presentation contract.

## Priority

Medium.

This is not a correctness bug, but it directly affects test readability and
review quality in a CLI project where output contracts matter. Left as-is, it
will make future help/output changes noisier to review and easier to assert
partially by accident.
