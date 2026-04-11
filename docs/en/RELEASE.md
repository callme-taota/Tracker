# Release Guide

## Workflow Layers

GitHub Actions are split into three layers:

- `CI`: runs Go tests, the web build, and a Docker smoke build on pull requests and pushes to `master`.
- `Release`: creates platform archives, `checksums.txt`, and a GitHub Release when a `vX.Y.Z` tag is pushed.
- `Docker`: builds and publishes GHCR images on `master` pushes and version tags.

## Release Artifact Contract

Each release tag produces self-contained archives with a consistent layout:

```text
tracker_<version>_<os>_<arch>/
  tracker[.exe]
  web/dist/
  configs/tracker.example.yaml
  configs/pipeline.example.yaml
  .env.example
  web/.env.example
  README.md
  README.zh.md
```

After extraction, `tracker serve` can start the complete web UI without requiring a separate frontend build.

## How To Release

1. Make sure CI is green on `master`.
2. Create and push a semantic version tag, for example:

```bash
git tag v0.1.0
git push origin v0.1.0
```

3. Wait for the `Release` workflow to finish, then download the matching archive from GitHub Releases.

## Docker Tag Strategy

- `master` pushes publish `ghcr.io/<owner>/tracker:main` and `ghcr.io/<owner>/tracker:sha-<shortsha>`
- `vX.Y.Z` tags publish `ghcr.io/<owner>/tracker:vX.Y.Z` and `ghcr.io/<owner>/tracker:latest`

## Security Constraints

- Release and public Docker builds never inject real runtime secrets.
- Runtime secrets such as `TRACKER_API_KEY` and `OPENAI_API_KEY` should be configured at deploy time, not baked into artifacts.
- `VITE_TRACKER_API_KEY` remains an optional build argument, but the default workflows do not provide a production value.
