SHELL := /usr/bin/env bash

# Defaults
NAMESPACE ?= hc4ai-operator
CHART ?= charts/llm-d

MS_VERSION      ?= v0.0.15
EPP_VERSION     ?= v0.1.0
VLLM_VERSION    ?= 0.0.8
ROUTING_PROXY_VERSION ?= 0.0.7
INFERENCE_SIM_VERSION ?= 0.0.4

.PHONY: help
help: ## Print help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: pre-helm
pre-helm:
	helm repo add bitnami https://charts.bitnami.com/bitnami

.PHONY: helm-lint
helm-lint: pre-helm ## Run helm lint on the specified chart
	@printf "\033[33;1m==== Running helm lint ====\033[0m\n"
	ct lint

.PHONY: helm-template
helm-template: pre-helm ## Render chart templates without installing
	@printf "\033[33;1m==== Running helm template ====\033[0m\n"
	helm template $(RELEASE) $(CHART) --namespace $(NAMESPACE)

.PHONY: helm-install
helm-install: pre-helm ## Install the chart into the given namespace
	@printf "\033[33;1m==== Running helm install ====\033[0m\n"
	helm install $(RELEASE) $(CHART) --namespace $(NAMESPACE) --create-namespace

.PHONY: helm-upgrade
helm-upgrade: pre-helm ## Upgrade the release if it exists
	@printf "\033[33;1m==== Running helm upgrade ====\033[0m\n"
	helm upgrade --install $(RELEASE) $(CHART) --namespace $(NAMESPACE) --create-namespace

.PHONY: helm-uninstall
helm-uninstall: ## Uninstall the Helm release
	@printf "\033[33;1m==== Running helm uninstall ====\033[0m\n"
	helm uninstall $(RELEASE) --namespace $(NAMESPACE)


##@ Automation

.Phony: bump-modelservice-crd
bump-modelservice-crd:
	git clone git@github.com:llm-d/llm-d-model-service.git -b $(MS_VERSION) --depth=1
	kustomize build llm-d-model-service/config/crd > charts/llm-d/crds/modelservice-crd.yaml
	rm -rf llm-d-model-service

# Setting SED allows macos users to install GNU sed and use the latter
# instead of the default BSD sed.
ifeq ($(shell command -v gsed 2>/dev/null),)
    SED ?= $(shell command -v sed)
else
    SED ?= $(shell command -v gsed)
endif
ifeq ($(shell ${SED} --version 2>&1 | grep -q GNU; echo $$?),1)
    $(error !!! GNU sed is required. If on OS X, use 'brew install gnu-sed'.)
endif

VALUES_FILE := charts/llm-d/values.yaml

.Phony: bump-image-tags
bump-image-tags:
	@echo "Updating image tags in $(VALUES_FILE)..."
	# Update modelservice.image.tag
	$(SED) -i '/^modelservice:/,/^[a-zA-Z]/ { /^  image:/,/^  [a-zA-Z]/ { s/^\(    tag: \).*$$/\1"$(MS_VERSION)"/; } }' $(VALUES_FILE)
	# Update modelservice.epp.image.tag
	$(SED) -i '/^modelservice:/,/^[a-zA-Z]/ { /^  epp:/,/^  [a-zA-Z]/ { /^    image:/,/^    [a-zA-Z]/ { s/^\(      tag: \).*$$/\1"$(EPP_VERSION)"/; } } }' $(VALUES_FILE)
	# Update modelservice.vllm.image.tag
	$(SED) -i '/^modelservice:/,/^[a-zA-Z]/ { /^  vllm:/,/^  [a-zA-Z]/ { /^    image:/,/^    [a-zA-Z]/ { s/^\(      tag: \).*$$/\1"$(VLLM_VERSION)"/; } } }' $(VALUES_FILE)
	# Update modelservice.routingProxy.image.tag
	$(SED) -i '/^modelservice:/,/^[a-zA-Z]/ { /^  routingProxy:/,/^  [a-zA-Z]/ { /^    image:/,/^    [a-zA-Z]/ { s/^\(      tag: \).*$$/\1"$(ROUTING_PROXY_VERSION)"/; } } }' $(VALUES_FILE)
	# Update modelservice.inferenceSimulator.image.tag
	$(SED) -i '/^modelservice:/,/^[a-zA-Z]/ { /^  inferenceSimulator:/,/^  [a-zA-Z]/ { /^    image:/,/^    [a-zA-Z]/ { s/^\(      tag: \).*$$/\1"$(INFERENCE_SIM_VERSION)"/; } } }' $(VALUES_FILE)
	@echo "Image tags updated successfully!"

.PHONY: bump-chart-version
# Bump Helm chart version, usage: make bump-chart-version bump_type=[patch|minor|major]
bump-chart-version:
	helpers/scripts/increment-chart-version.sh $(bump_type)
