GIT_COMMIT:=$(shell git rev-list -1 HEAD)
CURRENT_TIME:=$(shell date +"%s")

build:
	$(MAKE) build-server

build-server:
	cat /dev/null > "assets/build_commit_hash.txt"
	git rev-list -1 HEAD >> "assets/build_commit_hash.txt"

	#cat /dev/null > "assets/build_time.txt"
	#BUILD_TIME=$(date +"%s") && echo "$BUILD_TIME" >> "assets/build_time.txt"

	env GOOS=linux GOARCH=amd64 go build -o server-linux cmd/server/main.go
	env GOOS=darwin GOARCH=amd64 go build -o server-darwin cmd/server/main.go

	cat /dev/null > "assets/build_commit_hash.txt"
	cat /dev/null > "assets/build_time.txt"
	printf "0" > "assets/build_time.txt"

IMAGE_NAME?=ghcr.io/yusufsyaifudin/go-project-structure
IMAGE_TAG?=local

build-docker: build-server-image build-cli-image

build-server-image:
	docker build --target server -t $(IMAGE_NAME):$(IMAGE_TAG)-server .

build-cli-image:
	docker build --target cli -t $(IMAGE_NAME):$(IMAGE_TAG)-cli .

test:
	go test -v -race -count=1 \
		-coverprofile coverage.cov \
		-coverpkg=./internal/gateway/...,./internal/pkg/...,./internal/service/...,./pkg/...,./transport/... \
		./...
	go tool cover -func coverage.cov
	rm coverage.cov

lint:
	golangci-lint run --print-resources-usage --verbose --timeout=30m ./...

# Generate swagger docs from swaggo annotations.
# Must be run before building when annotations change.
swagger:
	swag init --parseDependency --parseInternal -g transport/restapi/doc.go -o docs/swagger
	@# Fix swag-generated comment that godot flags (missing period).
	@sed -i'' -e 's|// SwaggerInfo holds exported Swagger Info so clients can modify it$$|// SwaggerInfo holds exported Swagger Info so clients can modify it.|' docs/swagger/docs.go

# Create a new SQL migration file.
# Usage: make migration-new name=create_something
migration-new:
	@if [ -z "$(name)" ]; then echo "Usage: make migration-new name=<description>"; exit 1; fi
	@FILENAME="migrations/$$(date +'%Y-%m-%d-%H-%M-%S')_$(name).sql"; \
	printf '-- +migrate Up\n-- SQL in section '\''Up'\'' is executed when this migration is applied\n\n\n-- +migrate Down\n-- SQL in section '\''Down'\'' is executed when this migration is rolled back\n' > "$$FILENAME"; \
	echo "Created $$FILENAME"
