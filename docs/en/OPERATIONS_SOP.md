# Operations SOP

## Purpose

This document makes feature-flagged updates operationally repeatable with low friction.

## SOP template

### 1. Proposal

- Define the change
- Define the legacy path
- Define the flag key and variants
- Define success metrics and rollback triggers

### 2. Implementation

- Add the new codepath behind a flag
- Keep the legacy codepath intact
- Add tests for both control and new-path evaluation if practical
- Update `FEATURE_FLAGS.md`

### 3. Verification

- Validate locally with request override
- Validate in one non-production channel or ring
- Confirm web snapshot visibility only for web-exposed flags

### 4. Rollout

- Start with weight 0 for the new path
- Increase weight in controlled steps
- Observe logs, job outcomes, API errors, and user-facing regressions

### 5. Rollback

- Apply `TRACKER_FLAG_OVERRIDES`
- Reduce rollout weight
- Redeploy previous build if needed
- Record the incident and winning mitigation

### 6. Cleanup

- Flip the default to the winning variant
- Remove the losing path after the observation window
- Delete the flag and related docs entries

## Minimal checklist

- Flag exists
- Legacy path preserved
- Rollback documented
- Metrics identified
- Cleanup owner assigned
