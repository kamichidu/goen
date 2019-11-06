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
