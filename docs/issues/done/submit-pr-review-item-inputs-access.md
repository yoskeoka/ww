# submit_pr_review script accesses item.inputs but agent output has flat structure

**Source**: CI job failure on PR #27 ([job log](https://github.com/yoskeoka/ww/actions/runs/23166860680/job/67309365264))
**Type**: bug | **Priority**: High
**Affects**: All three gh-aw review workflows (plan-review, impl-review, spec-code-sync)

## Problem

The `submit_pr_review` custom safe output job script crashes with:

```
TypeError: Cannot read properties of undefined (reading 'event')
```

The script accesses `item.inputs.event` and `item.inputs.body`, but the gh-aw agent output items likely use a flat structure (`item.event`, `item.body`) without an `inputs` wrapper.

### Failing code (all 3 workflows)

```javascript
const items = output.items.filter(i => i.type === 'submit_pr_review');
for (const item of items) {
  await github.rest.pulls.createReview({
    ...
    event: item.inputs.event,  // TypeError: item.inputs is undefined
    body: item.inputs.body
  });
}
```

## Fix

Inspect the actual `agent_output.json` structure from a workflow run to determine the correct field paths. Likely one of:

- `item.event` / `item.body` (flat)
- `item.data.event` / `item.data.body`

Update the script in all 3 workflow `.md` files and recompile with `gh aw compile --strict`.

Add a guard for empty items to avoid silent no-ops:

```javascript
if (items.length === 0) {
  core.warning('No submit_pr_review items found in agent output');
}
```
