.PHONY: \
	build \
	deploy \
	test

COVERAGE_MIN=50

build:
	docker buildx build --platform=linux/amd64 \
		-f Dockerfile \
		-t livery-stable \
		.

lint:
	staticcheck ./...

test: 
	go test -timeout 10s -cover -coverprofile=coverage.out ./...
	@go tool cover -func coverage.out | \
		perl -an -E 'die "$$F[2] coverage does not meet threshold of ${COVERAGE_MIN}%\n" if /total/ && $$F[2] < ${COVERAGE_MIN}'