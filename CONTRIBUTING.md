# Contributing to k8s LogJedi

Thank you for your interest in contributing. This document explains how to get started, run tests, and submit changes.

## Code of conduct

Be respectful and constructive. We aim to keep the project welcoming and focused on the roadmap.

## How to contribute

1. **Open an issue** – Describe the change and how it fits the [ROADMAP](ROADMAP.md). For bugs, include steps to reproduce and environment (k8s version, provider).
2. **Discuss** – For larger changes, discuss in the issue before a big PR.
3. **Fork and branch** – Create a branch from `main` (e.g. `feature/your-feature` or `fix/issue-123`). We use a simple model: `main` is the default branch; feature/fix branches merge via PR.
4. **Make changes** – Keep PRs small and reviewable (one logical change per PR). Update README, ROADMAP, or ADRs when adding config or behavior.
5. **Run tests and lint** – See below. CI will run on your PR.
6. **Submit a PR** – Use the [pull request template](.github/PULL_REQUEST_TEMPLATE.md). Link the related issue. Ensure CI passes.

## Development setup

- **Operator:** Go 1.21+; from repo root: `make build-operator` or `cd operator && go build -o bin/operator .`
- **LLM service:** Python 3.14+ and [uv](https://docs.astral.sh/uv/); from repo root: `make build-llm` or `cd llm-service && uv sync`
- **Full build:** `make build`
- **Docker:** `make docker-build` (builds `logjedi-operator:latest` and `logjedi-llm-service:latest`)

See [README.md](README.md) for cluster setup (kind/minikube), deploy, and testing the full flow.

## Running tests

- **Operator:** `make test-operator` or `cd operator && go vet ./... && go test ./...`
- **LLM service:** `make test-llm` or `cd llm-service && uv run pytest -q`
- **All:** `make test`

CI runs these on every push and PR. Please run tests locally before submitting.

## Code style and lint

- **Go:** We use `go vet` and `go test`. Format with `go fmt ./...`. Optionally use [golangci-lint](https://golangci-lint.run/) with the repo’s `.golangci.yml`.
- **Python:** We use [ruff](https://docs.astral.sh/ruff/) for lint/format. Config is in `llm-service/pyproject.toml`. Run: `cd llm-service && uv run ruff check . && uv run ruff format --check .`
- **Pre-commit (optional):** Install [pre-commit](https://pre-commit.com/) and run `pre-commit install`. Hooks will run ruff and go fmt on commit.

## Release process

Releases are tagged (e.g. `v0.2.0`). See [docs/release.md](docs/release.md) for how to cut a release and update CHANGELOG. Compatibility and versioning are described in [ROADMAP.md](ROADMAP.md#versioning-and-releases).

## Questions

Open a [GitHub Discussion](https://github.com/adefemi171/k8s.LogJedi/discussions) or an issue if something is unclear.
