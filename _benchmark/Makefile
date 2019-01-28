.PHONY: help
help:  ## show this help
	@grep -E '^[a-zA-Z_\/-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: up/postgres
up/postgres:  ## up postgres docker container for benchmarking
	docker run --rm -d --name postgres \
		-p 5432:5432 \
		-e POSTGRES_USER=testing \
		-e POSTGRES_PASSWORD=testing \
		-e POSTGRES_DB=testing \
		postgres:9.6-alpine

.PHONY: down/postgres
down/postgres:  ## down postgres docker container for benchmarking
	docker stop postgres;
