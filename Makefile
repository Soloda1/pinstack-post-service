.PHONY: test test-unit test-integration test-user-integration clean build run docker-build setup-system-tests

BINARY_NAME=post-service
DOCKER_IMAGE=pinstack-post-service:latest
GO_VERSION=1.24.2
SYSTEM_TESTS_DIR=../pinstack-system-tests
SYSTEM_TESTS_REPO=https://github.com/Soloda1/pinstack-system-tests.git

# Проверка версии Go
check-go-version:
	@echo "🔍 Проверка версии Go..."
	@go version | grep -q "go$(GO_VERSION)" || (echo "❌ Требуется Go $(GO_VERSION)" && exit 1)
	@echo "✅ Go $(GO_VERSION) найден"

# Настройка system tests репозитория
setup-system-tests:
	@echo "🔄 Проверка system tests репозитория..."
	@if [ ! -d "$(SYSTEM_TESTS_DIR)" ]; then \
		echo "📥 Клонирование pinstack-system-tests..."; \
		git clone $(SYSTEM_TESTS_REPO) $(SYSTEM_TESTS_DIR); \
	else \
		echo "🔄 Обновление pinstack-system-tests..."; \
		cd $(SYSTEM_TESTS_DIR) && git pull origin main; \
	fi
	@echo "✅ System tests готовы"

# Форматирование и проверки
fmt: check-go-version
	gofmt -s -w .
	go fmt ./...

lint: check-go-version
	go vet ./...
	golangci-lint run

# Юнит тесты
test-unit: check-go-version
	go test -v -count=1 -race -coverprofile=coverage.txt ./...

# Запуск полной инфраструктуры для интеграционных тестов из существующего docker-compose
start-post-infrastructure: setup-system-tests
	@echo "🚀 Запуск полной инфраструктуры для интеграционных тестов..."
	cd $(SYSTEM_TESTS_DIR) && \
	docker compose -f docker-compose.test.yml up -d \
		user-db-test \
		user-migrator-test \
		user-service-test \
		auth-db-test \
		auth-migrator-test \
		auth-service-test \
		api-gateway-test \
		post-db-test \
		post-migrator-test \
		post-service-test
	@echo "⏳ Ожидание готовности сервисов..."
	@sleep 30

# Проверка готовности сервисов
check-services:
	@echo "🔍 Проверка готовности сервисов..."
	@docker exec pinstack-user-db-test pg_isready -U postgres || (echo "❌ User база данных не готова" && exit 1)
	@docker exec pinstack-auth-db-test pg_isready -U postgres || (echo "❌ Auth база данных не готова" && exit 1)
	@docker exec pinstack-post-db-test pg_isready -U postgres || (echo "❌ Post база данных не готова" && exit 1)
	@echo "✅ Базы данных готовы"
	@echo "=== User Service logs ==="
	@docker logs pinstack-user-service-test --tail=10
	@echo "=== Auth Service logs ==="
	@docker logs pinstack-auth-service-test --tail=10
	@echo "=== Post Service logs ==="
	@docker logs pinstack-post-service-test --tail=10
	@echo "=== API Gateway logs ==="
	@docker logs pinstack-api-gateway-test --tail=10

# Интеграционные тесты только для post service
test-post-integration: start-post-infrastructure check-services
	@echo "🧪 Запуск интеграционных тестов для Post Service..."
	cd $(SYSTEM_TESTS_DIR) && \
	go test -v -count=1 -timeout=10m ./internal/scenarios/integration/gateway_post/...

# Остановка всех контейнеров
stop-post-infrastructure:
	@echo "🛑 Остановка всей инфраструктуры..."
	cd $(SYSTEM_TESTS_DIR) && \
	docker compose -f docker-compose.test.yml stop \
		api-gateway-test \
		auth-service-test \
		auth-migrator-test \
		auth-db-test \
		user-service-test \
		user-migrator-test \
		user-db-test \
		post-service-test \
		post-migrator-test \
		post-db-test
	cd $(SYSTEM_TESTS_DIR) && \
	docker compose -f docker-compose.test.yml rm -f \
		api-gateway-test \
		auth-service-test \
		auth-migrator-test \
		auth-db-test \
		user-service-test \
		user-migrator-test \
		user-db-test \
		post-service-test \
		post-migrator-test \
		post-db-test

# Полная очистка (включая volumes)
clean-post-infrastructure:
	@echo "🧹 Полная очистка всей инфраструктуры..."
	cd $(SYSTEM_TESTS_DIR) && \
	docker compose -f docker-compose.test.yml down -v
	@echo "🧹 Очистка Docker контейнеров, образов и volumes..."
	docker container prune -f
	docker image prune -a -f
	docker volume prune -f
	docker network prune -f
	@echo "✅ Полная очистка завершена"

# Полные интеграционные тесты (с очисткой)
test-integration: test-post-integration stop-post-infrastructure

# Все тесты
test-all: fmt lint test-unit test-integration

# Логи сервисов
logs-user:
	cd $(SYSTEM_TESTS_DIR) && \
	docker compose -f docker-compose.test.yml logs -f user-service-test

logs-auth:
	cd $(SYSTEM_TESTS_DIR) && \
	docker compose -f docker-compose.test.yml logs -f auth-service-test

logs-post:
	cd $(SYSTEM_TESTS_DIR) && \
	docker compose -f docker-compose.test.yml logs -f post-service-test

logs-gateway:
	cd $(SYSTEM_TESTS_DIR) && \
	docker compose -f docker-compose.test.yml logs -f api-gateway-test

logs-db:
	cd $(SYSTEM_TESTS_DIR) && \
	docker compose -f docker-compose.test.yml logs -f user-db-test

logs-auth-db:
	cd $(SYSTEM_TESTS_DIR) && \
	docker compose -f docker-compose.test.yml logs -f auth-db-test

logs-post-db:
	cd $(SYSTEM_TESTS_DIR) && \
	docker compose -f docker-compose.test.yml logs -f post-db-test

# Очистка
clean: clean-post-infrastructure
	go clean
	rm -f $(BINARY_NAME)
	@echo "🧹 Финальная очистка Docker системы..."
	docker system prune -a -f --volumes
	@echo "✅ Вся очистка завершена"

# Экстренная полная очистка Docker (если что-то пошло не так)
clean-docker-force:
	@echo "🚨 ЭКСТРЕННАЯ ПОЛНАЯ ОЧИСТКА DOCKER..."
	@echo "⚠️  Это удалит ВСЕ Docker контейнеры, образы, volumes и сети!"
	@read -p "Продолжить? (y/N): " confirm && [ "$$confirm" = "y" ] || exit 1
	docker stop $$(docker ps -aq) 2>/dev/null || true
	docker rm $$(docker ps -aq) 2>/dev/null || true
	docker rmi $$(docker images -q) 2>/dev/null || true
	docker volume rm $$(docker volume ls -q) 2>/dev/null || true
	docker network rm $$(docker network ls -q) 2>/dev/null || true
	docker system prune -a -f --volumes
	@echo "💥 Экстренная очистка завершена"

# CI локально (имитация GitHub Actions)
ci-local: test-all
	@echo "🎉 Локальный CI завершен успешно!"

# Быстрый тест (только запуск без пересборки)
quick-test: start-post-infrastructure
	@echo "⚡ Быстрый запуск тестов без пересборки..."
	cd $(SYSTEM_TESTS_DIR) && \
	go test -v -count=1 -timeout=5m ./internal/scenarios/integration/gateway_post/...
	$(MAKE) stop-post-infrastructure