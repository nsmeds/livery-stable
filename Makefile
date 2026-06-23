.PHONY: \
	build \
	deploy \
	test \
	compose-up \
	compose-down

COVERAGE_MIN=50

build:
	docker buildx build --platform=linux/amd64 \
		-f Dockerfile \
		-t livery-stable \
		.

lint:
	staticcheck ./...

compose-up:
	docker-compose up -d
	@echo "Waiting for postgres..."
	@until docker-compose exec -T postgres pg_isready -U postgres > /dev/null 2>&1; do sleep 1; done
	@until docker-compose exec -T postgres_test pg_isready -U postgres > /dev/null 2>&1; do sleep 1; done
	@echo "Postgres ready."

compose-down:
	docker-compose down

test: compose-up
	go test -timeout 30s -cover -coverprofile=coverage.out ./...
	@go tool cover -func coverage.out | \
		perl -an -E 'die "$$F[2] coverage does not meet threshold of ${COVERAGE_MIN}%\n" if /total/ && $$F[2] < ${COVERAGE_MIN}'
