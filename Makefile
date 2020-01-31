LDFLAGS=

.PHONY: debug-template
debug-template:
	go generate ./generator/
	go run \
		${LDFLAGS} \
		./cmd/goen/*.go \
		-no-gofmt \
		-no-goimports \
		-o ./example/goen.go \
		./example/

.PHONY: debug
debug: build
	./goen ${GOENARGS}
	go install ./entity/
	go run debug.go ${RUNARGS}

.PHONY: build
build:
	go build ${LDFLAGS} .

.PHONY: generate
generate:
	go generate ./...

.PHONY: lint
lint:
	./_tools/golangci-lint-run.sh

.PHONY: deps
deps:
	cp go.mod go.mod.deps; cp go.sum go.sum.deps
	go get -v github.com/mjibson/esc
	mv go.mod.deps go.mod; mv go.sum.deps go.sum
