# Pinstack Post Service 📝

**Pinstack Post Service** — микросервис для управления постами в системе **Pinstack**.

## Основные функции:
- CRUD-операции для постов (создание, чтение, обновление, удаление).
- Хранение данных постов и медиафайлов.
- Взаимодействие с другими микросервисами через gRPC.
- **Redis кеширование** для оптимизации производительности.

## Технологии:
- **Go** — основной язык разработки.
- **gRPC** — для межсервисной коммуникации.
- **Docker** — для контейнеризации.
- **Prometheus** — для сбора метрик и мониторинга.
- **Grafana** — для визуализации метрик.
- **Loki** — для централизованного сбора логов.
- **PostgreSQL** — основная база данных.
- **Redis** — кеширование данных.

## Архитектура

Проект построен на основе **гексагональной архитектуры (Hexagonal Architecture)** с четким разделением слоев:

### Структура проекта
```
├── cmd/                    # Точки входа приложения
│   ├── server/             # gRPC сервер
│   └── migrate/            # Миграции БД
├── internal/
│   ├── domain/             # Доменный слой
│   │   ├── models/         # Доменные модели
│   │   └── ports/          # Интерфейсы (порты)
│   │       ├── input/      # Входящие порты (use cases)
│   │       └── output/     # Исходящие порты (репозитории, кэш, метрики)
│   ├── application/        # Слой приложения
│   │   └── service/        # Бизнес-логика и сервисы
│   └── infrastructure/     # Инфраструктурный слой
│       ├── inbound/        # Входящие адаптеры (gRPC, HTTP)
│       └── outbound/       # Исходящие адаптеры (PostgreSQL, Redis, Prometheus)
├── migrations/             # SQL миграции
└── mocks/                 # Моки для тестирования
```

### Принципы архитектуры
- **Dependency Inversion**: Зависимости направлены к доменному слою
- **Clean Architecture**: Четкое разделение ответственности между слоями
- **Port & Adapter Pattern**: Интерфейсы определяются в domain, реализуются в infrastructure
- **Testability**: Легкое модульное тестирование благодаря dependency injection

### Архитектура кеширования 🗄️

#### Стратегия кеширования:
- **User Cache** — кеширование пользователей (15 мин TTL) для решения N+1 запросов
- **Post Cache** — кеширование детальной информации о постах (30 мин TTL)

#### Реализация:
- **Decorator Pattern** — `PostServiceCacheDecorator` оборачивает основной сервис
- **Hexagonal Architecture** — кеш интегрирован через порты и адаптеры
- **Smart Invalidation** — автоматическая инвалидация при обновлении/удалении
- **Error Resilience** — при ошибках кеша обращается к основному источнику данных

#### Оптимизируемые операции:
1. **Список постов** — кеширование пользователей для избежания N+1 запросов
2. **Детали поста** — полное кеширование с вложенными объектами

### Мониторинг и метрики
Сервис включает полную интеграцию с системой мониторинга:
- **Prometheus метрики**: Автоматический сбор метрик gRPC, базы данных, кэша
- **Structured logging**: Интеграция с Loki для централизованного сбора логов
- **Health checks**: Проверки состояния всех компонентов
- **Performance monitoring**: Метрики времени ответа и throughput

## CI/CD Pipeline 🚀

### GitHub Actions
Проект использует GitHub Actions для автоматического тестирования при каждом push/PR.

**Этапы CI:**
1. **Unit Tests** — юнит-тесты с покрытием кода
2. **Integration Tests** — интеграционные тесты с полной инфраструктурой 
3. **Auto Cleanup** — автоматическая очистка Docker ресурсов

### Makefile команды 📋

#### Команды разработки

### Настройка и запуск
```bash
# Запуск легкой среды разработки (только Prometheus stack)
make start-dev-light

# Запуск полной среды разработки (с мониторингом)
make start-dev-full

# Остановка среды разработки
make stop-dev-full
```

### Мониторинг
```bash
# Запуск полного стека мониторинга (Prometheus, Grafana, Loki, ELK)
make start-monitoring

# Запуск только Prometheus stack
make start-prometheus-stack

# Запуск только ELK stack
make start-elk-stack

# Остановка мониторинга
make stop-monitoring

# Проверка состояния мониторинга
make check-monitoring-health

# Просмотр логов мониторинга
make logs-prometheus
make logs-grafana
make logs-loki
make logs-elasticsearch
make logs-kibana
```

### Доступ к сервисам мониторинга
- **Prometheus**: http://localhost:9090 - метрики и мониторинг
- **Grafana**: http://localhost:3000 (admin/admin) - дашборды и визуализация
- **Loki**: http://localhost:3100 - система сбора логов
- **Kibana**: http://localhost:5601 - анализ логов ELK
- **Elasticsearch**: http://localhost:9200 - поисковая система
- **PgAdmin**: http://localhost:5050 (admin@admin.com/admin) - управление БД
- **Kafka UI**: http://localhost:9091 - управление Kafka

#### Основные команды разработки:
```bash
```bash
# Проверка кода и тесты
make fmt                    # Форматирование кода (gofmt)
make lint                   # Статический анализ кода (go vet + golangci-lint)
make test-unit              # Юнит-тесты с покрытием кода
make test-integration       # Интеграционные тесты (с Docker)
make test-all               # Все тесты: форматирование + линтер + юнит + интеграционные

# CI локально
make ci-local               # Полный CI процесс локально (имитация GitHub Actions)
```

#### Управление инфраструктурой:
```bash
# Настройка репозиториев
make setup-system-tests        # Клонирует/обновляет pinstack-system-tests репозиторий
make setup-monitoring          # Клонирует/обновляет pinstack-monitoring-service репозиторий

# Запуск инфраструктуры
make start-post-infrastructure  # Поднимает все Docker контейнеры для тестов
make check-services            # Проверяет готовность всех сервисов

# Интеграционные тесты
make test-post-integration     # Запускает только интеграционные тесты
make quick-test               # Быстрый запуск тестов без пересборки контейнеров

# Остановка и очистка
make stop-post-infrastructure  # Останавливает все тестовые контейнеры
make clean-post-infrastructure # Полная очистка (контейнеры + volumes + образы)
make clean                    # Полная очистка проекта + Docker

# Быстрое тестирование с локальным кодом
make quick-test-local         # Быстрый запуск тестов с локальным post-service (без GitHub)
```

#### Redis команды для отладки:
```bash
# Redis инструменты
make redis-cli               # Подключение к Redis CLI
make redis-info              # Информация о Redis
make redis-keys              # Просмотр всех ключей в кеше
make redis-flush             # Очистка всех данных Redis (с подтверждением)
```

#### Логи и отладка:
```

#### Управление инфраструктурой:
```bash
# Настройка репозитория
make setup-system-tests        # Клонирует/обновляет pinstack-system-tests репозиторий

# Запуск инфраструктуры
make start-post-infrastructure  # Поднимает все Docker контейнеры для тестов
make check-services            # Проверяет готовность всех сервисов

# Интеграционные тесты
make test-post-integration     # Запускает только интеграционные тесты
make quick-test               # Быстрый запуск тестов без пересборки контейнеров

# Остановка и очистка
make stop-post-infrastructure  # Останавливает все тестовые контейнеры
make clean-post-infrastructure # Полная очистка (контейнеры + volumes + образы)
make clean                    # Полная очистка проекта + Docker

# Быстрое тестирование с локальным кодом
make quick-test-local         # Быстрый запуск тестов с локальным post-service (без GitHub)
```

#### Redis команды для отладки:
```bash
# Redis инструменты
make redis-cli               # Подключение к Redis CLI
make redis-info              # Информация о Redis
make redis-keys              # Просмотр всех ключей в кеше
make redis-flush             # Очистка всех данных Redis (с подтверждением)
```

#### Логи и отладка:
```bash
# Просмотр логов сервисов
make logs-user              # Логи User Service
make logs-auth              # Логи Auth Service  
make logs-post              # Логи Post Service
make logs-gateway           # Логи API Gateway
make logs-db                # Логи User Database
make logs-auth-db           # Логи Auth Database
make logs-post-db           # Логи Post Database
make logs-redis             # Логи Redis

# Экстренная очистка
make clean-docker-force     # Удаляет ВСЕ Docker ресурсы (с подтверждением)
```

### Зависимости для интеграционных тестов 🐳

Для интеграционных тестов автоматически поднимаются контейнеры:
- **user-db-test** — PostgreSQL для User Service
- **user-migrator-test** — миграции User Service  
- **user-service-test** — сам User Service
- **auth-db-test** — PostgreSQL для Auth Service
- **auth-migrator-test** — миграции Auth Service
- **auth-service-test** — Auth Service
- **post-db-test** — PostgreSQL для Post Service
- **post-migrator-test** — миграции Post Service
- **post-service-test** — сам Post Service
- **api-gateway-test** — API Gateway
- **redis** — Redis для кеширования

> 📍 **Требования:** Docker, docker-compose  
> 🚀 **Все сервисы собираются автоматически из Git репозиториев**  
> 🔄 **Репозитории `pinstack-system-tests` и `pinstack-monitoring-service` клонируются автоматически при запуске**

### Быстрый старт разработки ⚡

```bash
# 1. Проверить код
make fmt lint

# 2. Запустить юнит-тесты
make test-unit

# 3. Запустить интеграционные тесты
make test-integration

# 4. Или всё сразу
make ci-local

# 5. Запуск с мониторингом
make start-dev-full

# 6. Очистка после работы
make clean
```

### Особенности 🔧

- **Кеширование:** Полная интеграция с Redis для оптимизации производительности
- **Отключение кеша тестов:** все тесты запускаются с флагом `-count=1`
- **Фокус на Post Service:** интеграционные тесты тестируют только Post endpoints
- **Автоочистка:** CI автоматически удаляет все Docker ресурсы после себя
- **Параллельность:** в CI юнит и интеграционные тесты запускаются последовательно
- **Мониторинг:** Полная интеграция с Prometheus, Grafana, Loki и ELK stack

> ✅ Сервис готов к использованию.