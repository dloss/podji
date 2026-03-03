# Command Bar: Remaining Work

## Current Status

Implemented and in active use:

- `:` command bar overlay
- resource aliases (`po`, `deploy`, `svc`, `cm`, `sec`, `node`, `ing`, `pvc`, `ev`, `ns`)
- name-based jump/list behavior
- subviews (`logs`, `yaml`, `events`, `describe`)
- computed queries (`:unhealthy`, `:restarts`)
- label selector filtering (`key=value[,key=value]`)
- inline suggestions + tab completion + history

## Remaining Enhancements

1. CRD-aware command routing
- Allow command-bar access to CRD resources discovered in the resource browser.

2. Scope shortcuts (optional)
- Add explicit commands for context/namespace switching (for parity with `X`/`N` overlays only if UX win is clear).

3. Query ergonomics
- Consider richer selector operators (`!=`, `in`, `notin`) if complexity is justified.

4. Validation and telemetry
- Add integration coverage for command-bar flows in kube mode.
- Add lightweight debug counters for command parse failures and no-match outcomes.
