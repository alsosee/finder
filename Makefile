.PHONY: help
## help: prints this help message
help:
	@echo "Usage: \n"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

.PHONY: test
## test: run tests
test:
	@go test -cover ./...

.PHONY: lint
## lint: run golangci-lint
# Install: https://golangci-lint.run/usage/install/
lint:
	@golangci-lint run ./... --out-format colored-line-number

.PHONY: clean
## clean: clean the output
clean:
	@rm -rf ./output

.PHONY: build
## build: build the static site
build: lint
	@go run .
