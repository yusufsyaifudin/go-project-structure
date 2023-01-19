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

build-docker:
	docker build -t myapp .

test:
	go test -v -race -count=1 -coverprofile coverage.cov -cover ./...
	go tool cover -func coverage.cov
	rm coverage.cov
