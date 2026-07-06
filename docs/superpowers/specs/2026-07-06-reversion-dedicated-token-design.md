# Reversion — выделенный токен для торговли (design)

**Дата:** 2026-07-06
**Ветка:** `feat/reversion-rsi-dip`
**Статус:** одобрено, готово к плану

## Проблема

Сейчас для reversion-стратегии через окружение задаётся только номер счёта
(`REVERSION_ACCOUNT_ID`). Вся торговля идёт через **единственный общий gRPC-клиент
приложения**, построенный из общего токена `T_BANK` (`internal/service_provider/client.go:19`).

Нужно, чтобы reversion торговала под **своим** API-токеном Tinkoff, задаваемым
отдельной переменной окружения. Это нужно, когда счёт reversion живёт под отдельным
логином Tinkoff (свой токен), а не как второй счёт того же логина.

### Сопутствующее открытие (обязательный фикс)

Путь выставления ордеров **не аутентифицирован**:

- `NewOrdersServiceClient(conn)` не получает токен (`pkg/client/grpc/orders_service_client.go:18`).
- `PostOrder` не прикрепляет auth-credential, а executor вызывает его без опций
  (`pkg/client/grpc/orders_service_client.go:28`, `internal/service/trading_strategy/reversion/live/executor/executor.go:62`).

Остальные суб-клиенты (operations/marketdata/instruments/users) хранят `*Auth` и
прикрепляют `Bearer`-токен к каждому вызову через `NewRPCCredential`. Клиент ордеров —
нет. Значит реальная торговля упадёт на `UNAUTHENTICATED` под **любым** токеном.
Пока `TRADE_ENABLED` в проде не включали, дефект не всплывал. Он входит в объём этой
работы, потому что без него отдельный токен для ордеров бессмыслен.

## Решение

Выделенный gRPC-клиент + токен для reversion, полностью изолированный от общего
клиента приложения, плюс фикс auth для пути ордеров.

### 1. Конфиг

Добавить в `internal/config/reversion.go` в `ReversionConfig`:

```go
Token string `config:"REVERSION_TOKEN,required,backend=env"`
```

- Поле **обязательное** (`required`) — без токена приложение не стартует. Fallback на
  `T_BANK` не делаем: явное задание исключает случайную торговлю не тем токеном.
- Токен — **секрет**: инжектируется как переменная окружения контейнера (как `T_BANK`,
  `TELEGRAM`), **не** коммитится в `env/prod.env`.
- `NewReversionConfig()` дефолт для `Token` не задаёт (оставляем пустым; required-поле
  обязан заполнить оператор).

### 2. Выделенный gRPC-клиент reversion

В `internal/service_provider` добавить кэширующий геттер:

```go
func (s *ServiceProvider) GetReversionGrpcClient() (internalgrpc.GrpcClient, error)
```

- Лениво строит **второй** `GrpcClient` через
  `internalgrpc.NewClientGrpc(s.appConfig.GrpcClient.AddressProd, s.appConfig.Reversion.Token)`
  и кэширует в структуре `client` рядом с существующим `grpcClient`.
- `GetReversionLiveService()` (`internal/service_provider/service.go:229`) берёт свои
  четыре суб-клиента (Instruments, MarketData, Operations, Orders) из этого клиента
  вместо общего `GetGrpcClient()`.

Итог: любой вызов Tinkoff, который делает reversion (свечи, портфель/кэш, история
сделок, выставление ордеров), идёт под `REVERSION_TOKEN`.

**Почему второй коннект, а не переиспользование:** второй gRPC-коннект к тому же хосту
для часовой cron-стратегии стоит ничтожно (один почти простаивающий сокет). Взамен —
полная изоляция и минимальные правки. Market-data-вызовы аккаунт-независимы, поэтому
использовать для них токен reversion корректно.

### 3. Фикс auth для ордеров

- `NewOrdersServiceClient(conn grpc.ClientConnInterface, token string)` — хранит
  `auth *Auth` (как `operationsServiceClient`).
- `PostOrder` добавляет `NewRPCCredential(c.auth)` в передаваемые опции вызова.
- Обновить единственный call-site `pkg/client/grpc/grpc.go:78` — передать `token`.

Радиус: `NewOrdersServiceClient` вызывается только в `grpc.go:78` (там `token` уже в
области видимости); единственный реальный вызов `PostOrder` — reversion-executor.
Скальпинг этот путь не использует. Общий клиент приложения от этого фикса только
выигрывает (ордера начинают аутентифицироваться под `T_BANK`).

### 4. Тесты

- **orders client (unit):** `PostOrder` прикрепляет `Bearer`-credential — через стаб
  `investapi.OrdersServiceClient`, перехватывающий `CallOption` / метаданные, проверить
  что `Authorization: Bearer <token>` уходит.
- **config (unit):** загрузка `ReversionConfig` падает, если `REVERSION_TOKEN` не задан
  (проверка `required`).
- **verify (ручной):** старт приложения с заданным `REVERSION_TOKEN` и
  `TRADE_ENABLED=false` (бумажный режим) → приложение стартует, read-вызовы reversion
  (портфель, свечи) не дают auth-ошибок.

### 5. Документация и env

- `docs/reversion/live-runner.md`: добавить `REVERSION_TOKEN` в таблицу переменных
  окружения — обязательная, секрет, инжектится в рантайме; аутентифицирует собственный
  gRPC-клиент reversion; кейс — счёт под отдельным логином Tinkoff.
- `env/prod.env.example`: строка `REVERSION_TOKEN=` с пометкой «секрет, задать в
  рантайме, не коммитить значение».

## Вне объёма (YAGNI)

- Не трогаем общий `T_BANK`-клиент и другие стратегии (кроме побочного бенефита от
  фикса auth ордеров).
- Не переиспользуем существующий gRPC-коннект ради экономии одного сокета.
- Fallback `REVERSION_TOKEN` → `T_BANK` не делаем (выбран required).
- Отдельный адрес/эндпоинт для reversion не вводим — используем тот же `AddressProd`.

## Затрагиваемые файлы

- `internal/config/reversion.go` — поле `Token`.
- `internal/service_provider/client.go` — `GetReversionGrpcClient()` + поле кэша.
- `internal/service_provider/service.go` — `GetReversionLiveService()` берёт суб-клиенты
  из reversion-клиента.
- `pkg/client/grpc/orders_service_client.go` — токен + auth в `PostOrder`.
- `pkg/client/grpc/grpc.go` — передать token в `NewOrdersServiceClient`.
- `docs/reversion/live-runner.md`, `env/prod.env.example` — документация/пример.
- Тесты: `pkg/client/grpc/*_test.go`, `internal/config/*_test.go` (по месту).
