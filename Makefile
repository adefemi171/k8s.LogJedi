# LogSage mono-repo
# operator: Go controller
# llm-service: Python FastAPI + Strands

.PHONY: build build-operator build-llm docker-build docker-build-operator docker-build-llm deploy deploy-operator deploy-llm test test-operator test-llm dev

build: build-operator build-llm

build-operator:
	cd operator && go build -o bin/operator .

build-llm:
	cd llm-service && uv sync

docker-build: docker-build-operator docker-build-llm

docker-build-operator:
	docker build -t logsage-operator:latest ./operator

docker-build-llm:
	docker build -t logsage-llm-service:latest ./llm-service

deploy: deploy-operator deploy-llm

deploy-operator:
	kubectl apply -f operator/config/deploy/

deploy-llm:
	kubectl apply -f llm-service/deploy/

# Run unit tests (operator: go test; llm-service: pytest if available)
test: test-operator test-llm

test-operator:
	cd operator && go vet ./... && go test ./...

test-llm:
	cd llm-service && (uv run pytest -q 2>/dev/null || true)

# One-command dev: ensure cluster exists, build images, deploy, optionally load images (kind) and create sample failing workload.
# Requires: kind or minikube, kubectl, make. For kind: run "kind load docker-image logsage-operator:latest logsage-llm-service:latest --name kind" after first docker-build.
dev: build docker-build
	@echo "Cluster: ensure a cluster is running (e.g. kind create cluster, or minikube start)."
	@echo "Then: make deploy"
	@echo "Optional: kubectl apply -f operator/config/samples/failing-deployment.yaml"
	@echo "For kind: kind load docker-image logsage-operator:latest logsage-llm-service:latest"
