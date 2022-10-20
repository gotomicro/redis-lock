.PHONY:	bench
bench:
	@go test -bench=. -benchmem  ./...

.PHONY:	ut
ut:
	@go test -race ./...

.PHONY:	setup
setup:
	@sh ./script/setup.sh

.PHONY:	fmt
fmt:
	@sh ./script/goimports.sh

.PHONY:	lint
lint:
	@golangci-lint run -c .golangci.yml

.PHONY: tidy
tidy:
	@go mod tidy -v

.PHONY: check
check:
	@$(MAKE) fmt
	@$(MAKE) tidy

# e2e 测试
.PHONY: e2e
e2e:
	sh ./script/integrate_test.sh

.PHONY: e2e_up
e2e_up:
	docker compose -f script/docker-compose.yml up -d

.PHONY: e2e_down
e2e_down:
	docker compose -f script/docker-compose.yml down

.PHONY: mock
mock:
	mockgen -copyright_file=.license.go.header -package=mocks -destination=mocks/redis_cmdable.mock.go github.com/go-redis/redis/v9 Cmdable