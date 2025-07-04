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
build:
	@go build -o generator .
	./generator

.PHONY: hash
## hash: update static files hashes in index.html
hash:
	sed -i '' -E "s/files\.png\?crc=[0-9a-z]+/files.png?crc=$$(crc32 ./static/files.png)/" static/style.css
	sed -i '' -E "s/sprite\.png\?crc=[0-9a-z]+/sprite.png?crc=$$(crc32 ./static/sprite.png)/" static/style.css

.PHONY: serve
## serve: serve the static site
serve: hash build
	@wrangler pages dev --local-protocol=https output/ --compatibility-date=2024-02-25 --binding GHP_TOKEN=${GHP_TOKEN} --binding GITHUB_REPO=${GITHUB_REPO} --port=${WRANGLER_PORT}

.PHONY: codegen
## codegen: generate code from the schema
codegen:
	@cd codegen && go build -o codegen . && ./codegen -in ${INPUT_INFO}/_finder/schema.yml -out ../structs/content.gen.go
	@go fmt ./structs/content.gen.go

.PHONY: caddy
## caddy: run caddy server
caddy:
	@caddy run
