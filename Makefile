GOOS ?= $(shell go env GOOS)
GOARCH ?= amd64
BUILD_DIR ?= ./out
ORG = github.com/DevopsArtFactory
PROJECT = goployer
REPOPATH ?= $(ORG)/$(PROJECT)
RELEASE_BUCKET ?= $(PROJECT)
S3_RELEASE_PATH ?= s3://$(RELEASE_BUCKET)/releases/$(VERSION)
S3_RELEASE_LATEST ?= s3://$(RELEASE_BUCKET)/releases/latest

GCP_ONLY ?= false
GCP_PROJECT ?= goployer

SUPPORTED_PLATFORMS = linux-amd64 darwin-amd64 windows-amd64.exe linux-arm64
BUILD_PACKAGE = $(REPOPATH)

GOPLOYER_TEST_PACKAGES = ./pkg/goployer/... ./cmd/... ./hack/... ./pkg/webhook/...
GO_FILES = $(shell find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./pkg/diag/*")

VERSION_PACKAGE = $(REPOPATH)/pkg/goployer/version
COMMIT = $(shell git rev-parse HEAD)

ifeq "$(strip $(VERSION))" ""
 override VERSION = $(shell git describe --always --tags --dirty)
endif

LDFLAGS_linux = -static
LDFLAGS_darwin =
LDFLAGS_windows =

GO_BUILD_TAGS_linux = "osusergo netgo static_build release"
GO_BUILD_TAGS_darwin = "release"
GO_BUILD_TAGS_windows = "release"

GO_LDFLAGS = -X $(VERSION_PACKAGE).version=$(VERSION)
GO_LDFLAGS += -X $(VERSION_PACKAGE).buildDate=$(shell date +'%Y-%m-%dT%H:%M:%SZ')
GO_LDFLAGS += -X $(VERSION_PACKAGE).gitCommit=$(COMMIT)
GO_LDFLAGS += -X $(VERSION_PACKAGE).gitTreeState=$(if $(shell git status --porcelain),dirty,clean)
GO_LDFLAGS += -s -w

GO_LDFLAGS_windows =" $(GO_LDFLAGS)  -extldflags \"$(LDFLAGS_windows)\""
GO_LDFLAGS_darwin =" $(GO_LDFLAGS)  -extldflags \"$(LDFLAGS_darwin)\""
GO_LDFLAGS_linux =" $(GO_LDFLAGS)  -extldflags \"$(LDFLAGS_linux)\""

# Build for local development.
$(BUILD_DIR)/$(PROJECT): $(GO_FILES) $(BUILD_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=1 go build -tags $(GO_BUILD_TAGS_$(GOOS)) -ldflags $(GO_LDFLAGS_$(GOOS)) -o $@ $(BUILD_PACKAGE)

.PHONY: install
install: $(BUILD_DIR)/$(PROJECT)
	cp $(BUILD_DIR)/$(PROJECT) $(GOPATH)/bin/$(PROJECT)

.PRECIOUS: $(foreach platform, $(SUPPORTED_PLATFORMS), $(BUILD_DIR)/$(PROJECT)-$(platform))

.PHONY: cross
cross: $(foreach platform, $(SUPPORTED_PLATFORMS), $(BUILD_DIR)/$(PROJECT)-$(platform))

$(BUILD_DIR)/$(PROJECT)-%: $(STATIK_FILES) $(GO_FILES) $(BUILD_DIR) deploy/cross/Dockerfile
	$(eval os = $(firstword $(subst -, ,$*)))
	$(eval arch = $(lastword $(subst -, ,$(subst .exe,,$*))))
	$(eval ldflags = $(GO_LDFLAGS_$(os)))
	$(eval tags = $(GO_BUILD_TAGS_$(os)))

	docker build \
		--build-arg GOOS=$(os) \
		--build-arg GOARCH=$(arch) \
		--build-arg TAGS=$(tags) \
		--build-arg LDFLAGS=$(ldflags) \
		-f deploy/cross/Dockerfile \
		-t goployer/cross \
		.

	docker run --rm goployer/cross cat /build/goployer > $@
	shasum -a 256 $@ | tee $@.sha256
	file $@ || true

.PHONY: $(BUILD_DIR)/VERSION
$(BUILD_DIR)/VERSION: $(BUILD_DIR)
	@ echo $(VERSION) > $@

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

.PHONY: test
test: $(BUILD_DIR)
	@ ./hack/gotest.sh -count=1 -race -short -timeout=90s $(GOPLOYER_TEST_PACKAGES)
	@ ./hack/checks.sh
	@ ./hack/linters.sh

.PHONY: release
release: cross $(BUILD_DIR)/VERSION upload-only

.PHONY: release-build
release-build: cross
	docker build \
		-f deploy/goployer/Dockerfile \
		--target release \
		-t gcr.io/$(GCP_PROJECT)/goployer:edge \
		-t gcr.io/$(GCP_PROJECT)/goployer:$(COMMIT) \
		.
	aws s3 cp $(BUILD_DIR)/$(PROJECT)-* $(S3_RELEASE_PATH)/
	aws s3 cp -r $(S3_RELEASE_PATH)/* $(S3_RELEASE_LATEST)

.PHONY: upload-only
upload-only:
	aws s3 cp $(BUILD_DIR)/ $(S3_RELEASE_PATH)/ --recursive --include "$(PROJECT)-*"
	aws s3 cp $(BUILD_DIR)/VERSION $(S3_RELEASE_PATH)/VERSION
	aws s3 cp $(S3_RELEASE_PATH)/* $(S3_RELEASE_LATEST) --recursive

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

# utilities for goployer site - not used anywhere else
.PHONY: preview-docs
preview-docs:
	./deploy/docs/local-preview.sh hugo serve -D --bind=0.0.0.0 --ignoreCache

.PHONY: build-docs-preview
build-docs-preview:
	./deploy/docs/local-preview.sh hugo --baseURL=https://goployer.dev
