# AB rollout and release process

## Scope

Tracker now uses two rollout layers:

1. Codepath AB inside one build
2. Release / deployment rollout across environments or instance rings

Both layers must use the same flag keys and the same rollback language.

## Standard rollout flow

1. Define the flag and legacy path.
2. Keep the old path as the default variant.
3. Validate locally with request overrides.
4. Enable canary in one ring or for a small subject set.
5. Observe runtime health, job success, pipeline output, and web/API regressions.
6. Increase weight gradually.
7. Flip the default only after the new path is stable.
8. Remove legacy code and the flag after the cleanup window.

## Release metadata

Use:

- `release.channel`: `stable`, `staging`, `canary`, or another approved channel
- `release.ring`: logical rollout ring such as `global`, `ring-1`, `ring-2`
- `release.version`: release label for audit and debugging
- `release.instance`: stable process identity

## Rollback

Rollback options, in order:

1. Force the old variant with `TRACKER_FLAG_OVERRIDES`
2. Reduce rollout weight to zero
3. Move traffic back to the stable ring
4. Redeploy the previous build if needed

## Do not do this

- Do not delete the old path before one full observation window ends.
- Do not expose runtime-only flags to the web.
- Do not run a build-level rollout without a documented rollback command.
- Do not reuse old flag keys for unrelated features.
