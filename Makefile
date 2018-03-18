SOURCES = $(shell find . -name '*.go')

.PHONY: .coverprofile

default: build

build: $(SOURCES)
	go build

check: build
	go test

.coverprofile:
	go test -coverprofile .coverprofile

cover: .coverprofile
	go tool cover -func .coverprofile

publishcoverage: .coverprofile
	curl -s https://codecov.io/bash -o codecov
	bash codecov -Zf .coverprofile

showcover: .coverprofile
	go tool cover -html .coverprofile

fmt:
	gofmt -s -w $(SOURCES)

checkfmt: $(SOURCES)
	@echo check fmt
	@if [ "$$(gofmt -s -d $(SOURCES))" != "" ]; then false; else true; fi

ci-trigger: checkfmt build check
ifeq ($(TRAVIS_BRANCH)_$(TRAVIS_PULL_REQUEST), master_false)
	make publishcoverage
endif

clean:
	go clean -i -cache
