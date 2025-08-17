# Pinstack Post Service 📝

**Pinstack Post Service** — микросервис для управления постами в системе **Pinstack**.

## Основные функции:
- CRUD-операции для постов (создание, чтение, обновление, удаление).
- Хранение данных постов.
- Взаимодействие с другими микросервисами через gRPC.
- **Redis кеширование** для оптимизации производительности.

## Технологии:
- **Go** — основной язык разработки.
- **gRPC** — для межсервисной коммуникации.
- **PostgreSQL** — основная база данных.
- **Redis** — кеширование данных.
- **Docker** — для контейнеризации.

## Архитектура кеширования 🗄️

### Стратегия кеширования:
- **User Cache** — кеширование пользователей (15 мин TTL) для решения N+1 запросов
- **Post Cache** — кеширование детальной информации о постах (30 мин TTL)

### Реализация:
- **Decorator Pattern** — `PostServiceCacheDecorator` оборачивает основной сервис
- **Hexagonal Architecture** — кеш интегрирован через порты и адаптеры
- **Smart Invalidation** — автоматическая инвалидация при обновлении/удалении
- **Error Resilience** — при ошибках кеша обращается к основному источнику данных

### Оптимизируемые операции:
1. **Список постов** — кеширование пользователей для избежания N+1 запросов
2. **Детали поста** — полное кеширование с вложенными объектами


## CI/CD Pipeline 🚀

### GitHub Actions
Проект использует GitHub Actions для автоматического тестирования при каждом push/PR.

**Этапы CI:**
1. **Unit Tests** — юнит-тесты с покрытием кода
2. **Integration Tests** — интеграционные тесты с полной инфраструктурой 
3. **Auto Cleanup** — автоматическая очистка Docker ресурсов

### Makefile команды 📋

#### Основные команды разработки:
```bash
# Проверка кода и тесты
make fmt                    # Форматирование кода (gofmt)
make lint                   # Проверка кода (go vet)
make test-unit              # Юнит-тесты с покрытием
make test-integration       # Интеграционные тесты (с Docker)
make test-all               # Все тесты: форматирование + линтер + юнит + интеграционные

# CI локально
make ci-local               # Полный CI процесс локально (имитация GitHub Actions)
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
- **redis** — Redis для кеширования
- **api-gateway-test** — API Gateway

> 📍 **Требования:** Docker, docker-compose  
> 🚀 **Все сервисы собираются автоматически из Git репозиториев**  
> 🔄 **Репозиторий `pinstack-system-tests` клонируется автоматически при запуске тестов**

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

# 5. Очистка после работы
make clean
```

### Особенности 🔧

- **Отключение кеша тестов:** все тесты запускаются с флагом `-count=1`
- **Фокус на Post Service:** интеграционные тесты тестируют только Post endpoints
- **Автоочистка:** CI автоматически удаляет все Docker ресурсы после себя
- **Параллельность:** в CI юнит и интеграционные тесты запускаются последовательно

> ✅ Сервис готов к использованию.