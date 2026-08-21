# ELFV под `rsi_pullback` — план подготовки, калибровки и вердикта

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** довести ELFV (ЭЛ5-Энерго) до вердикта по стратегии `rsi_pullback`: каталог сеток с осями
по замерам инструмента, пакет параметров, десять тем rolling walk-forward, литерал и заведение в
боевую вселенную четырнадцатым тикером.

**Architecture:** тикер-специфичный каталог `data/params/rsi_pullback/elfv/` (десять однотемных
файлов + точка) + сторожевой тест осей в `internal/service/backtest` + пакет параметров
`strategy/elfv` (сначала baseline-состояние, затем литерал со снимком) + записи в двух реестрах
(бэктест и живой раннер) + заведение в боевую вселенную + разделы документации. Прогоны идут по
ШТАТНОЙ схеме каталога `-months 36 -train-months 12 -test-months 6`.

**Tech Stack:** Go 1.25, `cmd/backtest` (grid-search + rolling walk-forward), `cmd/pullparity`,
`go test`, `./bin/golangci-lint`, `./bin/mage ci`.

**Spec:** `docs/superpowers/specs/2026-08-21-elfv-rsi-pullback-prep-design.md`

## Global Constraints

- Схема прогонов — **`-months 36 -train-months 12 -test-months 6 -metric profit_factor`**,
  `-min-trades 20` (у темы `screen` — `-min-trades 1`). Это штатная схема каталога, поэтому числа
  ELFV сопоставимы построчно с UGLD, GAZP, FESH, WUSH, LENT, RENI, NVTK, LSNGP, SVAV и SIBN — в
  отличие от IVAT (адаптированная 25/9/4) и DOMRF (разведочная 3 фолда 3/2).
- Кэш освежён хвостовой дозагрузкой 2026-08-21 (штатный `CandleProvider.Load`, без `-refresh`):
  `ELFV_Minutes30.json` — **31 362 бара**, окно 2023-08-07 … 2026-08-21; расчётное окно 36 месяцев
  `2023-08-21 … 2026-08-21` содержит **31 141 бар, из них 23 714 будних**. `ELFV_Day1.json` —
  1 141 дневная свеча, в расчётном окне 778 будних дней. Все замеры ниже сняты по нему.
- **Запас истории — 14 дней.** `-months 37` упрётся в начало кэша. Повторный `-refresh` во время
  калибровки НЕ запускать: он сдвинет окно вперёд, укоротит его слева и сделает часть прогонов
  несравнимой с остальными.
- Планка, объявленная ДО прогонов и не пересматриваемая после: темы `entry` и `trend` **обе** дают
  pooled OOS PF ≥ 1.5 при ≥ 20 сделках; ведущая ось темы (`RSILower` для `entry`, `EMASlow` для
  `trend`) выбрана одинаково в ≥ 3 фолдах из 4. Вырожденный фолд (ни одной убыточной сделки) не
  засчитывается в пользу тикера. Условие «≥ 5 сделок в каждом фолде» из плана IVAT здесь НЕ
  применяется — оно было костылём под 25-месячную схему.
- **Четвёртый OOS-фолд (2026-02-21 … 2026-08-21) заведомо тяжёлый**: он накрывает полугодие с
  −30.5%. Слабый четвёртый фолд не переписывает планку и не служит оправданием, но его число
  выписывается отдельной строкой в каждой теме.
- Правило прода, подтверждённое владельцем 2026-08-21 ДО прогонов и применяемое **как есть**:
  литерал ставится и тикер заводится в `RSI_PULLBACK_TICKERS` четырнадцатым даже при непройденной
  планке. **Стоп-условие:** pooled OOS PF принятой точки < 1.0 либо < 20 сделок за 36 месяцев →
  остановиться, принести числа владельцу, задачи 12–15 не выполнять.
- **Априор самый сильный из всех кандидатов каталога, и это записано до прогонов.** Скринер
  (`reports/pullback_screen/pullback_screen_Minutes30_20260804_232456.md`, строка 11): `PFmed` 1.62,
  плато 75%, `Capped` 0/24, **`PFmed HO` 1.26** на 7 сделках. Контрольный baseline-прогон
  `core.DefaultParams()` на 36 месяцах: **181 сделка, PF 1.546, net +52 544 ₽**. Вес обоих фактов
  ограничен: колонка `PFmed HO` в каталоге предсказывает плохо в обе стороны (WUSH с 4.24 планку
  провалил, LSNGP с 0.99 вытянул все девять тем выше 1.5), а baseline считается на полной истории
  без walk-forward. Сильный априор НЕ смягчает планку и не сокращает набор тем.
- Замеры инструмента, на которые ссылаются `_comment` сеток (все — будние бары расчётного окна):
  - кроссы RSI вниз на уровнях 10/15/20/25/30/35/40: RSI(4) 250/517/899/1350/1841/2349/2861;
    RSI(5) 122/290/581/969/1436/1938/2452; RSI(6) 61/180/376/709/1113/1617/2139;
    **RSI(7) 34/115/273/529/892/1381/1915**; RSI(8) 23/82/186/408/747/1186/1715. Для справки
    RSI(3) 555/975/1467/1975/2523/3008/3401 — в сетку не берётся (три бара это шум, правило
    каталога); RSI(9)@10 даёт 14. Планка живого угла каталога — 29 кроссов: RSI(7)@10 = 34 её
    проходит, **RSI(8)@10 = 23 не проходит**, поэтому ось периода обрывается на 7;
  - кроссы RSI вверх (полоса выхода), RSI(4): 55 — 2987, 60 — 2590, 65 — 2100, 70 — 1622,
    75 — 1197, 80 — 808;
  - доля баров с `EMAFast > EMASlow`: **44.0–46.5% на всех 35 парах** (5/50 — 45.5%, 10/100 —
    45.6%, 20/150 — 45.7%, 30/100 — 46.5%, 40/200 — 44.0%). Размах 2.5 п.п. — допуск от выбора
    пары практически не зависит, тот же случай, что на SVAV (42.4–43.6%) и SIBN (45.1–46.3%), и не
    такой, как на IVAT (29.4–36.0%, монотонно);
  - дневной ATR(14): медиана **2.98%** цены, p10 2.18, p90 4.55, n=764 — середина каталога (SIBN
    2.52, LSNGP 2.77, SVAV 4.38). Круг издержек 0.1% = 0.034 ATR: на стопе 0.3 ATR комиссия
    съедает 11.2% риска, на 0.5 — 6.7%, на 0.7 — 4.8%, на 1.0 — 3.4%, на 1.3 — 2.6%. В процентах
    цены: 0.3 ATR = 0.89%, 0.5 = 1.49%, 0.7 = 2.08%, 1.0 = 2.98%, 1.3 = 3.87%;
  - **шаг цены 0.0002 ₽** (все котировки окна кратны ему; медианная цена 0.5134) = 0.039% цены.
    Бэктест наливает по закрытию бара и спред не моделирует, поэтому реальный круг издержек ближе
    к 0.2%, и тогда стоп 0.3 ATR теряет **22.4%** риска на издержки — за чертой 17%, по которой
    строку 0.3 вырезали из `domrf/`. Замер чувствительности (точка `VolMult 2.0`, 97 сделок):
    круг 0.1% → PF 2.003 / net 42 013; 0.2% → PF 1.668 / net 28 934; 0.3% → PF 1.381 / net 17 032;
  - выживаемость стопа (доля будних дней, чей размах достаёт уровня, n=764): 0.3 — 98.8%,
    0.5 — 88.5%, 0.7 — 66.4%, 0.8 — 57.5%, 1.0 — 36.3%, 1.25 — 22.1%, **1.3 — 20.2%**,
    1.5 — 13.2%;
  - день ко второму бару: медиана **0.34 ATR**; ветка «свежий день» ловит 4.2% баров при 0.2,
    6.6% при 0.25, **10.0% при 0.3**, 14.6% при 0.35, 19.6% при 0.4, 29.0% при 0.5; ветка «день
    исчерпан» — 58.8% при 0.6, **48.9% при 0.7**, 40.7% при 0.8, 32.5% при 0.9, 26.0% при 1.0,
    **20.8% при 1.1**, 15.4% при 1.25, 13.5% при 1.3, **8.7% при 1.5**;
  - объёмный гейт (доля баров, проходящих порог) при базе 14 дней: 29.0% при 1.0, 24.6% при 1.2,
    19.9% при 1.5, 14.6% при 2.0, 11.4% при 2.5, **9.2% при 3.0**; база 10 дней — 30.5 / 25.9 /
    21.2 / 15.8 / 12.3 / 10.1%; база 5 дней — 34.1 / 29.4 / 24.4 / 18.7 / 14.9 / 12.5%; база 3 дня —
    38.0 / 33.3 / 28.1 / 22.0 / 18.0 / 15.0%;
  - объёмный гейт в СДЕЛКАХ (точечные прогоны `-params` на дефолтах, `VolBaseDays = 14`,
    `VolLookbackBars = 3`, полная 36-месячная история): выключен → 181 сделка, PF 1.546;
    `VolMult` 1.2 → 127, PF 1.576; **2.0 → 97, PF 2.003**; 2.5 → 83, PF 1.858; **3.0 → 76,
    PF 1.702**; 3.5 → 69, PF 1.473; 4.0 → 61, PF 1.579. На 12-месячное обучающее окно это
    42 / 32 / 28 / 25 / 23 / 20 сделок. У гейта ВНУТРЕННИЙ МАКСИМУМ на 2.0 — противоположность
    SIBN, где ось падала монотонно;
  - окно объёмного гейта в СДЕЛКАХ (`VolMult = 2.0`, `VolBaseDays = 14`): `VolLookbackBars` 1 → 68
    сделок, PF 1.834; 2 → 84, PF 1.752; **3 → 97, PF 2.003**; 5 → 109, PF 1.608; 8 → 129,
    PF 1.811. Ось живая и немонотонная;
  - **покрытие баров**: медиана 34 бара в буднем дне, p10 = **19**, у **147 будних дней из 778
    (18.9%) баров меньше двадцати**; 805 будних баров (3.4%) имеют нулевой размах (`High == Low`);
    баров с нулевым объёмом нет. У SIBN за то же окно было 25 609 будних баров против 23 714 у
    ELFV;
  - оборот (лот 1000, сверен со средним скринера 57 млн): медиана **25.1 млн ₽/день**, p10 6.4,
    p90 159.8, среднее 70.2 — ХУДШАЯ медиана в каталоге (LENT 38, IVAT 43, SVAV 68, SIBN 519).
    Гейт отбора скринера в 50 млн ELFV проходит только средним;
  - гэпы открытия к закрытию предыдущего буднего дня (n=764): медиана +0.04%, p5 −0.69%,
    p95 +1.29%; хуже −3% **один** гэп: 2024-09-03 −3.00% (−0.95 ATR). Глубже 0.5 ATR открывается
    5 дней из 764 (0.7%), глубже 1.0 ATR — 1 (0.1%). ЭЛ5-Энерго не платит дивиденды, и главный
    незакрываемый риск SIBN (пять гэпов хуже −3%, худший −4.69 ATR) здесь отсутствует;
  - режим: всё окно **−49.5%**, пик-минимум **−65.9%**, первые 30 месяцев −24.9%, холдаут
    (последние 6 месяцев) **−32.5%**; полугодия +2.3% / −14.1% / +0.6% / −9.8% / −4.9% / −30.5% —
    растущих два из шести, оба на считанные проценты. Самый жёсткий режимный тест каталога.
- Отчёты прогонов пишутся в `./reports/ELFV_<тема>` (каталог `reports/` в `.gitignore`).
- Коммиты делать по завершении каждой задачи; сообщения на русском, в стиле существующей истории
  (`feat(rsi_pullback): ...`). Ветка — текущая `feat/elfv-pullback-prep` (отведена от
  `feat/sibn-pullback-prep`, HEAD `ec5f10b`; дизайн лежит коммитом `d498e2b`).

---

### Task 1: Каталог сеток ELFV со сторожевым тестом осей

**Files:**
- Create: `data/params/rsi_pullback/elfv/cal_screen.json`, `cal_entry.json`, `cal_trend.json`,
  `cal_day.json`, `cal_day_spent.json`, `cal_volume.json`, `cal_vol_window.json`, `cal_risk.json`,
  `cal_exit.json`, `cal_trail.json`
- Test: `internal/service/backtest/rsi_pullback_elfv_grid_test.go`

**Interfaces:**
- Consumes: хелперы `rsiPullbackTickerGrid(t, ticker, file)`
  (`internal/service/backtest/rsi_pullback_grid_test.go:40`), `sameSet(got, want...)`
  (`rsi_pullback_reni_grid_test.go:17`) и `containsValue(axis, want)`
  (`rsi_pullback_ivat_grid_test.go:88`). **Все три уже объявлены в пакете — НЕ переобъявлять**,
  иначе пакет не соберётся.
- Produces: каталог `elfv/`, на который ссылаются все прогоны Task 3–10, и функцию
  `elfvGrid(t, file)`.

- [ ] **Step 1: Написать падающий тест осей**

Создать `internal/service/backtest/rsi_pullback_elfv_grid_test.go`:

```go
package backtest

import "testing"

// elfvGrid читает файл сеток ELFV через общий хелпер.
func elfvGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "elfv", file)
}

// TestELFVGridsPinTheirMeasuredAxes сторожит оси каталога elfv/. Каталог собран 2026-08-21
// копированием формы sibn/ с пересадкой каждой оси на замеры самого ELFV (31 362 30-минутных
// бара в кэше, 31 141 в расчётном окне 36.0 месяца с 2023-08-21, из них 23 714 будних). Четыре
// решения отличают этот каталог от образца, и каждое опирается на замер:
//
//   - RSIPeriod СУЖЕН до 7 — обратное тому, что сделали на SIBN. Планка живого угла в каталоге —
//     29 кроссов (столько дал слабейший угол LSNGP, объявленный мёртвым; у RENI было 23).
//     RSI(7)@10 даёт 34 будних кросса за 36 месяцев и проходит её, RSI(8)@10 даёт 23 и не
//     проходит. На SIBN тот же замер давал 34 у RSI(8), и ось расширяли; здесь бумага дышит
//     крупнее (дневной ATR 2.98% против 2.52%), и медленный период умирает раньше.
//   - VolMult РАСШИРЕН до 3.0. Решает не доля баров (при базе 14 порог 3.0 пропускает 9.2%), а
//     счёт сделок и форма кривой: точечные прогоны дают 2.0 -> 97 сделок при PF 2.003 (внутренний
//     максимум оси), 3.0 -> 76 сделок при PF 1.702, то есть 25 сделок на 12-месячное обучающее
//     окно при -min-trades 20 — точка выбираема. На 3.5 остаётся 69 (23 на окно) и PF падает до
//     1.473. На SIBN та же ось обрывалась на 2.5, потому что там PF падал монотонно и 3.0 давало
//     45 сделок.
//   - Верхний край дневного гейта ОСТАВЛЕН на 1.5, хотя на SIBN его сдвигали на 1.3: там ветка
//     «день исчерпан» при 1.5 отбирала 5.3% баров, здесь — 8.7%, больше, чем на SVAV (7.7%), где
//     край держался. Ось уплотнена живыми уровнями 0.7 (48.9%) и 1.1 (20.8%).
//   - Строка StopDailyATR = 0.3 ВЫРЕЗАНА — второй такой случай в каталоге после domrf/. Шаг цены
//     на ELFV равен 0.0002 (0.039% медианной цены 0.5134), бэктест наливает по закрытию бара и
//     спред не моделирует, поэтому реальный круг издержек ближе к 0.2%, а не к 0.1%. При стопе
//     0.3 ATR (0.89% цены) это 22.4% риска — за чертой 17%, по которой строку вырезали из domrf/.
//     Замер чувствительности: круг 0.1% -> PF 2.003, 0.2% -> 1.668, 0.3% -> 1.381.
func TestELFVGridsPinTheirMeasuredAxes(t *testing.T) {
	screen := elfvGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; !sameSet(got, 0, 1) {
			t.Errorf("cal_screen.json: %s = %v, want ровно {0,1} — тема меряет цену каждого гейта в сделках", field, got)
		}
	}

	entry := elfvGrid(t, "cal_entry.json")
	// Выше 50 отката нет по определению: 50 — средняя линия осциллятора.
	for _, v := range entry["RSILower"] {
		if v > 50 {
			t.Errorf("cal_entry.json свипует RSILower=%v: выше 50 отката нет по определению", v)
		}
	}
	// Уровень 10 обязан остаться: на ELFV он живой — RSI(4)@10 даёт 250 будних кроссов за 36
	// месяцев, слабейший угол сетки RSI(7)@10 — 34, выше планки живого угла 29.
	if !containsValue(entry["RSILower"], 10) {
		t.Errorf("cal_entry.json: RSILower = %v, не содержит 10 — на ELFV это живой угол (250 кроссов RSI(4)@10)", entry["RSILower"])
	}
	// RSIPeriod короче 4 покупает шум, а не откат: RSI(3)@10 даёт 555 кроссов против 250 у RSI(4).
	for _, v := range entry["RSIPeriod"] {
		if v < 4 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: короче 4 это шум, а не откат", v)
		}
		// Верхний край 7: RSI(8)@10 даёт 23 кросса за 36 месяцев — ниже планки живого угла 29,
		// по которой угол объявляли мёртвым на LSNGP и RENI. На SIBN тот же край держался на 8,
		// потому что там RSI(8)@10 давал 34.
		if v > 7 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: слабейший угол RSI(8)@10 даёт 23 кросса, ось мертва за 7", v)
		}
	}
	// Сужение оси против образца sibn/ — сознательное, поэтому край прибит явно.
	if !containsValue(entry["RSIPeriod"], 7) {
		t.Errorf("cal_entry.json: RSIPeriod = %v, не содержит 7 — это последний живой угол (34 кросса RSI(7)@10)", entry["RSIPeriod"])
	}

	trend := elfvGrid(t, "cal_trend.json")
	// Ось тренда режется только вместе с этим тестом. На ELFV доля баров с EMAFast > EMASlow
	// укладывается в 44.0-46.5% на ВСЕХ 35 парах: выбор пары меняет не объём допуска, а то,
	// какие именно бары в него попадают. Значит, ни одна пара не мертва по выборке, и сужать
	// сетку не за что — а разница PF между парами читается как качество фильтра.
	if !containsValue(trend["EMASlow"], 200) {
		t.Errorf("cal_trend.json: EMASlow = %v, не содержит 200 — медленный край оси обязан остаться, допуск у него 44.0-44.6%%", trend["EMASlow"])
	}
	for _, fast := range trend["EMAFast"] {
		for _, slow := range trend["EMASlow"] {
			if fast >= slow {
				t.Errorf("cal_trend.json: пара EMAFast=%v EMASlow=%v вырождена — быстрая обязана быть быстрее медленной", fast, slow)
			}
		}
	}

	risk := elfvGrid(t, "cal_risk.json")
	// Строка 0.3 вырезана по издержкам, а не по выживаемости: уровня достаёт 98.8% дней, то есть
	// как стоп он жив. Убивает его шаг цены 0.0002 — см. доку теста.
	for _, v := range risk["StopDailyATR"] {
		if v < 0.5 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: при шаге цены 0.0002 реальный круг издержек съедает там больше 22%% риска", v)
		}
		// Верхний край 1.3: уровня 1.5 ATR достаёт лишь 13.2% дней, такой стоп перестаёт быть
		// защитой и становится способом вытеснить убыток в RSI-выход.
		if v > 1.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: до него доходит 13%% дней — это не стоп", v)
		}
	}
	if !containsValue(risk["StopDailyATR"], 0.5) {
		t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит 0.5 — это узкий край оси после выреза 0.3", risk["StopDailyATR"])
	}

	day := elfvGrid(t, "cal_day.json")
	// Ветка «свежий день» ловит 10.0% будних баров при пороге 0.3 и 29.0% при 0.5; ноль в оси —
	// это выключенная ветка, и она обязана остаться, потому что на всех прод-тикерах каталога
	// победил именно ноль.
	if !containsValue(day["FreshDayATR"], 0) {
		t.Errorf("cal_day.json: FreshDayATR = %v, не содержит 0 — выключенная ветка обязана быть в сетке", day["FreshDayATR"])
	}
	// Верхний край ветки «день исчерпан» ОСТАВЛЕН на 1.5: через него проходит 8.7% баров —
	// больше, чем на SVAV (7.7%), где край держался, и заметно больше, чем на SIBN (5.3%), где
	// его срезали до 1.3.
	if !containsValue(day["SpentDayATR"], 1.5) {
		t.Errorf("cal_day.json: SpentDayATR = %v, не содержит 1.5 — на ELFV край живой (8.7%% баров)", day["SpentDayATR"])
	}

	daySpent := elfvGrid(t, "cal_day_spent.json")
	if !containsValue(daySpent["SpentDayATR"], 1.5) {
		t.Errorf("cal_day_spent.json: SpentDayATR = %v, не содержит 1.5 — край живой по замеру 8.7%% баров", daySpent["SpentDayATR"])
	}
	// Уплотнение оси живыми уровнями — часть решения, а не случайность.
	for _, want := range []float64{0.7, 1.1} {
		if !containsValue(daySpent["SpentDayATR"], want) {
			t.Errorf("cal_day_spent.json: SpentDayATR = %v, не содержит %v — уровень живой (48.9%% и 20.8%% баров)", daySpent["SpentDayATR"], want)
		}
	}

	volume := elfvGrid(t, "cal_volume.json")
	// Край 3.0 — расширение против sibn/, где ось обрывалась на 2.5. Держится на счёте сделок:
	// 76 за 36 месяцев это 25 на обучающее окно при -min-trades 20.
	if !containsValue(volume["VolMult"], 3.0) {
		t.Errorf("cal_volume.json: VolMult = %v, не содержит 3.0 — ось расширена по замеру (76 сделок, 25 на train)", volume["VolMult"])
	}
	for _, v := range volume["VolMult"] {
		if v > 3.0 {
			t.Errorf("cal_volume.json свипует VolMult=%v: при 3.5 остаётся 69 сделок (23 на train) и PF падает до 1.473", v)
		}
	}

	// Десятая тема каталога, которой не было ни у одного тикера: VolLookbackBars везде оставался
	// дефолтом ядра. На ELFV у оси есть инструментальное основание — 18.9% будних дней несут
	// меньше двадцати баров (медиана 34, p10 19), поэтому «последние N будних баров» охватывают
	// здесь заметно больше календарного времени, чем на ликвидной бумаге. Точечные замеры:
	// 1 -> 68 сделок PF 1.834, 3 -> 97/2.003, 5 -> 109/1.608, 8 -> 129/1.811.
	volWindow := elfvGrid(t, "cal_vol_window.json")
	if !sameSet(volWindow["UseVolume"], 1) {
		t.Errorf("cal_vol_window.json: UseVolume = %v, want ровно {1} — тема меряет ОКНО гейта, а выключенный гейт окна не имеет", volWindow["UseVolume"])
	}
	for _, v := range volWindow["VolLookbackBars"] {
		if v < 1 {
			t.Errorf("cal_vol_window.json свипует VolLookbackBars=%v: окно меньше одного бара выключает гейт молча, а не меряет его", v)
		}
		// Верхний край 8: при медиане 34 бара в дне восемь баров это уже четверть дня, а на 19% дней
		// (p10 = 19 баров) — почти половина. Дальше окно перестаёт быть «недавним».
		if v > 8 {
			t.Errorf("cal_vol_window.json свипует VolLookbackBars=%v: на ELFV это больше четверти дня, окно перестаёт быть недавним", v)
		}
	}
	if !containsValue(volWindow["VolLookbackBars"], 3) {
		t.Errorf("cal_vol_window.json: VolLookbackBars = %v, не содержит 3 — дефолт ядра обязан остаться в сетке как опорная точка", volWindow["VolLookbackBars"])
	}

	exit := elfvGrid(t, "cal_exit.json")
	// Полоса выхода рабочая на всём диапазоне: кроссов вверх RSI(4) от 2987 (55) до 808 (80).
	for _, v := range exit["RSIUpper"] {
		if v <= 50 {
			t.Errorf("cal_exit.json свипует RSIUpper=%v: выход обязан стоять выше средней линии", v)
		}
	}
}
```

- [ ] **Step 2: Запустить тест и убедиться, что он падает**

Run: `go test ./internal/service/backtest/ -run TestELFVGridsPinTheirMeasuredAxes -v`
Expected: FAIL — каталога `data/params/rsi_pullback/elfv/` ещё нет, хелпер падает на чтении файла.

- [ ] **Step 3: Создать десять файлов сеток**

Каждый файл — объект с `_comment` и массивом `phases`. Сетки даны ниже дословно. `_comment`
пишется по образцу `sibn/` и обязан содержать четыре части: (1) что тема меряет и сколько в ней
прогонов; (2) замер из Global Constraints, из которого получена каждая ось этого файла, с прямым
указанием, почему край оси стоит там, где стоит; (3) команду запуска целиком (схема
`-months 36 -train-months 12 -test-months 6`); (4) пустое место под строку
«РЕЗУЛЬТАТ ПРОГОНА 2026-08-21: …», которую заполняют задачи 3–10.

**Жёсткое требование пакета, а не стиля:** `TestRSIPullbackCalFilesValid`
(`internal/service/backtest/rsi_pullback_grid_test.go:89`) падает, если `_comment` файла не
содержит его собственный путь вида `elfv/cal_entry.json`. Полная команда запуска с
`data/params/rsi_pullback/elfv/cal_entry.json` это условие выполняет; проверка ловит `_comment`,
скопированный у соседнего тикера без правки пути.

`cal_screen.json` — 4 прогона:
```json
{"phases": [{"name": "screen", "grid": {"UseDayATRGate": [0, 1], "UseVolume": [0, 1]}}]}
```

`cal_entry.json` — 168 прогонов (ось `RSIPeriod` сужена до 7):
```json
{"phases": [{"name": "entry", "grid": {
  "RSIUpper": [55, 60, 65, 70, 75, 80],
  "RSIPeriod": [4, 5, 6, 7],
  "RSILower": [10, 15, 20, 25, 30, 35, 40]
}}]}
```

`cal_trend.json` — 35 прогонов:
```json
{"phases": [{"name": "trend", "grid": {
  "EMAFast": [5, 10, 20, 30, 40],
  "EMASlow": [50, 70, 100, 120, 150, 170, 200]
}}]}
```

`cal_day.json` — 24 прогона (верхний край 1.5 сохранён):
```json
{"phases": [{"name": "day", "grid": {
  "UseDayATRGate": [1],
  "FreshDayATR": [0, 0.3, 0.4, 0.5],
  "SpentDayATR": [0.6, 0.8, 0.9, 1.0, 1.25, 1.5]
}}]}
```

`cal_day_spent.json` — 8 прогонов (0.7 и 1.1 добавлены):
```json
{"phases": [{"name": "day_spent", "grid": {
  "UseDayATRGate": [1],
  "FreshDayATR": [0],
  "SpentDayATR": [0.6, 0.7, 0.8, 0.9, 1.0, 1.1, 1.25, 1.5]
}}]}
```

`cal_volume.json` — 24 прогона (ось расширена до 3.0):
```json
{"phases": [{"name": "volume", "grid": {
  "UseVolume": [1],
  "VolMult": [1.0, 1.2, 1.5, 2.0, 2.5, 3.0],
  "VolBaseDays": [3, 5, 10, 14]
}}]}
```

`cal_vol_window.json` — 15 прогонов (десятая тема, новая для каталога):
```json
{"phases": [{"name": "vol_window", "grid": {
  "UseVolume": [1],
  "VolLookbackBars": [1, 2, 3, 5, 8],
  "VolMult": [1.5, 2.0, 2.5]
}}]}
```

`cal_risk.json` — 28 прогонов (строка 0.3 вырезана):
```json
{"phases": [{"name": "risk", "grid": {
  "StopDailyATR": [0.5, 0.7, 1.0, 1.3],
  "TPDailyATR": [0.5, 0.6, 0.8, 1.0, 1.5, 2.0, 2.5]
}}]}
```

`cal_exit.json` — 6 прогонов:
```json
{"phases": [{"name": "exit", "grid": {"RSIUpper": [55, 60, 65, 70, 75, 80]}}]}
```

`cal_trail.json` — 12 прогонов:
```json
{"phases": [{"name": "trail", "grid": {
  "UseRSIExit": [0, 1],
  "UseTrail": [1],
  "TrailDailyATR": [0, 0.3, 0.5, 0.7, 1.0, 1.3]
}}]}
```

- [ ] **Step 4: Запустить тесты пакета целиком**

Run: `go test ./internal/service/backtest/ -run 'RSIPullback|ELFV' -v`
Expected: PASS, включая общий `TestRSIPullbackGridControlPoints`
(`rsi_pullback_grid_test.go:147`) — он обходит каталог рекурсивно и требует, чтобы файл, свипующий
`StopDailyATR`, свипевал и цель шире самого широкого стопа (`cal_risk.json` это выполняет:
2.5 > 1.3). `cal_vol_window.json` стопа не свипует, поэтому под эту проверку не подпадает.

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/elfv internal/service/backtest/rsi_pullback_elfv_grid_test.go
git commit -m "feat(rsi_pullback): сетки ELFV с замеренными осями и новой темой окна объёма"
```

---

### Task 2: Пакет `strategy/elfv` в состоянии «калибровка не проводилась»

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/elfv/elfv.go`
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/elfv/elfv_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry.go` (импорт + запись в карту)
- Modify: `internal/service/backtest/rsi_pullback_registry_test.go` (сторожевой тест baseline)

**Interfaces:**
- Consumes: `core.Params`, `core.DefaultParams()` из
  `internal/service/trading_strategy/rsi_pullback/strategy/core`.
- Produces: `elfv.Ticker` (константа `"ELFV"`) и `elfv.DefaultParams() core.Params` — их используют
  Task 12 (литерал), Task 13 (реестр живого раннера) и Task 14 (вселенная).

- [ ] **Step 1: Написать падающий тест baseline-состояния**

Создать `internal/service/trading_strategy/rsi_pullback/strategy/elfv/elfv_test.go`:

```go
package elfv

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// Пакет заведён 2026-08-21 ДО калибровки: он обязан отслеживать baseline, чтобы правка дефолтов
// доходила до тикера, а не расходилась с ним молча. Тест держит это состояние и подлежит замене
// снимком литерала ровно тогда, когда калибровка закончится (задача 12 плана).
func TestParamsTrackTheBaseline(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("ELFV ещё не откалиброван, параметры обязаны совпадать с baseline:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsELFV(t *testing.T) {
	if Ticker != "ELFV" {
		t.Fatalf("Ticker = %q, want ELFV", Ticker)
	}
}
```

- [ ] **Step 2: Запустить тест и убедиться, что он падает**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/elfv/ -v`
Expected: FAIL — пакета нет, сборка не проходит.

- [ ] **Step 3: Создать пакет**

`internal/service/trading_strategy/rsi_pullback/strategy/elfv/elfv.go`:

```go
// Package elfv supplies the ticker and rsi_pullback Params for ELFV (ЭЛ5-Энерго).
//
// СОСТОЯНИЕ: калибровка не проводилась. Пакет возвращает core.DefaultParams(), то есть
// отслеживает baseline: правка дефолтов ядра доходит до этого тикера. Ставить ELFV в боевую
// вселенную RSI_PULLBACK_TICKERS в таком состоянии нельзя — торговля пошла бы параметрами,
// которые на этом инструменте никогда не проверялись.
//
// Что известно об инструменте до прогонов (кэш 2026-08-21, 31 362 30-минутных бара, расчётное
// окно 36.0 месяца с 2023-08-21 — 31 141 бар, 23 714 будних):
//
//   - АПРИОР САМЫЙ СИЛЬНЫЙ ИЗ ВСЕХ КАНДИДАТОВ КАТАЛОГА, и это записано ДО прогонов. Скринер:
//     PFmed 1.62, плато 75%, PFmed HO 1.26 на 7 сделках. Контрольный прогон baseline на 36
//     месяцах: 181 сделка, PF 1.546, net +52 544. Для сравнения, у SIBN было 1.027 на 129
//     сделках. Вес обоих фактов ограничен: колонка PFmed HO в этом каталоге предсказывает плохо
//     в обе стороны (WUSH с 4.24 планку провалил, LSNGP с 0.99 вытянул все девять тем выше 1.5),
//     а baseline считается на полной истории без walk-forward — он показывает лишь, что дефолты
//     ядра стоят не в мёртвой зоне (прецедент NVTK). Планку сильный априор не смягчает.
//   - ЦЕНА КОПЕЕЧНАЯ, ШАГ 0.0002. В окне цена прошла 0.6698 -> 0.3390 при медиане 0.5134, и все
//     котировки кратны 0.0002 — то есть шаг равен 0.039% цены. Бэктест наливает по закрытию бара
//     и НЕ моделирует спред, поэтому реальный круг издержек ближе к 0.2%, чем к моделируемым
//     0.1%. Замер чувствительности (точка VolMult 2.0, 97 сделок): круг 0.1% -> PF 2.003,
//     0.2% -> 1.668, 0.3% -> 1.381. Живой результат будет ниже бэктестового на величину, которую
//     бэктест не показывает. Косвенный признак той же гранулярности: 3.4% будних баров имеют
//     нулевой размах (High == Low).
//   - ЛИКВИДНОСТЬ ХУДШАЯ В КАТАЛОГЕ: оборот медианой 25.1 млн ₽/день при p10 = 6.4 млн (LENT 38,
//     IVAT 43, SVAV 68, SIBN 519). Гейт отбора скринера в 50 млн ELFV проходит ТОЛЬКО средним
//     (70.2 млн), вытянутым толстым правым хвостом: p90 равен 159.8 млн, вшестеро выше медианы.
//   - ДЫРЫ В БАРАХ: медиана 34 бара в буднем дне, p10 = 19, у 147 дней из 778 (18.9%) баров
//     меньше двадцати. Отсутствующий бар — это не пропуск данных, а полчаса без единой сделки.
//     Прямое следствие: ручка VolLookbackBars, задающая окно «последние N будних баров», меряет
//     на этой бумаге не только время, но и ликвидность. Под неё заведена десятая тема каталога
//     cal_vol_window.json — первая в каталоге, свипующая эту ось.
//   - ДИВИДЕНДНЫХ ГЭПОВ ПРАКТИЧЕСКИ НЕТ. За 36 месяцев ровно один гэп хуже −3% (2024-09-03,
//     −3.00%, это −0.95 дневного ATR); глубже 0.5 ATR открывается 5 дней из 764 (0.7%). ЭЛ5-Энерго
//     не платит дивиденды с 2020 года, и данные это подтверждают. Главный незакрываемый риск SIBN
//     (пять гэпов хуже −3%, худший −4.69 ATR) на этом инструменте отсутствует.
//   - ТРЕНДОВЫЙ ДОПУСК ПОЧТИ НЕ ЗАВИСИТ ОТ ПАРЫ: доля баров с EMAFast > EMASlow укладывается в
//     44.0-46.5% на всех 35 парах сетки, размах 2.5 процентных пункта. Тот же случай, что на SVAV
//     (42.4-43.6%) и SIBN (45.1-46.3%), и не такой, как на IVAT (29.4-36.0% с монотонной
//     зависимостью). Практическое следствие: выбор пары меняет не объём допуска, а то, какие
//     именно бары в него попадают, поэтому дефицит выборки из-за медленной пары здесь невозможен,
//     а разница PF между парами — это качество фильтра, а не размер выборки.
//   - РЕЖИМ САМЫЙ ЖЁСТКИЙ В КАТАЛОГЕ: всё окно −49.5%, пик-минимум −65.9%, первые 30 месяцев
//     −24.9%, холдаут −32.5%. Полугодия +2.3% / −14.1% / +0.6% / −9.8% / −4.9% / −30.5%. Это
//     одновременно достоинство — завысить лонговый результат режимом здесь нечем — и риск:
//     четвёртый OOS-фолд заведомо тяжёлый, и если планка не возьмётся именно там, отделить
//     «стратегия не работает» от «полугодие было −30.5%» на четырёх фолдах нельзя.
//   - ИСТОРИИ 36.0 МЕСЯЦА С ЗАПАСОМ В 14 ДНЕЙ, поэтому штатный протокол §8
//     docs/rsi_pullback/strategy.md (-months 36 -train-months 12 -test-months 6) исполним, и
//     числа ELFV сопоставимы построчно с остальным каталогом — в отличие от ivat (26 мес) и
//     domrf (8.8 мес).
//
// Сетки калибровки лежат в data/params/rsi_pullback/elfv/, их оси прибиты
// internal/service/backtest/rsi_pullback_elfv_grid_test.go.
package elfv

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "ELFV"

// DefaultParams returns the strategy baseline: ELFV is not calibrated yet.
func DefaultParams() core.Params {
	return core.DefaultParams()
}
```

- [ ] **Step 4: Зарегистрировать тикер в реестре бэктеста**

В `internal/service/backtest/rsi_pullback_registry.go` добавить импорт
`rsipullbackelfv "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/elfv"` и строку
карты рядом с остальными (образец — строка 56, запись SIBN):

```go
	rsipullbackelfv.Ticker:  rsiPullbackBindingFor(rsipullbackelfv.Ticker, rsipullbackelfv.DefaultParams),
```

- [ ] **Step 5: Добавить сторожевой тест baseline в реестр бэктеста**

В `internal/service/backtest/rsi_pullback_registry_test.go`:

```go
// TestRSIPullbackELFVTracksBaseline держит состояние «калибровка не проводилась»: пакет
// strategy/elfv заведён 2026-08-21 под будущий литерал, и до конца калибровки обязан возвращать
// core.DefaultParams(). Тест заменяется снимком литерала в тот день, когда литерал появится, —
// ровно так это было с reni, fesh, wush, lsngp, nvtk, ivat, svav и sibn.
func TestRSIPullbackELFVTracksBaseline(t *testing.T) {
	b, ok := rsiPullbackRegistry["ELFV"]
	if !ok {
		t.Fatal("ELFV отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("ELFV: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p != core.DefaultParams() {
		t.Fatalf("ELFV отклонился от baseline до калибровки: %+v", p)
	}
	if got := b.Build(p).Ticker(); got != "ELFV" {
		t.Fatalf("Ticker() = %q, want ELFV", got)
	}
}
```

- [ ] **Step 6: Запустить тесты**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/ -run 'ELFV|RSIPullback' -v`
Expected: PASS. Тест
`internal/service/trading_strategy/rsi_pullback/live/registry_test.go:TestBaselineTrackingTickersStayOutOfTheDefaultUniverse`
обязан остаться зелёным: ELFV пока не в реестре живого раннера и не в дефолтной вселенной.

- [ ] **Step 7: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/elfv internal/service/backtest/rsi_pullback_registry.go internal/service/backtest/rsi_pullback_registry_test.go
git commit -m "feat(rsi_pullback): пакет и реестр ELFV в состоянии «калибровка не проводилась»"
```

---

### Task 3: Тема `screen` — цена двух опциональных гейтов

**Files:**
- Modify: `data/params/rsi_pullback/elfv/cal_screen.json` (дописать результат в `_comment`)

**Interfaces:**
- Consumes: каталог сеток из Task 1, пакет из Task 2.
- Produces: знание, сколько сделок стоит каждый гейт — им пользуются задачи 6, 7 и 8 при разборе.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker ELFV -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/elfv/cal_screen.json -out ./reports/ELFV_screen \
  -months 36 -train-months 12 -test-months 6 -min-trades 1 -metric profit_factor
```

- [ ] **Step 2: Прочитать отчёт**

Открыть свежий файл в `./reports/ELFV_screen/`. Выписать: pooled OOS PF и счёт сделок каждой из
четырёх комбинаций, выбор калибратора по фолдам, и во сколько сделок обходится каждый гейт.
Опорные точки для сверки из Global Constraints: baseline (оба гейта в дефолтном положении
`UseDayATRGate=1`, `UseVolume=0`) даёт 181 сделку и PF 1.546 на полной истории; тот же прогон с
включённым объёмным гейтом при `VolMult = 1.2` даёт 127 сделок и PF 1.576.

- [ ] **Step 3: Записать результат в `_comment` сетки**

Дописать строку вида `РЕЗУЛЬТАТ ПРОГОНА 2026-08-21: pooled OOS PF <...> на <...> сделках, фолды
<...>; гейт дня стоит <...> сделок, объёмный — <...>.` Числа — фактические из отчёта, без
округления в свою пользу. Отдельной фразой — результат четвёртого фолда (падающее полугодие).

- [ ] **Step 4: Коммит**

```bash
git add data/params/rsi_pullback/elfv/cal_screen.json
git commit -m "feat(rsi_pullback): ELFV, тема screen прогнана"
```

---

### Task 4: Тема `entry` — ключевая, полоса RSI на суженной оси периода

**Files:**
- Modify: `data/params/rsi_pullback/elfv/cal_entry.json`

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: первое из двух чисел планки (pooled OOS PF темы `entry`, счёт сделок, устойчивость
  `RSILower` по фолдам) — Task 12 выносит по ним вердикт.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker ELFV -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/elfv/cal_entry.json -out ./reports/ELFV_entry \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

Тема самая тяжёлая: 168 комбинаций × 4 фолда.

- [ ] **Step 2: Выписать числа планки**

Из отчёта: pooled OOS PF, счёт сделок пула, PF и счёт сделок каждого из четырёх фолдов, выбор
`RSIPeriod` / `RSILower` / `RSIUpper` по каждому фолду. Отдельно отметить вырожденные фолды (без
убыточных сделок) — планка их не засчитывает. Сравнить in-sample и OOS по фолдам: разрыв втрое и
больше означает переобучение темы, и это записывается явно (случай IVAT; на SVAV фолд 3 дал 7.893
против 0.568 на 4 сделках, и это назвали прямо).

- [ ] **Step 3: Отдельно разобрать четвёртый фолд**

Четвёртое OOS-окно (2026-02-21 … 2026-08-21) накрывает полугодие с −30.5%. Выписать его PF и счёт
сделок отдельной строкой и сказать прямо, держится ли pooled на трёх фолдах или на четырёх. Если
тема берёт планку только без четвёртого фолда — планка НЕ взята, и это записывается именно так.

- [ ] **Step 4: Проверить суженный край оси периода**

Ось обрывается на `RSIPeriod = 7` по замеру (RSI(8)@10 даёт 23 кросса при планке живого угла 29).
Если тема выбирает 7 хотя бы в одном фолде, **перепроверить выбор точечным прогоном `-params`**:
сравнить полноисторичные PF и счёт сделок для 6 и 7 при остальных полях темы. Выбор самого края
оси на десятке сделок — это шум, а не победа периода, и так и записать.

- [ ] **Step 5: Записать результат в `_comment` сетки**

Формат: `РЕЗУЛЬТАТ ПРОГОНА 2026-08-21: pooled OOS PF <...> на <...> сделках, фолды <...> — порог
1.5 <взят|не взят>. Ведущая ось RSILower выбрана <...> — устойчивость <N> из 4.` Плюс отдельные
фразы про четвёртый фолд (Step 3) и край `RSIPeriod = 7` (Step 4).

- [ ] **Step 6: Коммит**

```bash
git add data/params/rsi_pullback/elfv/cal_entry.json
git commit -m "feat(rsi_pullback): ELFV, тема entry прогнана"
```

---

### Task 5: Тема `trend` — вторая ключевая

**Files:**
- Modify: `data/params/rsi_pullback/elfv/cal_trend.json`

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: второе число планки (pooled OOS PF темы `trend`, устойчивость `EMASlow`).

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker ELFV -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/elfv/cal_trend.json -out ./reports/ELFV_trend \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Проверить счёт сделок против замера допуска**

Ключевая проверка именно этой темы на этом тикере: допуск фильтра одинаков (44.0–46.5%) на всех 35
парах, поэтому **счёт сделок фолда не должен заметно меняться от выбора `EMASlow`**. Выписать счёт
сделок по нескольким парам с разных краёв оси (5/50, 10/100, 40/200). Если счёт всё-таки заметно
разошёлся — причина в другом гейте (дневном или объёмном), и это надо записать числом: замер
допуска сам по себе тогда не объясняет тему.

- [ ] **Step 3: Записать результат в `_comment` сетки**

Тот же формат, что в Task 4, но ведущая ось — `EMASlow`. Дополнительно записать счёт сделок по
парам из Step 2 и PF четвёртого фолда отдельной строкой.

- [ ] **Step 4: Коммит**

```bash
git add data/params/rsi_pullback/elfv/cal_trend.json
git commit -m "feat(rsi_pullback): ELFV, тема trend прогнана"
```

---

### Task 6: Темы `day` и `day_spent` — дневной гейт на уплотнённой оси

**Files:**
- Modify: `data/params/rsi_pullback/elfv/cal_day.json`, `cal_day_spent.json`

**Interfaces:**
- Consumes: каталог из Task 1, результат Task 3 (цена гейта в сделках).
- Produces: значения `FreshDayATR` и `SpentDayATR` для точки Task 11.

- [ ] **Step 1: Прогнать обе темы**

```bash
go run ./cmd/backtest -ticker ELFV -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/elfv/cal_day.json -out ./reports/ELFV_day \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker ELFV -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/elfv/cal_day_spent.json -out ./reports/ELFV_day_spent \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Сверить ветку «свежий день» со своим замером**

Ветка ловит 10.0% баров при пороге 0.3, 19.6% при 0.4 и 29.0% при 0.5. Если калибратор выбирает
ненулевой `FreshDayATR`, проверить, что прирост PF не куплен обвалом качества: выписать pooled PF
и счёт сделок обоих вариантов. На всех прод-тикерах каталога победил ноль, и отклонение от этого
должно опираться на число, а не на выбор калибратора.

- [ ] **Step 3: Проверить два новых уровня оси «день исчерпан» и сохранённый край**

Уровни 0.7 (48.9% баров) и 1.1 (20.8%) добавлены на этом тикере впервые, а край 1.5 (8.7%)
сохранён вопреки решению SIBN. Если тема выбирает один из уплотнённых уровней, выписать pooled PF
и счёт сделок соседей (0.6 и 0.8 для 0.7; 1.0 и 1.25 для 1.1): выбор, который держится ровно на
одном уровне и проваливается у обоих соседей, — это пик, а не полка, и в `_comment` он должен быть
назван именно так. Если тема выбирает край 1.5, выписать счёт сделок отдельно: через ветку проходит
8.7% баров, и решение сохранить край нуждается в подтверждении счётом, а не только долей.

- [ ] **Step 4: Записать результаты в оба `_comment`**

Включая PF четвёртого фолда отдельной строкой в каждом файле.

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/elfv/cal_day.json data/params/rsi_pullback/elfv/cal_day_spent.json
git commit -m "feat(rsi_pullback): ELFV, темы дневного гейта прогнаны"
```

---

### Task 7: Тема `volume` — фон объёмов на расширенной оси

**Files:**
- Modify: `data/params/rsi_pullback/elfv/cal_volume.json`

**Interfaces:**
- Consumes: каталог из Task 1, результат Task 3.
- Produces: решение о `UseVolume`, `VolMult`, `VolBaseDays` для точки Task 11.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker ELFV -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/elfv/cal_volume.json -out ./reports/ELFV_volume \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Сверить выбор темы с точечными замерами оси**

На дефолтах у оси ВНУТРЕННИЙ МАКСИМУМ: `VolMult` 1.2 → 127 сделок PF 1.576; 2.0 → 97, PF 2.003;
2.5 → 83, PF 1.858; 3.0 → 76, PF 1.702; 3.5 → 69, PF 1.473. Это противоположность SIBN, где PF
падал монотонно и гейт работал чистой потерей выборки. Если тема выбирает значение вдали от 2.0,
**перепроверить выбор точечным прогоном `-params`** под остальными осями темы: расхождение с
замером означает, что выигрыш куплен взаимодействием с другими осями конкретного фолда, и это надо
назвать числом.

- [ ] **Step 3: Проверить расширенный край 3.0 отдельно**

`VolMult = 3.0` — расширение против образца `sibn/`, и у каталога нет опыта работы с ним. Если
тема выбирает 3.0 хотя бы в одном фолде, выписать счёт сделок этого фолда: 76 сделок за 36 месяцев
это 25 на обучающее окно, то есть точка выбираема с запасом в пять сделок над `-min-trades 20`.
Победа края на выборке ближе к порогу — это шум, и так и записать.

- [ ] **Step 4: Проверить на вырождение фолда**

На GAZP и NVTK объёмный гейт покупал pooled PF вырожденным фолдом (17.146 на 19 сделках). Если
здесь повторится — гейт отвергается, и причина записывается числом, а не мнением.

- [ ] **Step 5: Записать результат в `_comment`**

- [ ] **Step 6: Коммит**

```bash
git add data/params/rsi_pullback/elfv/cal_volume.json
git commit -m "feat(rsi_pullback): ELFV, тема volume прогнана"
```

---

### Task 8: Тема `vol_window` — окно объёмного гейта, десятая тема каталога

**Files:**
- Modify: `data/params/rsi_pullback/elfv/cal_vol_window.json`

**Interfaces:**
- Consumes: каталог из Task 1, результат Task 7 (выбор множителя).
- Produces: значение `VolLookbackBars` для точки Task 11 — поле, которое во всех тринадцати
  предыдущих пакетах оставалось дефолтом ядра.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker ELFV -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/elfv/cal_vol_window.json -out ./reports/ELFV_vol_window \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Сверить выбор темы с точечными замерами оси**

Опорные замеры (`VolMult = 2.0`, `VolBaseDays = 14`, полная история): `VolLookbackBars` 1 → 68
сделок PF 1.834; 2 → 84, PF 1.752; 3 → 97, PF 2.003; 5 → 109, PF 1.608; 8 → 129, PF 1.811.
Зависимость немонотонная, поэтому лидерборду тут верить нельзя: **любой выбор темы, отличный от 3,
перепроверяется точечным прогоном `-params`** под остальными осями. Выписать pooled PF и счёт
сделок обоих вариантов — выбранного темой и дефолтного 3.

- [ ] **Step 3: Проверить, не куплен ли выбор счётом сделок**

Ось меняет счёт сделок почти вдвое (68 → 129), а `-min-trades 20` наказывает узкие окна. Если тема
выбирает 5 или 8 — проверить, не в том ли дело, что широкое окно просто набирает сделки, а не
отбирает лучшие: сравнить PF и expectancy, а не только PF. Если тема выбирает 1 или 2 — проверить
обратное: не ушли ли комбинации под порог `-min-trades` по процедурной причине. И то и другое
записывается прямо, а не выдаётся за вывод об окне.

- [ ] **Step 4: Записать результат в `_comment`**

Кроме pooled PF, фолдов и устойчивости `VolLookbackBars` записать прямой ответ на вопрос, ради
которого тема заведена: **меняет ли окно результат настолько, чтобы его вообще стоило свипевать**,
или дефолт ядра 3 остаётся лучшим. Ответ пригодится каталогу вне зависимости от знака — это первый
замер этой оси.

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/elfv/cal_vol_window.json
git commit -m "feat(rsi_pullback): ELFV, тема окна объёмного гейта прогнана"
```

---

### Task 9: Тема `risk` — стоп и цель на оси без строки 0.3

**Files:**
- Modify: `data/params/rsi_pullback/elfv/cal_risk.json`

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: `StopDailyATR` и `TPDailyATR` для точки Task 11.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker ELFV -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/elfv/cal_risk.json -out ./reports/ELFV_risk \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Проверить ось стопа на вытеснение убытков**

Капкан, разобранный на WUSH, LENT, LSNGP, IVAT, SVAV и SIBN: profit factor растёт с шириной стопа,
а доля выходов по стопу падает — это вытеснение убытка в RSI-выход, а не улучшение защиты. Признак,
который надо искать в первую очередь: **счёт сделок не меняется на всей оси стопа**. **Выписать
долю стоп-выходов для каждой из четырёх точек оси, а не только PF.** Опорный замер выживаемости на
ELFV: уровня 0.5 ATR достаёт 88.5% дней, 0.7 — 66.4%, 1.0 — 36.3%, 1.3 — 20.2%.

- [ ] **Step 3: Проверить узкий край 0.5 на издержки**

Строка 0.3 из сетки вырезана, поэтому 0.5 — теперь самый узкий стоп оси, и на нём проверка
издержек делается явно. Стоп 0.5 ATR это 1.49% цены; моделируемый круг 0.1% съедает 6.7% риска, а
реальный со спредом (шаг 0.0002 = 0.039% цены) — около 13.4%. Если калибратор выбирает 0.5,
пересчитать чистый результат этой точки при `-commission 0.001` (круг 0.2%) и записать обе цифры:
если разница в PF больше 20%, выбор куплен издержками, которых живой счёт не увидит в свою пользу.

- [ ] **Step 4: Записать результат в `_comment`**

Кроме pooled PF и фолдов записать таблицу «StopDailyATR → доля стоп-выходов → счёт сделок» и обе
цифры из Step 3.

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/elfv/cal_risk.json
git commit -m "feat(rsi_pullback): ELFV, тема risk прогнана"
```

---

### Task 10: Темы `exit` и `trail` — выходы

**Files:**
- Modify: `data/params/rsi_pullback/elfv/cal_exit.json`, `cal_trail.json`

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: `RSIUpper`, `UseRSIExit`, `UseTrail`, `TrailDailyATR` для точки Task 11.

- [ ] **Step 1: Прогнать обе темы**

```bash
go run ./cmd/backtest -ticker ELFV -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/elfv/cal_exit.json -out ./reports/ELFV_exit \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker ELFV -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/elfv/cal_trail.json -out ./reports/ELFV_trail \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Проверить характер сделки при выбранной полосе**

На LSNGP `RSIUpper` 55 уронил медиану удержания до 4 баров, на SVAV — до 5 при доле многодневных
сделок 14.6%: многодневная стратегия стала внутридневной. Если тема выбирает низкую полосу,
выписать медиану удержания и долю сделок длиннее одного торгового дня — это плата, которую надо
назвать явно.

На ELFV у короткого удержания есть своя цена, обратная случаю SIBN: там оно снижало экспозицию к
дивидендным гэпам, здесь гэпов практически нет (один хуже −3% за 36 месяцев), зато **каждая сделка
платит спред**, а шаг цены равен 0.039% цены. Больше сделок на том же движении — больше кругов
издержек. Если низкая полоса победит, пересчитать её результат при `-commission 0.001` и записать
обе цифры.

- [ ] **Step 3: Учесть структурный перекос темы `trail`**

`-min-trades 20` структурно топит ветку `UseRSIExit=0`: без RSI-выхода удержание длиннее, сделок
меньше, открытая позиция блокирует входы. Если все строки с `UseRSIExit=0` ушли под порог, это
процедурная причина, а не вывод о трейле — записать это в `_comment` прямо, а не выдавать за
результат. Проверяется просто: выписать счёт сделок ветки `UseRSIExit=0` на полной истории точечным
прогоном `-params` и поделить на три (число 12-месячных обучающих окон).

- [ ] **Step 4: Записать результаты в оба `_comment`**

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/elfv/cal_exit.json data/params/rsi_pullback/elfv/cal_trail.json
git commit -m "feat(rsi_pullback): ELFV, темы выходов прогнаны"
```

---

### Task 11: Сборка точки и точечный walk-forward

**Files:**
- Create: `data/params/rsi_pullback/elfv/plateau_point.json`

**Interfaces:**
- Consumes: результаты задач 3–10.
- Produces: конкретный набор из восемнадцати полей `core.Params` и его замеры — их прибивает
  Task 12.

- [ ] **Step 1: Собрать точку из лидербордов тем**

Взять по каждой теме её выбор. Где тема мерила ось поверх дефолтов, стоящих вне рабочей зоны
инструмента (случай NVTK и SIBN), — проверить ось точечными прогонами `-params` и записать, что
выбор расходится с темой и почему. На SVAV таких расхождений было три из восьми осей, на SIBN — три
из девяти; это норма, а не сбой.

Наивная сборка «по победителям тем без проверки» ломается предсказуемо: на SVAV она дала 23 сделки
за 36 месяцев при PF 9.18, то есть шум у самой границы стоп-условия. Проверять счёт сделок
собранной точки обязательно.

**Отдельная проверка, которой не было у предшественников:** если из тем пришли одновременно
`VolMult` (Task 7) и `VolLookbackBars` (Task 8), эти два поля взаимодействуют — обе темы мерили
свою ось при дефолтном значении другой. Прогнать точечно решётку 3×3 вокруг обоих выбранных
значений и взять пару, а не два независимых победителя.

- [ ] **Step 2: Создать файл точки**

`plateau_point.json` — одна фаза `point`, каждое из восемнадцати полей задано массивом из одного
значения (формат `sibn/plateau_point.json`; массив из двух значений уронит
`TestRSIPullbackPlateauFilesArePoints`, `rsi_pullback_grid_test.go:204`). `_comment` обязан
содержать: замеры принятой точки, явную оговорку «для фиксированной точки это НЕ out-of-sample»,
как собиралась каждая ось (выбор темы или точечный прогон), и команду запуска с путём
`data/params/rsi_pullback/elfv/plateau_point.json`.

Поля, которые обязаны быть в файле: `RSIPeriod`, `RSILower`, `RSIUpper`, `EMAFast`, `EMASlow`,
`DailyATRPeriod`, `UseDayATRGate`, `FreshDayATR`, `SpentDayATR`, `StopDailyATR`, `TPDailyATR`,
`UseVolume`, `VolBaseDays`, `VolLookbackBars`, `VolMult`, `UseRSIExit`, `UseTrail`, `TrailDailyATR`.

Отдельно записать, какие поля НИ ОДНОЙ темой не свипуются и остаются дефолтом ядра, а не выбором.
На ELFV это **только `DailyATRPeriod`**: `VolLookbackBars`, который у всех тринадцати
предшественников был во второй строке этого списка, здесь измерен темой `vol_window`.

- [ ] **Step 3: Прогнать точку тем же walk-forward**

```bash
go run ./cmd/backtest -ticker ELFV -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/elfv/plateau_point.json -out ./reports/ELFV_point \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 4: Проверить стоп-условие плана**

Если pooled OOS PF < 1.0 или сделок меньше 20 — остановиться, вынести числа владельцу, задачи 12–15
не выполнять до его решения. С учётом априора (baseline PF 1.546 на 181 сделке) сработать оно не
должно, но проверка обязательна.

- [ ] **Step 5: Замерить плато соседями**

По каждой оси прогнать соседние значения точечно и выписать pooled PF и счёт сделок: плато шириной
в один шаг — это пик, а не полка, и в доке пакета это должно быть названо (случай UGLD, где
`RSILower` 20 роняет точку с 3.627 до 1.580).

- [ ] **Step 6: Проверить, что результат не держится одной неделей**

Прогнать принятую точку через `-params` на OOS-окне и разложить итог: вклад лучшей недели, лучшего
месяца, топ-1 и топ-5 сделок, распределение по полугодиям, число убыточных месяцев. На IVAT 85%
результата сделала одна неделя июля 2026, и без этой проверки pooled PF читается неверно; на SVAV и
SIBN та же проверка показала обратное (лучшая неделя 9.2% и 17.9%).

- [ ] **Step 7: Замерить точку под удвоенными издержками**

Главный незакрываемый риск этого тикера — немоделируемый спред при шаге цены 0.0002. Прогнать
принятую точку тем же walk-forward с `-commission 0.001` (круг 0.2%) и выписать pooled OOS PF и
счёт сделок рядом с дефолтными. Это не критерий приёма — планка и стоп-условие считаются по
дефолтной комиссии, как у всего каталога, — но число обязано быть записано, потому что живой
результат окажется между этими двумя.

- [ ] **Step 8: Коммит**

```bash
git add data/params/rsi_pullback/elfv/plateau_point.json
git commit -m "feat(rsi_pullback): ELFV, принятая точка и её замеры"
```

---

### Task 12: Литерал в пакете и снимок

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/elfv/elfv.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/elfv/elfv_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry_test.go` (заменить baseline-тест снимком)

**Interfaces:**
- Consumes: набор полей из Task 11.
- Produces: `elfv.DefaultParams()`, возвращающий литерал, — его читают Task 13 (реестр раннера) и
  Task 14 (вселенная).

- [ ] **Step 1: Заменить тест baseline снимком литерала**

В `elfv_test.go` удалить `TestParamsTrackTheBaseline` и написать снимок по образцу `sibn_test.go`:
`TestCalibratedLiteralIsPinned` (все восемнадцать полей), `TestParamsDoNotTrackTheBaseline`,
`TestTickerIsELFV`, `TestStopIsArmed`, `TestRSIExitIsArmed`, плюс тесты под фактически принятую
конфигурацию — `TestOnlySpentDayBranchIsArmed` / `TestVolumeGateIsArmed` / `TestTrailStaysOff`
пишутся под то, что получилось, а не копируются вслепую. Каждый тест несёт в комментарии замер,
объясняющий, почему поле именно такое.

Обязательные инварианты, которые снимок обязан сторожить независимо от результата калибровки:
`StopDailyATR > 0`, `UseRSIExit == 1` (живой раннер держит RSI-выход обязательным для всех тикеров
реестра — это проверяет `TestRegisteredTickersKeepTheRSIExitArmed`), `RSIUpper > RSILower`,
`TPDailyATR > 0`, и при `UseTrail == 0` — `TrailDailyATR == 0`.

**Дополнительный тест, специфичный для ELFV:** `TestStopStaysWideEnoughForTheSpread` — проверяет
`StopDailyATR >= 0.5`. Комментарий несёт замер: шаг цены 0.0002 = 0.039% медианной цены, реальный
круг издержек со спредом около 0.2%, и на стопе уже 0.5 ATR (1.49% цены) это 13.4% риска; строка
0.3 по этой причине вырезана из сетки, и литерал не имеет права оказаться уже неё через правку
руками.

- [ ] **Step 2: Запустить и убедиться, что падает**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/elfv/ -v`
Expected: FAIL — `DefaultParams()` ещё возвращает baseline.

- [ ] **Step 3: Поставить литерал**

В `elfv.go` заменить `return core.DefaultParams()` литералом из Task 11 и переписать доку пакета:
раздел «СОСТОЯНИЕ: калибровка не проводилась» → разбор калибровки (результат десяти тем, вердикт по
планке пункт за пунктом, разбор каждого поля литерала, граница приёма). Замеры инструмента из
прежней редакции доки сохраняются целиком — включая априор, шаг цены, ликвидность, дыры в барах и
режим, потому что они не устаревают от прогонов.

Отдельным абзацем — **что дала десятая тема**: выбрала ли она окно, отличное от дефолтного 3, и
стоит ли свипевать `VolLookbackBars` на других тикерах каталога. Это первый замер оси, и вывод
должен быть записан так, чтобы им можно было воспользоваться на следующем тикере.

- [ ] **Step 4: Заменить сторожевой тест в реестре бэктеста**

`TestRSIPullbackELFVTracksBaseline` → `TestRSIPullbackELFVIsRegisteredAndCalibrated` по образцу
теста SIBN (`rsi_pullback_registry_test.go:519`): проверяет наличие в карте, несовпадение с
baseline, равенство литералу пакета и `Ticker()`. Комментарий теста обязан назвать вердикт по
планке — взята она или нет, с числами обеих ключевых тем.

- [ ] **Step 5: Запустить тесты и линт**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/ && ./bin/golangci-lint run ./internal/service/...`
Expected: PASS, 0 issues.

- [ ] **Step 6: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/elfv internal/service/backtest/rsi_pullback_registry_test.go
git commit -m "feat(rsi_pullback): ELFV откалиброван — литерал вместо отслеживания baseline"
```

---

### Task 13: Реестр живого раннера

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/live/registry.go`

**Interfaces:**
- Consumes: `elfv.Ticker`, `elfv.DefaultParams()` из Task 12.
- Produces: `ParamsFor("ELFV")` и `StrategyFor("ELFV")`, без которых раннер тикер не увидит.

- [ ] **Step 1: Добавить импорт и запись в карту**

Импорт `"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/elfv"` рядом с остальными
и строка карты (образец — строка 137, запись SIBN):

```go
	elfv.Ticker:  elfv.DefaultParams(),
```

- [ ] **Step 2: Дописать комментарий карты**

Абзац про ELFV: штатная схема прогонов, вердикт по планке с числами обеих ключевых тем, самый
сильный априор каталога (baseline PF 1.546 на 181 сделке, holdout скринера 1.26) и чем он
закончился, шаг цены 0.0002 и немоделируемый спред как главный риск с замером чувствительности
(PF 2.003 → 1.668 → 1.381 при круге 0.1% / 0.2% / 0.3%), худшая в каталоге ликвидность (25.1 млн ₽
медианой, p10 6.4), холдаут −32.5% как самый жёсткий режимный тест каталога, и отсутствие
дивидендных гэпов как единственный риск, которого у этого тикера нет.

- [ ] **Step 3: Запустить тесты пакета**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/ -v`
Expected: PASS — включая `TestRegisteredTickersKeepTheRSIExitArmed`, который обходит всю карту, и
`TestBaselineTrackingTickersStayOutOfTheDefaultUniverse`.

- [ ] **Step 4: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/live/registry.go
git commit -m "feat(rsi_pullback): ELFV в реестре живого раннера"
```

---

### Task 14: Заведение в боевую вселенную

**Files:**
- Modify: `internal/config/rsi_pullback.go` (дефолт `Tickers` на строке 127 + комментарий)
- Modify: `internal/config/rsi_pullback_test.go:54` (ожидание дефолта)
- Modify: `env/prod.env:20`, `env/prod.env.example:30`, `env/local.env.example:27`
- Modify: `docs/rsi_pullback/live.md` (таблица §8 строка 207, раздел про реестр строка 250, §9
  порядок выката строка 293)

**Interfaces:**
- Consumes: литерал из Task 12, запись реестра из Task 13.
- Produces: боевую вселенную из четырнадцати тикеров.

- [ ] **Step 1: Проверить стоп-условие**

Если Task 11 остановился на стоп-условии — эта задача не выполняется. Иначе продолжать.

- [ ] **Step 2: Обновить тест дефолта**

```go
	want := []string{"UGLD", "T", "GAZP", "DOMRF", "FESH", "WUSH", "LENT", "RENI", "NVTK", "LSNGP", "IVAT", "SVAV", "SIBN", "ELFV"}
```

- [ ] **Step 3: Запустить тест и убедиться, что падает**

Run: `go test ./internal/config/ -run TestNewRSIPullbackConfig_Defaults -v`
Expected: FAIL — дефолт ещё из тринадцати тикеров.

- [ ] **Step 4: Обновить дефолт и env-файлы**

`Tickers: []string{"UGLD", "T", "GAZP", "DOMRF", "FESH", "WUSH", "LENT", "RENI", "NVTK", "LSNGP", "IVAT", "SVAV", "SIBN", "ELFV"}`
и та же строка в `RSI_PULLBACK_TICKERS` трёх env-файлов. В комментарий функции дописать абзац про
ELFV с типом его риска: **риск здесь не в априоре, а в исполнении** — самая тонкая ликвидность
каталога (25.1 млн ₽ медианой, p10 6.4) и шаг цены 0.0002, который бэктест не моделирует; плюс
самый жёсткий режим (холдаут −32.5%). Отсутствие дивидендных гэпов назвать явно как единственный
риск, которого у тикера нет.

- [ ] **Step 5: Обновить live.md**

Таблица §8 (дефолт переменной, строка 207), раздел про реестр («знает четырнадцать пакетов», строка
250 — добавить `elfv` в перечисление), §9 пункт 1 — добавить ELFV в список сверки `pullparity`
(строка 293).

- [ ] **Step 6: Запустить тесты**

Run: `go test ./internal/config/ ./internal/service/trading_strategy/rsi_pullback/... -v`
Expected: PASS — `TestEveryDefaultTickerIsRegistered` и
`TestBaselineTrackingTickersStayOutOfTheDefaultUniverse` читают вселенную из конфига и покроют новый
состав автоматически.

- [ ] **Step 7: Коммит**

```bash
git add internal/config env docs/rsi_pullback/live.md
git commit -m "feat(rsi_pullback): завести ELFV в боевую вселенную"
```

---

### Task 15: Документация — разбор калибровки и принятый риск

**Files:**
- Modify: `docs/rsi_pullback/strategy.md` (§8.0.1 строка каталога на строке 328 + раздел с разбором
  прогонов)
- Modify: `docs/rsi_pullback/live.md` (§10, риск 17 — следующий за риском 16 про SIBN, строка 626)

**Interfaces:**
- Consumes: числа задач 3–11, решение задачи 14.
- Produces: справочник, по которому тикер сопровождают в живой торговле.

- [ ] **Step 1: Дописать §8.0.1 `strategy.md`**

В ячейку «откалиброван» добавить `elfv` с датой, схемой прогонов (штатная), вердиктом по планке,
замерами принятой точки и ссылкой на риск 17 в `live.md`. Отдельно назвать две вещи, которых в
каталоге не было: **десятую тему** (`cal_vol_window.json`, первый свип `VolLookbackBars`) и
**самый сильный априор каталога** — это первый случай, когда тикер брали в работу с baseline
PF выше 1.5, и результат должен быть записан как прецедент в обе стороны, симметрично записи про
SIBN с его baseline 1.027.

- [ ] **Step 2: Дописать раздел с разбором прогонов**

По образцу разделов SIBN, SVAV и UGLD: рамки данных, режим, вердикт по планке пункт за пунктом,
разбор каждого поля литерала, граница приёма («для фиксированной точки это НЕ out-of-sample»).
Отдельными абзацами — три свойства, которых у соседей по каталогу не было:

1. **шаг цены и спред** — замер чувствительности из Task 11 Step 7 (pooled OOS PF при круге 0.1%
   и 0.2%), и прямой вывод, где между этими числами окажется живой результат;
2. **десятая тема** — что дал первый в каталоге свип `VolLookbackBars` и стоит ли повторять его на
   других тикерах;
3. **режим −49.5% с холдаутом −32.5%** — что именно проверено падением и почему четвёртый фолд
   читается отдельно.

- [ ] **Step 3: Дописать риск 17 в `live.md` §10**

Замеры, практические следствия для наблюдения (распределение выходов, медиана удержания, просадка),
и четыре ограничения:

1. **спред не моделируется, и он здесь дорогой**: шаг 0.0002 = 0.039% цены, круг из одного тика
   удваивает моделируемые издержки; замер PF 2.003 → 1.668 → 1.381 при круге 0.1% / 0.2% / 0.3%.
   Наблюдать фактическую цену исполнения против цены закрытия бара на первых же сделках;
2. **ликвидность худшая в каталоге**: медиана 25.1 млн ₽/день, p10 6.4 млн — каждый десятый день
   бумага оборачивает меньше семи миллионов; размер живой позиции обязан соотноситься с p10, а не
   с медианой;
3. **режим падающий, холдаут −32.5%**: четвёртый фолд заведомо тяжёлый, и слабость именно в нём
   нельзя отличить от слабости стратегии на четырёх фолдах;
4. **дыры в барах**: 18.9% будних дней несут меньше двадцати баров, 3.4% баров имеют нулевой
   размах; RSI считается по неравномерно расставленным барам, и ни один параметр этого не
   устраняет.

Плюс явная строка про риск, которого у этого тикера НЕТ: дивидендных гэпов практически нет (один
хуже −3% за 36 месяцев против пяти у SIBN), потому что ЭЛ5-Энерго не платит дивиденды. Это
единственный тикер вселенной, у которого ночное удержание не несёт риска отсечки.

- [ ] **Step 4: Проверить, что доки не противоречат числам**

Run: `grep -rn "ELFV" docs/rsi_pullback/*.md internal/service/trading_strategy/rsi_pullback/strategy/elfv/*.go internal/config/rsi_pullback.go`
Сверить каждое число с отчётами прогонов в `./reports/ELFV_*`.

- [ ] **Step 5: Коммит**

```bash
git add docs/rsi_pullback
git commit -m "docs(rsi_pullback): разбор калибровки ELFV и принятый риск"
```

---

### Task 16: Финальная проверка

**Files:** нет изменений, только проверки.

- [ ] **Step 1: Полный гейт качества**

Run: `./bin/mage ci`
Expected: lint + `go test -race ./...` + проверка дрейфа моков — зелёные.

- [ ] **Step 2: Сверка живой сборки с бэктестом**

```bash
go run ./cmd/pullparity -tickers ELFV -months 24
```
Expected: ноль расхождений. **24 месяца, а не 36:** живой сборщик тянет дневные свечи окном
`dailyFetchDays = 730` (`live/marketdata/marketdata.go:47`), и на большем горизонте появляются
ожидаемые расхождения длины `Daily*` рядов (`maxDailyHorizonMonths`, выяснено на IVAT).
Расхождение на 24 месяцах означает, что живой раннер и бэктест считают сигнал по-разному, и
заведение в прод откатывается до выяснения.

- [ ] **Step 3: Записать результат сверки в `live.md` §9**

Строка вида «ELFV заведён 2026-08-21 и сверяется за 24 месяца (`go run ./cmd/pullparity -tickers
ELFV -months 24` — <N> баров, **ноль расхождений**)» рядом с такими же строками про IVAT, SVAV и
SIBN. Коммит: `docs(rsi_pullback): сверка ELFV — 24 месяца, ноль расхождений`.

- [ ] **Step 4: Сообщить владельцу результат**

Короткий отчёт: вердикт по планке пункт за пунктом, замеры принятой точки (включая её результат под
удвоенными издержками), что заведено в прод, какие риски записаны, что осталось (первые живые
сделки, условия пересмотра). Двумя отдельными строками:

1. **чем закончилась десятая тема** — стоит ли свипевать `VolLookbackBars` на остальных тикерах
   каталога или дефолт ядра 3 подтверждён;
2. **чем закончился самый сильный априор каталога** — это симметричный ответ на вопрос, заданный
   после SIBN: если тикер с baseline PF 1.546 планку тоже не берёт, значит колонки скринера и
   контрольный baseline не предсказывают исход протокола ни снизу, ни сверху, и тратить на них
   решение о запуске калибровки бессмысленно.

---

## Self-review

**Покрытие спеки.** Априор → Global Constraints, дока пакета (Task 2, Task 12), Task 15 Step 1 и
Task 16 Step 4; рамки данных → Global Constraints; штатный протокол → Global Constraints и каждая
команда прогона; свойство 1 (шаг цены и спред) → Task 1 (тест выреза строки 0.3), Task 9 Step 3,
Task 10 Step 2, Task 11 Step 7, Task 12 Step 1 (`TestStopStaysWideEnoughForTheSpread`), риск 17;
свойство 2 (ликвидность и дыры в барах) → дока пакета Task 2, тема `vol_window` (Task 1 + Task 8),
риск 17 пункты 2 и 4; свойство 3 (нет дивидендных гэпов) → дока пакета Task 2, Task 10 Step 2,
Task 15 Step 3 (явная строка про отсутствующий риск); свойство 4 (плоская трендовая ось) →
сторожевой тест Task 1, Task 5 Step 2; свойство 5 (волатильность и издержки) → Task 1, Task 9;
свойство 6 (дневной гейт) → Task 1, Task 6; свойство 7 (объёмный фон) → Task 1, Task 7; свойство 8
(окно гейта) → Task 1, Task 8; режим → Global Constraints, Task 4 Step 3, риск 17 пункт 3; оси
десяти сеток → Task 1; сужение `RSIPeriod` до 7 → Task 1 (сетка + два сторожевых утверждения) и
Task 4 Step 4; сохранение края дневного гейта 1.5 → Task 1 и Task 6 Step 3; расширение `VolMult`
до 3.0 → Task 1 и Task 7 Step 3; вырез строки стопа 0.3 → Task 1 и Task 9 Step 3; планка →
Global Constraints, вердикт выносится в Task 12 и Task 15; правило прода и стоп-условие →
Task 11 Step 4 и Task 14 Step 1; артефакты спеки → задачи 1, 2, 12, 13, 14, 15; порядок работы
спеки → порядок задач; риск 6 спеки (у каталога нет опыта интерпретации `vol_window`) → Task 8
Step 2 и Step 3, Task 11 Step 1 (решётка 3×3).

**Плейсхолдеры.** В задачах 3–11 значения результатов прогонов не выдуманы: там, где число станет
известно только после запуска, стоит явное указание выписать его из отчёта — это данные, которых до
прогона не существует, а не плейсхолдер плана. Код сторожевого теста осей, теста baseline, доки
пакета и все десять сеток даны целиком. Снимок литерала (Task 12) задан списком обязательных
инвариантов плюс одним конкретным тестом (`TestStopStaysWideEnoughForTheSpread` с порогом 0.5),
потому что остальные его значения — результат Task 11.

**Согласованность типов.** `elfv.Ticker` (строка `"ELFV"`) и `elfv.DefaultParams() core.Params`
объявлены в Task 2 и используются под теми же именами в задачах 12, 13, 14. Хелпер `elfvGrid`
объявлен в Task 1; `containsValue`, `sameSet` и `rsiPullbackTickerGrid` берутся из существующих
файлов пакета и НЕ переобъявляются (это отдельно оговорено в Task 1 — иначе пакет не соберётся).
Имена тестов, заменяемых на следующем шаге (`TestParamsTrackTheBaseline` → снимок,
`TestRSIPullbackELFVTracksBaseline` → `TestRSIPullbackELFVIsRegisteredAndCalibrated`), названы в
обеих задачах одинаково. Имя фазы `vol_window` совпадает в сетке (Task 1 Step 3), в команде прогона
(Task 8 Step 1) и в ссылках задач 11, 12, 15.
