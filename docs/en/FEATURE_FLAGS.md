# Feature flags and experimental updates

## Goal

Tracker uses feature flags to wrap new behavior without breaking the legacy path. New runtime, API, web, plugin, or rollout logic must be guarded by a flag first, then promoted gradually.

## Mandatory rules

1. Every user-visible or runtime-significant update must have a flag or kill-switch before replacing legacy behavior.
2. The legacy path must remain available until the rollout window closes and rollback is no longer required.
3. Each flag must declare:
   - `key`
   - `description`
   - `default_variant`
   - rollout variants and weights
   - whether it is exposed to the web
4. Long-lived flags must have a cleanup owner and expected removal window.
5. Emergency bug fixes may use a short-lived kill-switch instead of a long AB experiment, but still must be documented.

## Current runtime model

- Config source: `tracker.yaml` / `config.yaml`
- App-level fields:
  - `app.release.channel`
  - `app.release.ring`
  - `app.release.version`
  - `app.release.instance`
  - `app.feature_flags`
- Environment overrides:
  - `TRACKER_RELEASE_CHANNEL`
  - `TRACKER_RELEASE_RING`
  - `TRACKER_RELEASE_VERSION`
  - `TRACKER_RELEASE_INSTANCE`
  - `TRACKER_FLAG_OVERRIDES`

## Built-in baseline flags

- `runtime.executor_v2`
  - Guards the unified execution flow across API, worker, scheduler, and CLI.
  - Variants: `legacy`, `unified`
- `web.runtime_experiments`
  - Controls whether the web shell shows the experiment/release badge.
  - Variants: `off`, `on`
- `runtime.legacy_formats_compat`
  - Temporary compatibility guard for deprecated permissive format matching.

## Request-level overrides

Use one of:

- Header: `X-Tracker-Experiment: flag_a=variant1,flag_b=variant2`
- Query: `?experiment=flag_a=variant1,flag_b=variant2`

Use only for debugging, QA, or approved canary validation. Do not rely on request overrides as the final rollout mechanism.

## Reserved runtime metadata

Every pipeline node may receive a reserved config field:

- `__tracker_runtime`

This contains release and resolved experiment metadata. Existing plugins must ignore unknown fields. New plugins may read it for observability, but should not mutate it.

## Web exposure

Only flags marked `expose_to_web: true` are returned by `GET /api/feature-flags/snapshot`. Runtime-only flags must stay server-side.

## Cleanup policy

After a successful full rollout:

1. Flip the default to the winning variant.
2. Observe one full release window.
3. Remove the losing path.
4. Delete the flag definition.
5. Update docs and SOP references.
