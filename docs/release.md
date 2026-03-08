# Release process

k8s LogJedi uses [Semantic Versioning](https://semver.org/) and keeps a [CHANGELOG](../CHANGELOG.md). This document describes how to cut a release.

## When to release

- After a set of [ROADMAP](ROADMAP.md) items or bug fixes is merged to `main`.
- When you want to provide a stable tag for users (e.g. Helm `appVersion`, or "install from tag v0.2.0").

## Steps to release

1. **Update CHANGELOG**
   - Under `[Unreleased]`, move the entries that belong to this release into a new section `[X.Y.Z] - YYYY-MM-DD`.
   - Add a link at the bottom: `[X.Y.Z]: https://github.com/adefemi171/k8s.LogJedi/releases/tag/vX.Y.Z` and update the `[Unreleased]` link to compare `vX.Y.Z...HEAD`.
   - Commit: `chore: release vX.Y.Z` (or similar).

2. **Tag the release**
   - Create an annotated tag: `git tag -a vX.Y.Z -m "Release vX.Y.Z"`.
   - Push the tag: `git push origin vX.Y.Z`.

3. **Create a GitHub Release (optional but recommended)**
   - Go to [Releases](https://github.com/adefemi171/k8s.LogJedi/releases) → "Draft a new release".
   - Choose tag `vX.Y.Z`, title "Release vX.Y.Z", and paste the relevant CHANGELOG section into the description.
   - Attach any artifacts (e.g. Helm chart `.tgz`) if you built them locally or via CI.

4. **Update version references (optional)**
   - If you maintain a version in `charts/logjedi/Chart.yaml` (version / appVersion), update it and commit before or after the tag so the chart and the release align.
   - For the LLM service, `pyproject.toml` and Docker labels can be updated in the same release commit.

## CI and automation

- If [GitHub Actions](../.github/workflows/) is configured to run on tag push (e.g. `release.yml`), it can build container images, run tests, and optionally publish images or Helm charts. Adjust the workflow to your registry and publishing process.
- Until automation is in place, building and pushing images (e.g. `docker build`, push to a registry) and publishing the Helm chart are manual steps after tagging.

## Compatibility

- The operator and LLM service communicate via the [API contract](api-contract.md). Adding optional request/response fields is backward compatible. Changing required fields or semantics should coincide with a major or minor version bump and be documented in the CHANGELOG and ROADMAP.
