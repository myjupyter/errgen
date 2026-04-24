BINPATH     := $(PWD)/bin
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
export PATH := $(PATH):$(BINPATH)

# build the binary
.PHONY: build
build:
	@go build -ldflags "-X main.version=$(VERSION)" -o ./bin/errgen .

.PHONY: gen
gen:
	@go generate $(PWD)/...


.PHONY: test
test: build
	@echo "Generating test cases..."
	@go generate $(PWD)/test/...
	@echo "Building test packages..."
	@go build $(PWD)/test/...
	@echo "All test cases passed without errors"

.PHONY: clean
clean:
	@rm -rf $(BINPATH)