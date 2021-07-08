SHELL := /bin/bash # Use bash syntax

# Set up variables
GO111MODULE=on

DOCKER_REPOSITORY ?= amazon/aws-controllers-k8s
DOCKER_USERNAME ?= ""
DOCKER_PASSWORD ?= ""

AWS_SERVICE=$(shell echo $(SERVICE) | tr '[:upper:]' '[:lower:]')

# Build ldflags
VERSION ?= "v0.0.0"
GITCOMMIT=$(shell git rev-parse HEAD)
BUILDDATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GO_LDFLAGS=-ldflags "-X main.version=$(VERSION) \
			-X main.buildHash=$(GITCOMMIT) \
			-X main.buildDate=$(BUILDDATE)"

ACK_CODE_GENERATOR_SOURCE_PATH = "../../aws-controllers-k8s/code-generator"

# We need to use the codegen tag when building and testing because the
# aws-sdk-go/private/model/api package is gated behind a build tag "codegen"...
GO_TAGS=-tags codegen

.PHONY: all build-ack-generate build-controller-image \
	build-controller kind-test delete-all-kind-clusters

all: test

build-ack-generate:	## Build ack-generate binary
	@if [ ! -d "$(ACK_CODE_GENERATOR_SOURCE_PATH)" ]; then \
		echo "Unable to find aws-controllers-k8s/code-generator source repository. Please git clone the aws-controllers-k8s/code-generator repository into your Go src path."; \
	else \
		echo -n "building ack-generate in aws-controllers-k8s/code-generator repo ... "; \
		pushd "$(ACK_CODE_GENERATOR_SOURCE_PATH)" 1>/dev/null; \
		go build $(GO_TAGS) $(GO_LDFLAGS) -o bin/ack-generate cmd/ack-generate/main.go; \
		popd 1>/dev/null; \
		echo "ok."; \
	fi


build-controller-image: export LOCAL_MODULES = false
build-controller-image:	## Build container image for SERVICE
	@./scripts/build-controller-image.sh $(AWS_SERVICE)

local-build-controller-image: export LOCAL_MODULES = true
local-build-controller-image:	## Build container image for SERVICE allowing local modules
	@./scripts/build-controller-image.sh $(AWS_SERVICE)

publish-controller-image:  ## docker push a container image for SERVICE
	@echo $(DOCKER_PASSWORD) | docker login -u $(DOCKER_USERNAME) --password-stdin
	./scripts/publish-controller-image.sh $(AWS_SERVICE)

build-controller: build-ack-generate ## Generate controller code for SERVICE
	@./scripts/install-controller-gen.sh 
	@./scripts/build-controller.sh $(AWS_SERVICE)

kind-test: export PRESERVE = true
kind-test: export LOCAL_MODULES = false
kind-test: ## Run functional tests for SERVICE with ACK_ROLE_ARN
	@./scripts/kind-build-test.sh $(AWS_SERVICE)

local-kind-test: export PRESERVE = true
local-kind-test: export LOCAL_MODULES = true
local-kind-test: ## Run functional tests for SERVICE with ACK_ROLE_ARN allowing local modules
	@./scripts/kind-build-test.sh $(AWS_SERVICE)

delete-all-kind-clusters:	## Delete all local kind clusters
	@kind get clusters | \
	while read name ; do \
	kind delete cluster --name $$name; \
	done
	@rm -rf build/tmp-test*

eks-test: ## Run functional tests for SERVICE using EKS_CLUSTER_NAME
	@./scripts/eks-helm-test.sh $(AWS_SERVICE)

eks-setup-irsa: ## Setup IRSA for SERVICE using EKS_CLUSTER_NAME
	@./scripts/eks-setup-irsa.sh $(AWS_SERVICE)

cleanup-eks-test-tmp-files: ## Delete all temporary file, directories created for EKS test
	@rm -rf build/tmp-eks-test*

help:           ## Show this help.
	@grep -F -h "##" $(MAKEFILE_LIST) | grep -F -v grep | sed -e 's/\\$$//' \
		| awk -F'[:#]' '{print $$1 = sprintf("%-30s", $$1), $$4}'
