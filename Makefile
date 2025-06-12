export GO111MODULE=auto
NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m
DEPS = $(go list -f '{{range .TestImports}}{{.}} {{end}}' ./...)
VERSION = $(shell cat core/version.go | grep 'const VERSION' | egrep -o '\d+\.\d+\.\d+')
GO ?= $(shell echo go)

TEST?=$$(go list ./... | grep -v /vendor/)
VETARGS?=-all -race
GOFMT_FILES?=$$(find . -name '*.go' | grep -v vendor)

default: test

all: format deps test

dev-server:
	@$(GO) run *.go -v=0 -alsologtostderr=true --outputs '$(IMG_OUTPUTS)' server

dev-server-s3:
	@$(GO) run *.go --outputs $(IMG_OUTPUTS) --aws_access_key_id $(AWS_ACCESS_KEY_ID) --aws_secret_key $(AWS_SECRET_KEY) --aws_bucket $(AWS_BUCKET) --aws_region $(AWS_REGION) --listen 127.0.0.1 --remote_base_path $(IMG_REMOTE_BASE_PATH) --remote_base_url $(IMG_REMOTE_BASE_URL) server

test:
	go test $(TEST) $(TESTARGS) -timeout=30s -parallel=4

version:
	@echo $(VERSION)

deps:
	@echo "$(OK_COLOR)==> Installing dependencies$(NO_COLOR)"
	@$(GO) get -d -v ./...

dev-deps: deps
	@echo $(DEPS) | xargs -n1 go get -d
	@$(GO) get golang.org/x/tools/cmd/godoc
	@$(GO) get golang.org/x/tools/cmd/vet

update-deps:
	@echo "$(OK_COLOR)==> Updating all dependencies$(NO_COLOR)"
	@$(GO) get -d -v -u ./...
	@echo $(DEPS) | xargs -n1 go get -d -u

clean:
	@rm -rf bin
	@rm -rf tmp
	@rm -rf public

format:
	@gofmt -l -w .

build:
	@mkdir -p bin/
	@rm -f bin/images
	@echo "$(OK_COLOR)==> Building$(NO_COLOR)"
	@$(GO) build -o bin/images-$(VERSION)
	@cd bin && ln -s images-$(VERSION) images
	@echo "$(OK_COLOR)==> Building for solaris amd64$(NO_COLOR)"
	@GOOS=solaris GOARCH=amd64 $(GO) build -o bin/images-solaris-$(VERSION)
	@echo "$(OK_COLOR)==> Building for darwin amd64$(NO_COLOR)"
	@GOOS=darwin GOARCH=amd64 $(GO) build -o bin/images-darwin-$(VERSION)
	@echo "$(OK_COLOR)==> Building for linux amd64$(NO_COLOR)"
	@GOOS=linux GOARCH=amd64 $(GO) build -o bin/images-linux-$(VERSION)

build-docker:
	docker build -t image-server --build-arg SHORT_COMMIT_HASH=$(git rev-parse --short HEAD) .

run-docker:
	docker run -p 7000:7000 -p 7002:7002 image-server

release: test build
  # Mac
	@mput -f bin/darwin/images-$(VERSION) /$(MANTA_USER)/public/images/bin/images-darwin-$(VERSION)
	@echo "$(VERSION)" | mput -H 'content-type: text/plain' /$(MANTA_USER)/public/images/bin/images-darwin-version
  # Linux
	@mput -f bin/linux/images-$(VERSION) /$(MANTA_USER)/public/images/bin/images-linux-$(VERSION)
	@echo "$(VERSION)" | mput -H 'content-type: text/plain' /$(MANTA_USER)/public/images/bin/images-linux-version

fmt:
	gofmt -w $(GOFMT_FILES)
	
.PHONY: all test clean