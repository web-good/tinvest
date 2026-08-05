# Доступ к Tinkoff Invest API: корневой сертификат Минцифры

Сертификат `invest-public-api.tinkoff.ru` подписан корнем **Russian Trusted Root CA**
(Минцифры), которого нет ни в одном стандартном хранилище доверия. Без него любая
команда проекта падает так:

```
tls: failed to verify certificate: x509: certificate signed by unknown authority
```

Бьёт по всему сразу: `cmd/main`, `cmd/backtest`, `cmd/pullscreen`, `cmd/divscreen`.

## Починка (один раз на машину)

Из корня проекта:

```bash
sudo cp deploy/certs/russian_trusted_root_ca.crt /usr/local/share/ca-certificates/ && sudo update-ca-certificates
```

Чинит для всего сразу — терминала, GoLand, любых `go run`. Переменные окружения после
этого не нужны.

Проверка:

```bash
go run ./cmd/pullscreen -tickers UGLD -months 12 -top 5
```

## Откат

```bash
sudo rm /usr/local/share/ca-certificates/russian_trusted_root_ca.crt && sudo update-ca-certificates --fresh
```

## Разовый вариант, без sudo

Если ставить в систему не хочется — собрать бандл «системные CA + этот корень» и
передавать его через `SSL_CERT_FILE` (Go читает эту переменную):

```bash
cat /etc/ssl/certs/ca-certificates.crt deploy/certs/russian_trusted_root_ca.crt > ~/.certs/tinvest-bundle.pem
SSL_CERT_FILE=$HOME/.certs/tinvest-bundle.pem go run ./cmd/pullscreen -tickers UGLD -months 12 -top 5
```

Переменная должна стоять **в той же строке** (или быть в `export` этой сессии терминала).
В GoLand оболочечный `export` не подхватывается — env прописывается в Run Configuration.

## Сертификат

`deploy/certs/russian_trusted_root_ca.crt` — корневой сертификат, скачан с `gu-st.ru`:

```
CN     = Russian Trusted Root CA
sha256 = D2:6D:2D:02:31:B7:C3:9F:92:CC:73:85:12:BA:54:10:35:19:E4:40:5D:68:B5:BD:70:3E:97:88:CA:8E:CF:31
годен до 2032-02-27
```

Промежуточный (`Russian Trusted Sub CA`) не нужен — API отдаёт его в цепочке сам.

## Прод

Тот же корень вшит в образ (`Dockerfile`): `ca-certificates` + копирование сертификата +
`update-ca-certificates`. Без этого прод-контейнер (`alpine:latest` + `tzdata`) к API не
подключается — проверено сборкой: базовый alpine даёт `Verify return code: 19`, образ с
правкой — `0 (ok)`.
