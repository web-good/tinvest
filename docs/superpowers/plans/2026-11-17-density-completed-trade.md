# DENSITY completed trade notification implementation plan

> **For agentic workers:** Use superpowers:executing-plans. Steps use checkbox syntax.

**Goal:** Отправлять одно уведомление после полного закрытия виртуальной сделки.
**Architecture:** Форматировать снимок закрытого paper.Order; среднюю цену продажи
получать из накопленного GrossPnL. Сохранить существующую outbox и мигрировать
структурированные сообщения при её загрузке.
**Tech Stack:** Go, существующие paper engine и JSON outbox.
**Spec:** docs/superpowers/specs/2026-11-17-density-completed-trade-design.md

## Global Constraints

Не менять исполнение сделок и реальные уведомления. Не отправлять отчёты
об открытых, частично закрытых, отменённых и unresolved сделках.

## Review Focus

Частичный выход; исходная плотность после переноса; восстановление старой очереди;
несколько последовательных сделок; медленный Telegram и повторные отправки.

## Task 1: Итоговое уведомление

**Files:** simulation/notification.go, outbox.go, notification_test.go,
service_test.go, follow_screener_test.go в internal/service/screener/density/.
**Interfaces:** FormatFill(paper.Record) string, OpenOutbox(string) (*Outbox, error).

- [x] Обновить тесты формата, очереди и интеграции для закрытых сделок; проверить
  отсутствие сообщений при входе и частичном выходе и восстановление очереди.
- [x] Запустить тесты и убедиться в ожидаемом падении старого поведения.
- [x] Обновить форматтер и загрузку очереди, сохранив журнал всех событий.
- [x] Запустить тесты DENSITY с race, линтер и общий CI gate.
- [x] Проверить diff и обновить описание механики симуляционных уведомлений.

## Verification

`go test -race ./internal/service/screener/density/...` — PASS.
`./bin/mage ci` — PASS (lint, mock drift, all tests with race).
Independent review: no actionable findings.
