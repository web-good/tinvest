# Сборочный тулинг: `magefiles/` (mage)

`magefiles/mage.go` — единая точка входа для проверок качества: линт, тесты и генерация/проверка
mock-ов. Это Go-код, исполняемый через [magefile/mage](https://magefile.org): каждая **экспортированная
функция** пакета становится таргетом командной строки (mage переводит имя в нижний регистр —
`func Lint()` → `mage lint`).

Инструменты (`golangci-lint`, `mockery`) ставятся в локальную папку `./bin` c закреплёнными версиями,
так же, как `make install-deps`. `./bin` игнорируется git — бинарники не коммитятся.

---

## Предпосылки

- **Go 1.25** (проект требует именно её; CI поднимает через `actions/setup-go`).
- Бинарь `mage`. В репозитории он уже лежит в `./bin/mage` (отслеживается git). Альтернатива без
  предустановки — `go run github.com/magefile/mage <target>` (так делает CI; зависимость
  `github.com/magefile/mage v1.17.2` есть в `go.mod`).

Все команды запускаются **из корня репозитория**.

---

## Таргеты

| Команда | Что делает |
|---|---|
| `mage tools` | Ставит `golangci-lint` и `mockery` в `./bin` на закреплённых версиях. |
| `mage lint` | `golangci-lint run ./...` по всему модулю. |
| `mage test` | `go test -race ./...` — весь набор тестов с race-детектором. |
| `mage mocks` | Перегенерирует все mock-и по `.mockery.yaml`. |
| `mage mocksCheck` | Генерит mock-и и падает, если рабочее дерево изменилось (страж дрейфа). |
| `mage ci` | Полный набор: `lint` → `test` → `mocksCheck`. То, что гоняет CI. |

Список всегда можно получить локально: `./bin/mage -l`.

Закреплённые версии инструментов лежат константами в начале `magefiles/mage.go`
(`golangciLintVersion`, `mockeryVersion`). Меняются **только** там — и синхронно с CI.

---

## Типовые сценарии

### Первичная настройка (после клона / смены версий)

```bash
./bin/mage tools     # поставит golangci-lint + mockery в ./bin
./bin/mage -l        # проверить, что таргеты видны
```

Проверка версий (должны совпасть с константами в `mage.go`):

```bash
./bin/golangci-lint version   # 2.12.2 ... built with go1.25.x
./bin/mockery --version       # v2.53.4
```

### Перед коммитом / пушем

```bash
./bin/mage ci
```

Зелёный `mage ci` = ровно то, что проверит CI-гейт: линт `0 issues`, все тесты проходят под `-race`,
mock-и не разошлись с интерфейсами. Если чинишь по частям — `mage lint` и `mage test` по отдельности.

> Совет по линту: у `golangci-lint` дефолтные лимиты (`max-same-issues`, `max-issues-per-linter`)
> скрывают повторяющиеся находки. Чтобы увидеть **все**:
> `./bin/golangci-lint run ./... --max-same-issues=0 --max-issues-per-linter=0`.

### Добавить mock для нового интерфейса

1. Добавь интерфейс в `.mockery.yaml` (см. раздел ниже).
2. Перегенерируй: `./bin/mage mocks`.
3. Собери и проверь: `go build ./internal/... ./pkg/... ./cmd/...`.
4. Закоммить сгенерированные файлы `**/mocks/mock_*.go` вместе с `.mockery.yaml`.
5. Убедись, что страж чистый: `./bin/mage mocksCheck` → exit 0.

### Изменил интерфейс, у которого есть mock

Перегенерируй mock и закоммить его в том же коммите, что и правку интерфейса:

```bash
./bin/mage mocks
./bin/mage test          # тесты должны остаться зелёными
git add <интерфейс> '**/mocks/**'
```

Забудешь — CI упадёт на `mocksCheck` (см. «Страж дрейфа»).

---

## Как устроена генерация mock-ов (`.mockery.yaml`)

Используется схема **mockery v2** (`packages:`). Ключевые настройки:

- `with-expecter: true` — генерится типобезопасный API `.EXPECT().Method(...)`.
- `dir: "{{.InterfaceDir}}/mocks"` — mock кладётся в подпакет `mocks/` **рядом** с интерфейсом
  (локальные пути импорта, без одного гигантского дерева).
- `mockname: "Mock{{.InterfaceName}}"`, `filename: "mock_{{.InterfaceName}}.go"`, `outpkg: "mocks"`.

Каждый интерфейс перечислен явно под своим пакетом. Для пакетов, где нужно мокать **только**
отдельные интерфейсы (а не все подряд), задаётся `config: { all: false }`. Неэкспортированные
интерфейсы мокать можно — сгенерированный тип всё равно экспортируется (`Mock` + имя как есть,
например `MockcandleFetcher`).

В тестах mock создаётся конструктором с `*testing.T`:

```go
m := mocks.NewMockOrdersClient(t)
m.EXPECT().PostOrder(mock.Anything, mock.Anything).Return(resp, nil)
```

Передача `t` включает авто-`AssertExpectations` через `t.Cleanup` и падение теста на неожиданном
вызове. Практика по ожиданиям:

- вызов происходит **безусловно** на пути теста → обычное ожидание (или `.Times(n)`);
- вызов **может не случиться** на данном пути → `.Maybe()`;
- нужно проверить аргументы → `mock.MatchedBy(...)` или `.Run(...)`;
- метод **не должен** вызываться → просто не задавай ожидание (неожиданный вызов уронит тест).

Не ставь `.Maybe()` на безусловный вызов — это молча ослабляет проверку.

Линт исключает сгенерированный код из проверок (`.golangci.yml`, секции
`linters.exclusions.paths` и `formatters.exclusions.paths`): `internal/pb`, `internal/converter`,
`pkg/client/grpc/converter` и `mocks`.

---

## Страж дрейфа mock-ов (`mocksCheck`)

`mage mocksCheck` перегенерирует mock-и и запускает `git diff --exit-code -- '**/mocks/**'`.
Если сгенерированное отличается от закоммиченного (интерфейс поменяли, а mock не обновили) —
diff непустой, команда падает, CI краснеет.

Проверить, что страж живой, можно вручную: временно измени сигнатуру метода замоканного
интерфейса → `mage mocksCheck` упадёт с непустым diff; откати правку и перегенерируй → снова зелёно.

> Ограничение: `git diff` ловит только **отслеживаемые** файлы. Если новый интерфейс порождает
> совершенно новый (untracked) mock-файл, страж этого не заметит — но добавление интерфейса всегда
> идёт через явную правку `.mockery.yaml`, так что на практике не теряется.

---

## Интеграция с CI

`.github/workflows/main.yaml`, job `checks` (гоняется на каждый push/PR):

```yaml
- uses: actions/setup-go@v5
  with: { go-version: '1.25', cache: true }
- run: go run github.com/magefile/mage tools
- run: go run github.com/magefile/mage ci
```

`image-build-and-push` завязан на `needs: checks` и условие push-в-`main`, `deploy-image` наследует
гейт транзитивно. То есть сборка и деплой не стартуют, пока `mage ci` не зелёный.

---

## Подводные камни

- **`go build ./...` спотыкается на пакете `magefiles`** с ошибкой `function main is undeclared`.
  Это ожидаемо: у `magefiles/` нет `func main()` — mage генерирует её на лету. Для проверки сборки
  используй ограниченный путь: `go build ./internal/... ./pkg/... ./cmd/...`. Линт/тесты/CI это
  не затрагивает (`golangci-lint` даёт 0 issues, `go test` видит «no test files»).
- **Без тега `//go:build mage`.** В layout'е с директорией `magefiles/` тег не нужен (он только для
  однофайлового варианта в корне). Файл — обычный `package main`.
- **Версии инструментов — только через константы** в `mage.go`, синхронно с CI. Не прыгай на
  mockery v3 (другая схема конфига).
- **`./bin` игнорируется git** — установленные `mage tools` бинарники не коммить.
