# Reversion: выход-в-безубыток (exit `BE`) — дизайн

**Дата:** 2026-06-16
**Ветка:** feat/reversion-rsi-dip
**Мотивация:** анализ убытков (`reports/_analysis/reversion_loss_analysis_2026-06-16.md`)
показал, что на MDMG **7 из 12** убыточных сделок успевали уйти в плюс >1% (MFE), затем
разворачивались и выбивали ATR-стоп в −0.5…−1.5%. Снятие ATR-стопа MDMG **ухудшило**
(−985 → −2850), т.е. проблема не в стопе, а в отдаче набранной прибыли (give-back).
Лечение — защитный пол на уровне цены входа, взводимый после достаточного хода в плюс.

## Цель

Добавить опциональный выход **чистый безубыток**: после того как сделка ушла в плюс на
`BreakevenArmATR × EntryATR`, при возврате цены к цене входа закрыть позицию по цене входа
(reason `BE`). «Чистый» = выход ровно по `PurchasePrice`, без фиксации буфера прибыли.

## Параметры (станет 20)

| Параметр | Тип | Дефолт | Назначение |
|---|---|---|---|
| `UseBreakeven` | int (0/1) | `0` | флаг включения; `0` сохраняет текущее поведение |
| `BreakevenArmATR` | float64 | `1.0` | порог взвода в кратных EntryATR |

Оба свипаются рефлексией в гридах и хардкодятся по тикеру — как все прочие фичи.

## Логика

### Состояние (без изменений движка)
`Position.MaxFavorablePrice` уже ведётся движком (`portfolio.go`: монотонный максимум
закрытий с момента входа, сидируется ценой входа). `manage()` читает его из `in.pos` —
**новое состояние/изменения движка не требуются**.

### Взвод (armed)
```
armed = in.pos.MaxFavorablePrice >= in.pos.PurchasePrice + BreakevenArmATR*in.pos.EntryATR
```
Монотонность `MaxFavorablePrice` делает латч взвода необратимым.

### Срабатывание `BE`
```
UseBreakeven==1 && in.pos.EntryATR>0 && BreakevenArmATR>0 && armed && in.price <= in.pos.PurchasePrice
→ SignalSell, Reason="BE", fill по закрытию
```
Тройной guard (`UseBreakeven==1 && EntryATR>0 && BreakevenArmATR>0`) держит выход инертным
в live-торговле (EntryATR там не персистится) и при нулевом пороге — по образцу ATR-стопа.

**Нюанс close-fill (важно):** движок (`engine.go`) исполняет все выходы reversion по
закрытию бара; спец-цена есть только у reason `"SL"/"TRAIL"/"TP"`, а `BE` (как и `ATRSL`)
к ним не относится. Значит «чистый безубыток» на практике = выход по **первому закрытию,
которое вернулось к/ниже цены входа** после взвода. Это закрытие может быть чуть ниже
`PurchasePrice` (возможен небольшой минус) — выход не гарантирует ровный ноль, а режет
give-back до «около нуля». Это сознательно и консистентно с close-fill семантикой ATR-стопа.

### Приоритет выходов
```
OB → RSI50 → BE → средний(RSIOS | ATRSL) → EMAX
```
- **BE выше среднего выхода:** уровень BE (`PurchasePrice`) выше уровня ATR-стопа
  (`PurchasePrice − mult×ATR`); при падении цены BE достигается раньше, поэтому на баре,
  где сработали бы оба, репортится `BE` и убыток режется до ~0. Это прямое лечение give-back.
- **RSI50 выше BE:** если импульс затух, пока цена ещё выше входа, RSI50 заберёт маленький
  плюс (лучше безубытка). BE — защитный пол, не замена момент-выходу.
- OB остаётся первым; с BE взаимоисключающ на практике (перекупленность ⇒ цена высоко,
  `price ≤ PurchasePrice` ложно).

### Доступность ATR
Сейчас `buildInput` считает ATR только при `UseATRStop==1`, и `decide()` штампует
`sig.ATR=in.atr` на покупке (движок → `Position.EntryATR`). Чтобы BE работал самодостаточно:
- `buildInput`: считать ATR при `(UseATRStop==1 || UseBreakeven==1) && ATRPeriod>0`.
- `Lookback`: добавлять окно `ATRPeriod+1` под тем же расширенным условием.
Для MDMG не критично (`UseATRStop=1`), но делает фичу независимой от ATR-стопа.

## Изменения по файлам

- `core.go`:
  - `Params`: + `UseBreakeven int`, `BreakevenArmATR float64`.
  - `buildInput`/`Lookback`: расширить ATR-гейт на `UseBreakeven==1`.
  - `manage()`: новый `case` для `BE` между RSI50 и средним выходом.
  - `decideInput` уже несёт `pos` — `MaxFavorablePrice` доступен через `in.pos`.
- `reversion_registry.go` / `ParseParams`: без изменений (JSON-unmarshal partial overrides
  подхватит новые поля по рефлексии).
- Все 8 per-ticker `DefaultParams` + `genericReversionDefaults`: + `UseBreakeven:0,
  BreakevenArmATR:1.0`. **MDMG**: `UseBreakeven:1` + обновить остальные параметры до
  победителя калибровки (как сделано для NVTK).
- Все 8 `reversion_grid.json`: добавить в подходящую фазу `UseBreakeven:[0,1],
  BreakevenArmATR:[0.5,1.0,1.5]`.
- `docs/reversion/strategy.md` + doc-comment `core.go`: описать выход `BE` и 2 параметра.

## Тесты (TDD)

Core (`core_test.go`):
- `TestExitBreakevenFiresAfterArm` — взвод (MFP ≥ entry+arm×ATR), затем price ≤ entry → `BE`.
- `TestNoBreakevenBeforeArm` — MFP не достигал порога → BE не срабатывает.
- `TestBreakevenOffByFlag` — `UseBreakeven=0` → не срабатывает.
- `TestBreakevenInertWhenEntryATRZero` — live-сценарий (EntryATR=0).
- `TestBreakevenInertWhenArmZero` — `BreakevenArmATR=0`.
- `TestExitPrecedenceBreakevenOverMiddle` — armed + price ниже ATR-стопа → reason `BE`, не `ATRSL`.
- `TestExitPrecedenceRSI50OverBreakeven` — RSI50-кросс + armed + price≤entry на одном баре → `RSI50`.
- `TestBuildInputATRForBreakeven` — `UseBreakeven=1, UseATRStop=0` ⇒ atr посчитан.
- `TestLookbackIncludesATRForBreakeven` — Lookback учитывает окно ATR при `UseBreakeven=1`.

## Проверка (после реализации)

A/B на MDMG, OOS, тот же протокол:
```bash
# baseline уже есть (best.md со стопом, без BE)
go run ./cmd/backtest -ticker MDMG -strategy reversion \
  -calibrate data/params/mdmg/reversion_be.json -out ./reports/MDMG \
  -months 50 -test-months 12 -metric net_pnl
```
где `reversion_be.json` — пиннинг-грид победителя MDMG c `UseBreakeven:1, BreakevenArmATR:1.0`.
Успех = чистый PnL и PF выше текущих (−985 / 0.87) при сопоставимом числе сделок.

## Вне области (YAGNI)

- Фиксация буфера прибыли (выход по `entry + M×ATR`) — пользователь выбрал **чистый** безубыток.
- Трейлинг-стоп (следование за `MaxFavorablePrice`) — отдельная фича, не сейчас.
- Порог взвода в % вместо ATR — придерживаемся ATR ради консистентности с ATR-стопом.
