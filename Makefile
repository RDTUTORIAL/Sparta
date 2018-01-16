.DEFAULT_GOAL=build

GO_LINT := $(GOPATH)/bin/golint

################################################################################
# Meta
################################################################################
reset:
		git reset --hard
		git clean -f -d

################################################################################
# Code generation
################################################################################
generate:
	./generate-constants.sh
	@echo "Generate complete: `date`"

################################################################################
# Hygiene checks
################################################################################

GO_SOURCE_FILES := find . -type f -name '*.go' \
	! -path './vendor/*' \

.PHONY: install_requirements
install_requirements:
	go get -u honnef.co/go/tools/cmd/megacheck
	go get -u honnef.co/go/tools/cmd/gosimple
	go get -u honnef.co/go/tools/cmd/unused
	go get -u honnef.co/go/tools/cmd/staticcheck
	go get -u golang.org/x/tools/cmd/goimports
	go get -u github.com/fzipp/gocyclo
	go get -u github.com/golang/lint/golint
	go get -u github.com/mjibson/esc

.PHONY: vet
vet: install_requirements
	for file in $(shell $(GO_SOURCE_FILES)); do \
		go tool vet "$${file}" || exit 1 ;\
	done

.PHONY: lint
lint: install_requirements
	for file in $(shell $(GO_SOURCE_FILES)); do \
		$(GO_LINT) "$${file}" || exit 1 ;\
	done

.PHONY: fmt
fmt: install_requirements
	$(GO_SOURCE_FILES) -exec goimports -w {} \;

.PHONY: fmtcheck
fmtcheck:install_requirements
	@ export output="$$($(GO_SOURCE_FILES) -exec goimports -d {} \;)"; \
		test -z "$${output}" || (echo "$${output}" && exit 1)

.PHONY: validate
validate: install_requirements vet lint fmtcheck
	megacheck -ignore github.com/mweagle/Sparta/CONSTANTS.go:*

docs:
	@echo ""
	@echo "Sparta godocs: http://localhost:8090/pkg/github.com/mweagle/Sparta"
	@echo
	godoc -v -http=:8090 -index=true

################################################################################
# Travis
################################################################################
travis-depends: install_requirements
	go get -u github.com/golang/dep/...
	dep ensure
	# Move everything in the ./vendor directory to the $(GOPATH)/src directory
	rsync -a --quiet --remove-source-files ./vendor/ $(GOPATH)/src


.PHONY: travis-ci-test
travis-ci-test: travis-depends test build
	go test -v -cover ./...

################################################################################
# Sparta commands
################################################################################
provision: build
	go run ./applications/hello_world.go --level info provision --s3Bucket $(S3_BUCKET)

execute: build
	./sparta execute

describe: build
	rm -rf ./graph.html
	go test -v -run TestDescribe

################################################################################
# ALM commands
################################################################################
.PHONY: clean
clean:
	go clean .
	go env

.PHONY: test
test: validate
	go test -v -cover ./...

.PHONY: build
build: validate test
	go build .
	@echo "Build complete"

.PHONY: publish
publish: generate
	$(info Checking Git tree status)
	git diff --exit-code
	./buildinfo.sh
	git commit -a -m "Tagging Sparta commit"
	git push origin