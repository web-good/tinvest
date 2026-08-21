# SIBN под `rsi_pullback` — план подготовки, калибровки и вердикта

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** довести SIBN (Газпром нефть) до вердикта по стратегии `rsi_pullback`: каталог сеток с
осями по замерам инструмента, пакет параметров, девять тем rolling walk-forward, литерал и решение
о боевой вселенной.

**Architecture:** тикер-специфичный каталог `data/params/rsi_pullback/sibn/` (девять однотемных
файлов + точка) + сторожевой тест осей в `internal/service/backtest` + пакет параметров
`strategy/sibn` (сначала baseline-состояние, затем литерал со снимком) + записи в двух реестрах
(бэктест и живой раннер) + заведение в боевую вселенную + разделы документации. Прогоны идут по
ШТАТНОЙ схеме каталога `-months 36 -train-months 12 -test-months 6`.

**Tech Stack:** Go 1.25, `cmd/backtest` (grid-search + rolling walk-forward), `cmd/pullparity`,
`go test`, `./bin/golangci-lint`, `./bin/mage ci`.

**Spec:** `docs/superpowers/specs/2026-08-21-sibn-rsi-pullback-prep-design.md`

## Global Constraints

- Схема прогонов — **`-months 36 -train-months 12 -test-months 6 -metric profit_factor`**,
  `-min-trades 20` (у темы `screen` — `-min-trades 1`). Это штатная схема каталога, поэтому числа
  SIBN сопоставимы построчно с UGLD, GAZP, FESH, WUSH, LENT, RENI, NVTK, LSNGP и SVAV — в отличие
  от IVAT (адаптированная 25/9/4) и DOMRF (разведочная 3 фолда 3/2).
- Кэш освежён хвостовой дозагрузкой 2026-08-21: `SIBN_Minutes30.json` — 36 735 баров, окно
  2023-08-04 … 2026-08-21; расчётное окно 36 месяцев `2023-08-21 … 2026-08-21` содержит 36 321
  бар, из них 25 609 будних. `SIBN_Day1.json` — 1 139 дневных свечей, 1 029 будних, окно с
  2022-08-04. Все замеры ниже сняты по нему.
- **Запас истории — 17 дней.** `-months 37` упрётся в начало кэша. Повторный `-refresh` во время
  калибровки НЕ запускать: он сдвинет окно вперёд, укоротит его слева и сделает часть прогонов
  несравнимой с остальными.
- Планка, объявленная ДО прогонов и не пересматриваемая после: темы `entry` и `trend` **обе** дают
  pooled OOS PF ≥ 1.5 при ≥ 20 сделках; ведущая ось темы (`RSILower` для `entry`, `EMASlow` для
  `trend`) выбрана одинаково в ≥ 3 фолдах из 4. Вырожденный фолд (ни одной убыточной сделки) не
  засчитывается в пользу тикера. Условие «≥ 5 сделок в каждом фолде» из плана IVAT здесь НЕ
  применяется — оно было костылём под 25-месячную схему.
- Правило прода, подтверждённое владельцем 2026-08-21 ДО прогонов и применяемое **как есть**:
  литерал ставится и тикер заводится в `RSI_PULLBACK_TICKERS` тринадцатым даже при непройденной
  планке. **Стоп-условие:** pooled OOS PF < 1.0 либо < 20 сделок за 36 месяцев → остановиться,
  принести числа владельцу, задачи 11–14 не выполнять. С учётом априора (см. ниже) это условие
  здесь не формальность.
- **Априор слабый, и это записано до прогонов.** Колонка `PFmed HO` скринера = 0.20 на 9 сделках —
  худшее значение первой девятки рейтинга. Контрольный baseline-прогон на 36 месяцах
  (`core.DefaultParams()`): **129 сделок, PF 1.027, net +1 473 ₽** — инструмент торгует ровно в
  ноль. Вес обоих фактов ограничен (колонка `PFmed HO` в каталоге предсказывает плохо в обе
  стороны; baseline меряет дефолты, а не рабочую зону — прецедент NVTK), но подгонять
  интерпретацию под результат после прогонов нельзя.
- Замеры инструмента, на которые ссылаются `_comment` сеток (все — будние бары расчётного окна):
  - кроссы RSI вниз на уровнях 10/15/20/25/30/35/40: RSI(4) 359/696/1145/1576/1973/2376/2709;
    RSI(5) 179/443/776/1195/1602/2001/2346; RSI(6) 99/282/565/923/1323/1739/2121;
    RSI(7) 56/182/413/725/1103/1517/1921; **RSI(8) 34/123/294/586/936/1336/1763**. Для справки
    RSI(3) 736/1226/1717/2179/2593/2969/3245 — в сетку не берётся (три бара это шум, правило
    каталога); RSI(9) 23/93/230/476/805/1204/1613 — слабейший угол 23 ниже планки живого угла 29,
    поэтому ось обрывается на 8;
  - кроссы RSI вверх (полоса выхода), RSI(4): 55 — 2876, 60 — 2677, 65 — 2307, 70 — 1796,
    75 — 1354, 80 — 963;
  - доля баров с `EMAFast > EMASlow`: **45.1–46.3% на всех 35 парах** (5/50 — 46.0%, 10/100 —
    45.5%, 20/150 — 46.0%, 40/170 — 46.3%, 30/50 — 45.1%). Размах 1.2 п.п. — допуск от выбора пары
    практически не зависит, тот же случай, что на SVAV (42.4–43.6%), и не такой, как на IVAT
    (29.4–36.0%, монотонно);
  - дневной ATR(14): медиана **2.52%** цены, p10 1.87, p90 3.58, n=1015 — САМЫЙ НИЗКИЙ в каталоге
    (LSNGP 2.77, SVAV 4.38). Круг издержек 0.1% = 0.040 ATR, то есть на стопе 0.3 ATR комиссия
    съедает **13.2%** риска (черта, по которой строку 0.3 вырезали из `domrf/`, — 17%), на 0.5 —
    7.9%, на 0.7 — 5.7%, на 1.0 — 4.0%, на 1.3 — 3.1%. В процентах цены: 0.3 ATR = 0.76%,
    0.5 = 1.26%, 0.7 = 1.76%, 1.0 = 2.52%, 1.3 = 3.28%;
  - выживаемость стопа (доля дней, чей размах достаёт уровня, n=1014): 0.3 — 98.7%, 0.5 — 90.4%,
    0.7 — 70.2%, 0.8 — 56.9%, 1.0 — 39.4%, 1.25 — 21.0%, 1.3 — 18.7%, 1.5 — 11.8%;
  - день ко второму бару: медиана **0.29 ATR**; ветка «свежий день» ловит 5.2% баров при 0.2,
    7.7% при 0.25, **11.0% при 0.3**, 14.5% при 0.35, 19.0% при 0.4, 30.1% при 0.5; ветка «день
    исчерпан» — 58.6% при 0.6, **46.9% при 0.7**, 35.6% при 0.8, 28.2% при 0.9, 21.5% при 1.0,
    **16.4% при 1.1**, 10.7% при 1.25, 9.2% при 1.3, **5.3% при 1.5**;
  - объёмный гейт (доля баров, проходящих порог) при базе 14 дней: 56.2% при 1.0, 47.4% при 1.2,
    37.1% при 1.5, 25.5% при 2.0, **19.0% при 2.5**, 14.3% при 3.0; база 10 дней — 57.9 / 49.1 /
    38.7 / 27.0 / 19.9 / 15.0%; база 5 дней — 62.7 / 54.3 / 43.4 / 31.3 / 23.0%; база 3 дня —
    66.5 / 58.6 / 48.8 / 35.6 / 27.2%;
  - объёмный гейт в СДЕЛКАХ (точечные прогоны `-params` на дефолтах, `VolBaseDays = 14`, полная
    36-месячная история): `VolMult` 1.2 → 96 сделок, PF 1.247; 2.0 → 71, PF 1.200; 2.5 → 56,
    PF 1.139; **3.0 → 45, PF 0.855**. На 12-месячное обучающее окно это 32 / 24 / 19 / 15 сделок,
    поэтому край 3.0 в ось НЕ берётся: при `-min-trades 20` он не может быть выбран никогда;
  - оборот: медиана **519 млн ₽/день**, p10 206, p90 1 317, среднее 694 (лот = 1) — лучший в
    каталоге с отрывом (SVAV 68 при p10 = 18, IVAT 43, LENT 38);
  - гэпы открытия к закрытию предыдущего буднего дня (n=764): медиана +0.04%, p5 −0.41%,
    p95 +0.86%; пять гэпов хуже −3%: 2023-12-27 −8.49% (−4.69 ATR), 2024-10-14 −6.64% (−3.13),
    2024-06-13 −5.03% (−1.61), 2026-07-06 −4.72% (−1.35), 2025-07-08 −4.51% (−2.01). Глубже
    дефолтного стопа 0.5 ATR открывается 14 дней из 764 (**1.8%**), глубже 1.0 ATR — 7 (0.9%);
  - режим: train −25.0%, holdout −5.9%, всё окно −29.5%, пик-минимум −61.0%; полугодия
    +20.9% / −18.7% / +0.0% / −18.2% / −5.6% / −6.9% — растущих два из шести;
  - контрольный baseline-прогон на 36 мес: **129 сделок, PF 1.027**, net +1 473.
- Отчёты прогонов пишутся в `./reports/SIBN_<тема>` (каталог `reports/` в `.gitignore`).
- Коммиты делать по завершении каждой задачи; сообщения на русском, в стиле существующей истории
  (`feat(rsi_pullback): ...`). Ветка — текущая `feat/sibn-pullback-prep` (отведена от
  `feat/svav-pullback-prep`, HEAD `a48b88a`).

---

### Task 1: Каталог сеток SIBN со сторожевым тестом осей

**Files:**
- Create: `data/params/rsi_pullback/sibn/cal_screen.json`, `cal_entry.json`, `cal_trend.json`,
  `cal_day.json`, `cal_day_spent.json`, `cal_volume.json`, `cal_risk.json`, `cal_exit.json`,
  `cal_trail.json`
- Test: `internal/service/backtest/rsi_pullback_sibn_grid_test.go`

**Interfaces:**
- Consumes: хелперы `rsiPullbackTickerGrid(t, ticker, file)`
  (`internal/service/backtest/rsi_pullback_grid_test.go:40`), `sameSet(got, want...)`
  (`rsi_pullback_reni_grid_test.go:17`) и `containsValue(axis, want)`
  (`rsi_pullback_ivat_grid_test.go:88`). **Все три уже объявлены в пакете — НЕ переобъявлять**,
  иначе пакет не соберётся.
- Produces: каталог `sibn/`, на который ссылаются все прогоны Task 3–10, и функцию
  `sibnGrid(t, file)`.

- [ ] **Step 1: Написать падающий тест осей**

Создать `internal/service/backtest/rsi_pullback_sibn_grid_test.go`:

```go
package backtest

import "testing"

// sibnGrid читает файл сеток SIBN через общий хелпер.
func sibnGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "sibn", file)
}

// TestSIBNGridsPinTheirMeasuredAxes сторожит оси каталога sibn/. Каталог собран 2026-08-21
// копированием формы svav/ с пересадкой каждой оси на замеры самого SIBN (36 735 30-минутных
// баров в кэше, 36 321 в расчётном окне 36.0 месяца с 2023-08-21, из них 25 609 будних). Три
// оси изменены против образца, и каждое изменение опирается на замер:
//
//   - RSIPeriod расширен до 8. Планка живого угла в каталоге — 29 кроссов (столько дал
//     слабейший угол LSNGP, объявленный мёртвым; у RENI было 23). RSI(8)@10 даёт 34 и проходит
//     её, RSI(9)@10 даёт 23 и не проходит. Направление выбрано по свойству инструмента: у SIBN
//     САМЫЙ НИЗКИЙ дневной ATR каталога (2.52% против 4.38% у SVAV), RSI колеблется мельче, и
//     медленный период правдоподобнее. Ось RSILower при этом НЕ расширяется: каталог дважды
//     получил отрицательный результат именно от неё (WUSH 2.000 -> 1.674 при растяжке до 50).
//   - Верхний край дневного гейта сдвинут с 1.5 на 1.3: ветка «день исчерпан» при 1.5 отбирает
//     лишь 5.3% баров (на SVAV было 7.7% и край держался). Взамен ось уплотнена живыми
//     уровнями 0.7 (46.9%) и 1.1 (16.4%).
//   - VolMult остановлен на 2.5, хотя по доле баров ось не исчерпана (3.0 пропускает ещё
//     14.3%). Решает счёт сделок: точечные прогоны дают 2.5 -> 56 сделок за 36 месяцев,
//     3.0 -> 45, то есть 19 и 15 на 12-месячное обучающее окно. При -min-trades 20 точка 3.0
//     не может быть выбрана никогда — мёртвая точка создавала бы видимость проверенного края.
func TestSIBNGridsPinTheirMeasuredAxes(t *testing.T) {
	screen := sibnGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; !sameSet(got, 0, 1) {
			t.Errorf("cal_screen.json: %s = %v, want ровно {0,1} — тема меряет цену каждого гейта в сделках", field, got)
		}
	}

	entry := sibnGrid(t, "cal_entry.json")
	// Выше 50 отката нет по определению: 50 — средняя линия осциллятора.
	for _, v := range entry["RSILower"] {
		if v > 50 {
			t.Errorf("cal_entry.json свипует RSILower=%v: выше 50 отката нет по определению", v)
		}
	}
	// Уровень 10 обязан остаться: на SIBN он живой — RSI(4)@10 даёт 359 будних кроссов за 36
	// месяцев, слабейший угол сетки RSI(8)@10 — 34, выше планки живого угла 29.
	if !containsValue(entry["RSILower"], 10) {
		t.Errorf("cal_entry.json: RSILower = %v, не содержит 10 — на SIBN это живой угол (359 кроссов RSI(4)@10)", entry["RSILower"])
	}
	// RSIPeriod короче 4 покупает шум, а не откат: RSI(3)@10 даёт 736 кроссов — вдвое больше
	// RSI(4), и это дыхание цены, а не откаты.
	for _, v := range entry["RSIPeriod"] {
		if v < 4 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: короче 4 это шум, а не откат", v)
		}
		// Верхний край 8: RSI(9)@10 даёт 23 кросса за 36 месяцев — ниже планки живого угла 29,
		// по которой угол объявляли мёртвым на LSNGP и RENI.
		if v > 8 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: слабейший угол RSI(9)@10 даёт 23 кросса, ось мертва за 8", v)
		}
	}
	// Расширение оси периода — сознательное и первое в каталоге, поэтому прибито явно.
	if !containsValue(entry["RSIPeriod"], 8) {
		t.Errorf("cal_entry.json: RSIPeriod = %v, не содержит 8 — на SIBN угол живой (34 кросса RSI(8)@10) и ось расширена по замеру", entry["RSIPeriod"])
	}

	trend := sibnGrid(t, "cal_trend.json")
	// Ось тренда режется только вместе с этим тестом. На SIBN доля баров с EMAFast > EMASlow
	// укладывается в 45.1-46.3% на ВСЕХ 35 парах: выбор пары меняет не объём допуска, а то,
	// какие именно бары в него попадают. Значит, ни одна пара не мертва по выборке, и сужать
	// сетку не за что — а разница PF между парами читается как качество фильтра.
	if !containsValue(trend["EMASlow"], 200) {
		t.Errorf("cal_trend.json: EMASlow = %v, не содержит 200 — медленный край оси обязан остаться, допуск у него тот же 45.2%%", trend["EMASlow"])
	}
	for _, fast := range trend["EMAFast"] {
		for _, slow := range trend["EMASlow"] {
			if fast >= slow {
				t.Errorf("cal_trend.json: пара EMAFast=%v EMASlow=%v вырождена — быстрая обязана быть быстрее медленной", fast, slow)
			}
		}
	}

	risk := sibnGrid(t, "cal_risk.json")
	// Строка 0.3 сохранена намеренно, но она ближе к черте, чем у любого другого тикера: при
	// дневном ATR 2.52% круг издержек 0.1% съедает 13.2% риска против черты 17%, по которой её
	// вырезали из domrf/. Если дефолтная комиссия когда-нибудь вырастет, эту строку пересмотреть
	// первой.
	if !containsValue(risk["StopDailyATR"], 0.3) {
		t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит 0.3 — на SIBN комиссия съедает там 13.2%% риска, строка ещё живая", risk["StopDailyATR"])
	}
	for _, v := range risk["StopDailyATR"] {
		if v <= 0 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: стоп обязан быть положительным", v)
		}
		// Верхний край 1.3: уровня 1.5 ATR достаёт лишь 11.8% дней, такой стоп перестаёт быть
		// защитой и становится способом вытеснить убыток в RSI-выход.
		if v > 1.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: до него доходит меньше 12%% дней — это не стоп", v)
		}
	}

	day := sibnGrid(t, "cal_day.json")
	// Ветка «свежий день» ловит 11.0% будних баров при пороге 0.3 и 30.1% при 0.5; ноль в оси —
	// это выключенная ветка, и она обязана остаться, потому что на всех прод-тикерах каталога
	// победил именно ноль.
	if !containsValue(day["FreshDayATR"], 0) {
		t.Errorf("cal_day.json: FreshDayATR = %v, не содержит 0 — выключенная ветка обязана быть в сетке", day["FreshDayATR"])
	}
	// Верхний край ветки «день исчерпан» сдвинут с 1.5 на 1.3: при 1.5 через ветку проходит
	// 5.3% баров — это уже не гейт, а отбор десятка баров.
	for _, v := range day["SpentDayATR"] {
		if v > 1.3 {
			t.Errorf("cal_day.json свипует SpentDayATR=%v: на SIBN порог 1.5 отбирает 5.3%% баров, край оси стоит на 1.3", v)
		}
	}

	daySpent := sibnGrid(t, "cal_day_spent.json")
	for _, v := range daySpent["SpentDayATR"] {
		if v > 1.3 {
			t.Errorf("cal_day_spent.json свипует SpentDayATR=%v: край оси стоит на 1.3 по замеру 9.2%% баров", v)
		}
	}
	// Уплотнение оси живыми уровнями — часть решения, а не случайность.
	for _, want := range []float64{0.7, 1.1} {
		if !containsValue(daySpent["SpentDayATR"], want) {
			t.Errorf("cal_day_spent.json: SpentDayATR = %v, не содержит %v — уровень живой (46.9%% и 16.4%% баров)", daySpent["SpentDayATR"], want)
		}
	}

	volume := sibnGrid(t, "cal_volume.json")
	// Край 2.5 унаследован от svav/ и остаётся; 3.0 отвергнут по счёту сделок, а не по доле
	// баров: 45 сделок за 36 месяцев это 15 на обучающее окно при -min-trades 20.
	if !containsValue(volume["VolMult"], 2.5) {
		t.Errorf("cal_volume.json: VolMult = %v, не содержит 2.5 — край оси, живой по счёту сделок (56 за 36 мес.)", volume["VolMult"])
	}
	for _, v := range volume["VolMult"] {
		if v > 2.5 {
			t.Errorf("cal_volume.json свипует VolMult=%v: при 3.0 остаётся 45 сделок (15 на train), -min-trades 20 топит точку всегда", v)
		}
	}

	exit := sibnGrid(t, "cal_exit.json")
	// Полоса выхода рабочая на всём диапазоне: кроссов вверх RSI(4) от 2876 (55) до 963 (80).
	for _, v := range exit["RSIUpper"] {
		if v <= 50 {
			t.Errorf("cal_exit.json свипует RSIUpper=%v: выход обязан стоять выше средней линии", v)
		}
	}
}
```

- [ ] **Step 2: Запустить тест и убедиться, что он падает**

Run: `go test ./internal/service/backtest/ -run TestSIBNGridsPinTheirMeasuredAxes -v`
Expected: FAIL — каталога `data/params/rsi_pullback/sibn/` ещё нет, хелпер падает на чтении файла.

- [ ] **Step 3: Создать девять файлов сеток**

Каждый файл — объект с `_comment` и массивом `phases`. Сетки даны ниже дословно. `_comment`
пишется по образцу `svav/` и обязан содержать четыре части: (1) что тема меряет и сколько в ней
прогонов; (2) замер из Global Constraints, из которого получена каждая ось этого файла, с прямым
указанием, почему край оси стоит там, где стоит; (3) команду запуска целиком (схема
`-months 36 -train-months 12 -test-months 6`); (4) пустое место под строку
«РЕЗУЛЬТАТ ПРОГОНА 2026-08-21: …», которую заполняют задачи 3–9.

**Жёсткое требование пакета, а не стиля:** `TestRSIPullbackCalFilesValid`
(`internal/service/backtest/rsi_pullback_grid_test.go:89`) падает, если `_comment` файла не
содержит его собственный путь вида `sibn/cal_entry.json`. Полная команда запуска с
`data/params/rsi_pullback/sibn/cal_entry.json` это условие выполняет; проверка ловит `_comment`,
скопированный у соседнего тикера без правки пути.

`cal_screen.json` — 4 прогона:
```json
{"phases": [{"name": "screen", "grid": {"UseDayATRGate": [0, 1], "UseVolume": [0, 1]}}]}
```

`cal_entry.json` — 210 прогонов (ось `RSIPeriod` расширена до 8):
```json
{"phases": [{"name": "entry", "grid": {
  "RSIUpper": [55, 60, 65, 70, 75, 80],
  "RSIPeriod": [4, 5, 6, 7, 8],
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

`cal_day.json` — 24 прогона (верхний край 1.3 вместо 1.5):
```json
{"phases": [{"name": "day", "grid": {
  "UseDayATRGate": [1],
  "FreshDayATR": [0, 0.3, 0.4, 0.5],
  "SpentDayATR": [0.6, 0.8, 0.9, 1.0, 1.25, 1.3]
}}]}
```

`cal_day_spent.json` — 8 прогонов (1.5 убран, 0.7 и 1.1 добавлены):
```json
{"phases": [{"name": "day_spent", "grid": {
  "UseDayATRGate": [1],
  "FreshDayATR": [0],
  "SpentDayATR": [0.6, 0.7, 0.8, 0.9, 1.0, 1.1, 1.25, 1.3]
}}]}
```

`cal_volume.json` — 20 прогонов:
```json
{"phases": [{"name": "volume", "grid": {
  "UseVolume": [1],
  "VolMult": [1.0, 1.2, 1.5, 2.0, 2.5],
  "VolBaseDays": [3, 5, 10, 14]
}}]}
```

`cal_risk.json` — 35 прогонов:
```json
{"phases": [{"name": "risk", "grid": {
  "StopDailyATR": [0.3, 0.5, 0.7, 1.0, 1.3],
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

Run: `go test ./internal/service/backtest/ -run 'RSIPullback|SIBN' -v`
Expected: PASS, включая общий `TestRSIPullbackGridControlPoints`
(`rsi_pullback_grid_test.go:147`) — он обходит каталог рекурсивно и требует, чтобы файл, свипующий
`StopDailyATR`, свипевал и цель шире самого широкого стопа (`cal_risk.json` это выполняет:
2.5 > 1.3).

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/sibn internal/service/backtest/rsi_pullback_sibn_grid_test.go
git commit -m "feat(rsi_pullback): сетки SIBN с замеренными осями"
```

---

### Task 2: Пакет `strategy/sibn` в состоянии «калибровка не проводилась»

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/sibn/sibn.go`
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/sibn/sibn_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry.go` (импорт + запись в карту)
- Modify: `internal/service/backtest/rsi_pullback_registry_test.go` (сторожевой тест baseline)

**Interfaces:**
- Consumes: `core.Params`, `core.DefaultParams()` из
  `internal/service/trading_strategy/rsi_pullback/strategy/core`.
- Produces: `sibn.Ticker` (константа `"SIBN"`) и `sibn.DefaultParams() core.Params` — их используют
  Task 11 (литерал), Task 12 (реестр живого раннера) и Task 13 (вселенная).

- [ ] **Step 1: Написать падающий тест baseline-состояния**

Создать `internal/service/trading_strategy/rsi_pullback/strategy/sibn/sibn_test.go`:

```go
package sibn

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// Пакет заведён 2026-08-21 ДО калибровки: он обязан отслеживать baseline, чтобы правка дефолтов
// доходила до тикера, а не расходилась с ним молча. Тест держит это состояние и подлежит замене
// снимком литерала ровно тогда, когда калибровка закончится (задача 11 плана).
func TestParamsTrackTheBaseline(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("SIBN ещё не откалиброван, параметры обязаны совпадать с baseline:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsSIBN(t *testing.T) {
	if Ticker != "SIBN" {
		t.Fatalf("Ticker = %q, want SIBN", Ticker)
	}
}
```

- [ ] **Step 2: Запустить тест и убедиться, что он падает**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/sibn/ -v`
Expected: FAIL — пакета нет, сборка не проходит.

- [ ] **Step 3: Создать пакет**

`internal/service/trading_strategy/rsi_pullback/strategy/sibn/sibn.go`:

```go
// Package sibn supplies the ticker and rsi_pullback Params for SIBN (Газпром нефть).
//
// СОСТОЯНИЕ: калибровка не проводилась. Пакет возвращает core.DefaultParams(), то есть
// отслеживает baseline: правка дефолтов ядра доходит до этого тикера. Ставить SIBN в боевую
// вселенную RSI_PULLBACK_TICKERS в таком состоянии нельзя — торговля пошла бы параметрами,
// которые на этом инструменте никогда не проверялись.
//
// Что известно об инструменте до прогонов (кэш 2026-08-21, 36 735 30-минутных баров, расчётное
// окно 36.0 месяца с 2023-08-21 — 36 321 бар, 25 609 будних):
//
//   - АПРИОР САМЫЙ СЛАБЫЙ В КАТАЛОГЕ, и это записано ДО прогонов. Колонка PFmed HO скринера
//     равна 0.20 на 9 сделках — худшее значение первой девятки рейтинга. Контрольный прогон
//     baseline на 36 месяцах: 129 сделок, PF 1.027, net +1 473 — инструмент торгует ровно в
//     ноль. Вес обоих фактов ограничен: колонка PFmed HO в этом каталоге предсказывает плохо в
//     обе стороны (WUSH с 4.24 планку провалил, LSNGP с 0.99 вытянул все девять тем выше 1.5),
//     а baseline меряет дефолты, а не рабочую зону инструмента (прецедент NVTK, где дефолтный
//     дневной гейт стоял там, где стратегии нет).
//   - ВОЛАТИЛЬНОСТЬ САМАЯ НИЗКАЯ В КАТАЛОГЕ: дневной ATR(14) медианой 2.52% цены (p10 1.87,
//     p90 3.58) против 2.77% у LSNGP и 4.38% у SVAV. Круг издержек 0.1% оборота стоит 0.040
//     ATR, то есть на стопе 0.3 ATR комиссия съедает 13.2% риска — ближе к черте 17%, по
//     которой строку 0.3 вырезали из domrf/, чем у любого другого тикера каталога. Обратная
//     сторона: стоп 1.3 ATR это всего 3.28% цены против 5.7% на SVAV.
//   - ЛИКВИДНОСТЬ ЛУЧШАЯ В КАТАЛОГЕ, С ОТРЫВОМ: оборот медианой 519 млн ₽/день при p10 = 206
//     млн (SVAV 68 при p10 = 18, IVAT 43, LENT 38). Даже десятый перцентиль вчетверо выше гейта
//     отбора скринера в 50 млн, поэтому стандартное ограничение бэктеста — наливает по цене
//     закрытия бара и не моделирует проскальзывание — на этом тикере перестаёт быть
//     содержательным риском.
//   - ДИВИДЕНДНЫЕ ГЭПЫ ПРОБИВАЮТ СТОП НАСКВОЗЬ. Стратегия держит позицию через ночь, а SIBN —
//     дивидендная бумага с регулярными отсечками. За окно пять гэпов вниз хуже −3%: −8.49%
//     (2023-12-27, это −4.69 дневного ATR), −6.64% (2024-10-14, −3.13), −5.03% (2024-06-13,
//     −1.61), −4.72% (2026-07-06, −1.35), −4.51% (2025-07-08, −2.01). Глубже стопа 0.5 ATR
//     1.8% дней. На таких днях стоп не пол, а пожелание: исполнение приходит на цене гэпа.
//     Низкая волатильность работает ПРОТИВ инструмента — гэп той же процентной величины на
//     SVAV стоил бы вдвое меньше ATR. Параметра против этого в стратегии нет.
//   - ТРЕНДОВЫЙ ДОПУСК НЕ ЗАВИСИТ ОТ ПАРЫ: доля баров с EMAFast > EMASlow укладывается в
//     45.1-46.3% на всех 35 парах сетки, размах 1.2 процентного пункта. Тот же случай, что на
//     SVAV (42.4-43.6%), и не такой, как на IVAT (29.4-36.0% с монотонной зависимостью).
//     Практическое следствие: выбор пары меняет не объём допуска, а то, какие именно бары в
//     него попадают, поэтому дефицит выборки из-за медленной пары здесь невозможен, а разница
//     PF между парами — это качество фильтра, а не размер выборки.
//   - РЕЖИМ падающий, но не однобокий: train −25.0%, holdout −5.9%, всё окно −29.5%,
//     пик-минимум −61.0%; из шести полугодий растущих два (+20.9% и +0.0%). Мягче, чем у SVAV
//     (одно из шести) и IVAT (ноль из пяти), — конфигурация проверится на обоих режимах. Но и
//     защиты «завысить лонговый результат режимом нечем» здесь меньше: холдаут падает всего на
//     5.9%, то есть почти нейтрален.
//   - ИСТОРИИ 36.0 МЕСЯЦА С ЗАПАСОМ В 17 ДНЕЙ, поэтому штатный протокол §8
//     docs/rsi_pullback/strategy.md (-months 36 -train-months 12 -test-months 6) исполним, и
//     числа SIBN сопоставимы построчно с остальным каталогом — в отличие от ivat (26 мес) и
//     domrf (8.8 мес).
//
// Сетки калибровки лежат в data/params/rsi_pullback/sibn/, их оси прибиты
// internal/service/backtest/rsi_pullback_sibn_grid_test.go.
package sibn

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "SIBN"

// DefaultParams returns the strategy baseline: SIBN is not calibrated yet.
func DefaultParams() core.Params {
	return core.DefaultParams()
}
```

- [ ] **Step 4: Зарегистрировать тикер в реестре бэктеста**

В `internal/service/backtest/rsi_pullback_registry.go` добавить импорт
`rsipullbacksibn "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/sibn"` и строку
карты рядом с остальными (образец — строка 55, запись SVAV):

```go
	rsipullbacksibn.Ticker:  rsiPullbackBindingFor(rsipullbacksibn.Ticker, rsipullbacksibn.DefaultParams),
```

- [ ] **Step 5: Добавить сторожевой тест baseline в реестр бэктеста**

В `internal/service/backtest/rsi_pullback_registry_test.go`:

```go
// TestRSIPullbackSIBNTracksBaseline держит состояние «калибровка не проводилась»: пакет
// strategy/sibn заведён 2026-08-21 под будущий литерал, и до конца калибровки обязан возвращать
// core.DefaultParams(). Тест заменяется снимком литерала в тот день, когда литерал появится, —
// ровно так это было с reni, fesh, wush, lsngp, nvtk, ivat и svav.
func TestRSIPullbackSIBNTracksBaseline(t *testing.T) {
	b, ok := rsiPullbackRegistry["SIBN"]
	if !ok {
		t.Fatal("SIBN отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("SIBN: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p != core.DefaultParams() {
		t.Fatalf("SIBN отклонился от baseline до калибровки: %+v", p)
	}
	if got := b.Build(p).Ticker(); got != "SIBN" {
		t.Fatalf("Ticker() = %q, want SIBN", got)
	}
}
```

- [ ] **Step 6: Запустить тесты**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/ -run 'SIBN|RSIPullback' -v`
Expected: PASS. Тест
`internal/service/trading_strategy/rsi_pullback/live/registry_test.go:TestBaselineTrackingTickersStayOutOfTheDefaultUniverse`
обязан остаться зелёным: SIBN пока не в реестре живого раннера и не в дефолтной вселенной.

- [ ] **Step 7: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/sibn internal/service/backtest/rsi_pullback_registry.go internal/service/backtest/rsi_pullback_registry_test.go
git commit -m "feat(rsi_pullback): пакет и реестр SIBN в состоянии «калибровка не проводилась»"
```

---

### Task 3: Тема `screen` — цена двух опциональных гейтов

**Files:**
- Modify: `data/params/rsi_pullback/sibn/cal_screen.json` (дописать результат в `_comment`)

**Interfaces:**
- Consumes: каталог сеток из Task 1, пакет из Task 2.
- Produces: знание, сколько сделок стоит каждый гейт — им пользуются задачи 6 и 7 при разборе.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker SIBN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sibn/cal_screen.json -out ./reports/SIBN_screen \
  -months 36 -train-months 12 -test-months 6 -min-trades 1 -metric profit_factor
```

- [ ] **Step 2: Прочитать отчёт**

Открыть свежий файл в `./reports/SIBN_screen/`. Выписать: pooled OOS PF и счёт сделок каждой из
четырёх комбинаций, выбор калибратора по фолдам, и во сколько сделок обходится каждый гейт.
Опорная точка для сверки: baseline (оба гейта в дефолтном положении `UseDayATRGate=1`,
`UseVolume=0`) даёт 129 сделок и PF 1.027 на полной истории.

- [ ] **Step 3: Записать результат в `_comment` сетки**

Дописать строку вида `РЕЗУЛЬТАТ ПРОГОНА 2026-08-21: pooled OOS PF <...> на <...> сделках, фолды
<...>; гейт дня стоит <...> сделок, объёмный — <...>.` Числа — фактические из отчёта, без
округления в свою пользу.

- [ ] **Step 4: Коммит**

```bash
git add data/params/rsi_pullback/sibn/cal_screen.json
git commit -m "feat(rsi_pullback): SIBN, тема screen прогнана"
```

---

### Task 4: Тема `entry` — ключевая, полоса RSI целиком на расширенной оси периода

**Files:**
- Modify: `data/params/rsi_pullback/sibn/cal_entry.json`

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: первое из двух чисел планки (pooled OOS PF темы `entry`, счёт сделок, устойчивость
  `RSILower` по фолдам) — Task 11 выносит по ним вердикт.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker SIBN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sibn/cal_entry.json -out ./reports/SIBN_entry \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

Тема самая тяжёлая в каталоге: 210 комбинаций × 4 фолда.

- [ ] **Step 2: Выписать числа планки**

Из отчёта: pooled OOS PF, счёт сделок пула, PF и счёт сделок каждого из четырёх фолдов, выбор
`RSIPeriod` / `RSILower` / `RSIUpper` по каждому фолду. Отдельно отметить вырожденные фолды (без
убыточных сделок) — планка их не засчитывает. Сравнить in-sample и OOS по фолдам: разрыв втрое и
больше означает переобучение темы, и это записывается явно (случай IVAT; на SVAV фолд 3 дал 7.893
против 0.568 на 4 сделках, и это назвали прямо).

- [ ] **Step 3: Отдельно разобрать расширенный край оси периода**

`RSIPeriod = 8` — первое в каталоге значение выше 7, и у каталога нет опыта работы с ним. Если
тема выбирает 8 хотя бы в одном фолде, **перепроверить выбор точечным прогоном `-params`**, а не
принимать по лидерборду: сравнить полноисторичные PF и счёт сделок для 7 и 8 при остальных полях
темы. Замер, из которого ось расширена: RSI(8)@10 даёт 34 будних кросса за 36 месяцев, RSI(9)@10 —
23 (ниже планки живого угла 29). Если 8 выигрывает на десятке сделок — это не победа периода, а
шум, и так и записать.

- [ ] **Step 4: Записать результат в `_comment` сетки**

Формат: `РЕЗУЛЬТАТ ПРОГОНА 2026-08-21: pooled OOS PF <...> на <...> сделках, фолды <...> — порог
1.5 <взят|не взят>. Ведущая ось RSILower выбрана <...> — устойчивость <N> из 4.` Плюс отдельная
фраза про край `RSIPeriod = 8` из Step 3.

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/sibn/cal_entry.json
git commit -m "feat(rsi_pullback): SIBN, тема entry прогнана"
```

---

### Task 5: Тема `trend` — вторая ключевая

**Files:**
- Modify: `data/params/rsi_pullback/sibn/cal_trend.json`

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: второе число планки (pooled OOS PF темы `trend`, устойчивость `EMASlow`).

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker SIBN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sibn/cal_trend.json -out ./reports/SIBN_trend \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Проверить счёт сделок против замера допуска**

Ключевая проверка именно этой темы на этом тикере: допуск фильтра одинаков (45.1–46.3%) на всех 35
парах, поэтому **счёт сделок фолда не должен заметно меняться от выбора `EMASlow`**. Выписать счёт
сделок по нескольким парам с разных краёв оси (5/50, 10/100, 40/200). Если счёт всё-таки заметно
разошёлся — причина в другом гейте (дневном или объёмном), и это надо записать числом: замер
допуска сам по себе тогда не объясняет тему.

- [ ] **Step 3: Записать результат в `_comment` сетки**

Тот же формат, что в Task 4, но ведущая ось — `EMASlow`. Дополнительно записать счёт сделок по
парам из Step 2.

- [ ] **Step 4: Коммит**

```bash
git add data/params/rsi_pullback/sibn/cal_trend.json
git commit -m "feat(rsi_pullback): SIBN, тема trend прогнана"
```

---

### Task 6: Темы `day` и `day_spent` — дневной гейт на уплотнённой оси

**Files:**
- Modify: `data/params/rsi_pullback/sibn/cal_day.json`, `cal_day_spent.json`

**Interfaces:**
- Consumes: каталог из Task 1, результат Task 3 (цена гейта в сделках).
- Produces: значения `FreshDayATR` и `SpentDayATR` для литерала Task 10.

- [ ] **Step 1: Прогнать обе темы**

```bash
go run ./cmd/backtest -ticker SIBN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sibn/cal_day.json -out ./reports/SIBN_day \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker SIBN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sibn/cal_day_spent.json -out ./reports/SIBN_day_spent \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Сверить ветку «свежий день» со своим замером**

Ветка ловит 11.0% баров при пороге 0.3, 19.0% при 0.4 и 30.1% при 0.5. Если калибратор выбирает
ненулевой `FreshDayATR`, проверить, что прирост PF не куплен обвалом качества: выписать pooled PF
и счёт сделок обоих вариантов. На всех прод-тикерах каталога победил ноль, и отклонение от этого
должно опираться на число, а не на выбор калибратора.

- [ ] **Step 3: Проверить два новых уровня оси «день исчерпан»**

Уровни 0.7 (46.9% баров) и 1.1 (16.4%) добавлены на этом тикере впервые. Если тема выбирает один
из них, выписать pooled PF и счёт сделок соседей (0.6 и 0.8 для 0.7; 1.0 и 1.25 для 1.1): выбор,
который держится ровно на одном уплотнённом уровне и проваливается у обоих соседей, — это пик, а
не полка, и в `_comment` он должен быть назван именно так.

- [ ] **Step 4: Записать результаты в оба `_comment`**

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/sibn/cal_day.json data/params/rsi_pullback/sibn/cal_day_spent.json
git commit -m "feat(rsi_pullback): SIBN, темы дневного гейта прогнаны"
```

---

### Task 7: Тема `volume` — фон объёмов

**Files:**
- Modify: `data/params/rsi_pullback/sibn/cal_volume.json`

**Interfaces:**
- Consumes: каталог из Task 1, результат Task 3.
- Produces: решение о `UseVolume`, `VolMult`, `VolBaseDays` для литерала Task 10.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker SIBN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sibn/cal_volume.json -out ./reports/SIBN_volume \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Сверить выбор темы с точечными замерами оси**

На дефолтах PF по оси падает МОНОТОННО с ростом множителя: `VolMult` 1.2 → 96 сделок PF 1.247;
2.0 → 71, PF 1.200; 2.5 → 56, PF 1.139; 3.0 → 45, PF 0.855. Это противоположность SVAV, где гейт
работал как отбор и имел внутренний максимум на 1.2. **Если тема выберет высокий множитель,
перепроверить выбор точечным прогоном `-params`**, а не принимать по лидерборду: расхождение с
монотонным замером означает, что выигрыш куплен взаимодействием с другими осями конкретного фолда,
и это надо назвать числом.

- [ ] **Step 3: Проверить на вырождение фолда**

На GAZP и NVTK объёмный гейт покупал pooled PF вырожденным фолдом (17.146 на 19 сделках). Если
здесь повторится — гейт отвергается, и причина записывается числом, а не мнением.

- [ ] **Step 4: Записать результат в `_comment`**

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/sibn/cal_volume.json
git commit -m "feat(rsi_pullback): SIBN, тема volume прогнана"
```

---

### Task 8: Тема `risk` — стоп и цель на самой низкой волатильности каталога

**Files:**
- Modify: `data/params/rsi_pullback/sibn/cal_risk.json`

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: `StopDailyATR` и `TPDailyATR` для литерала Task 10.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker SIBN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sibn/cal_risk.json -out ./reports/SIBN_risk \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Проверить ось стопа на вытеснение убытков**

Капкан, разобранный на WUSH, LENT, LSNGP, IVAT и SVAV: profit factor растёт с шириной стопа, а
доля выходов по стопу падает — это вытеснение убытка в RSI-выход, а не улучшение защиты. Признак,
который надо искать в первую очередь: **счёт сделок не меняется на всей оси стопа**. **Выписать
долю стоп-выходов для каждой из пяти точек оси, а не только PF.** Опорный замер выживаемости на
SIBN: уровня 0.3 ATR достаёт 98.7% дней, 0.5 — 90.4%, 0.7 — 70.2%, 1.0 — 39.4%, 1.3 — 18.7%.

- [ ] **Step 3: Проверить строку 0.3 отдельно**

На SIBN круг издержек съедает 13.2% риска при стопе 0.3 ATR — ближе к черте 17%, по которой строку
вырезали из `domrf/`, чем у любого другого тикера каталога. Если калибратор выбирает 0.3, выписать
чистый результат этой точки против 0.5: разница в PF меньше 13% означает, что выбор куплен
издержками, которые бэктест считает по фиксированной ставке 0.05% за сторону, а живой счёт может
иметь другую.

- [ ] **Step 4: Записать результат в `_comment`**

Кроме pooled PF и фолдов записать таблицу «StopDailyATR → доля стоп-выходов → счёт сделок».

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/sibn/cal_risk.json
git commit -m "feat(rsi_pullback): SIBN, тема risk прогнана"
```

---

### Task 9: Темы `exit` и `trail` — выходы

**Files:**
- Modify: `data/params/rsi_pullback/sibn/cal_exit.json`, `cal_trail.json`

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: `RSIUpper`, `UseRSIExit`, `UseTrail`, `TrailDailyATR` для литерала Task 10.

- [ ] **Step 1: Прогнать обе темы**

```bash
go run ./cmd/backtest -ticker SIBN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sibn/cal_exit.json -out ./reports/SIBN_exit \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker SIBN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sibn/cal_trail.json -out ./reports/SIBN_trail \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Проверить характер сделки при выбранной полосе**

На LSNGP `RSIUpper` 55 уронил медиану удержания до 4 баров, на SVAV — до 5 при доле многодневных
сделок 14.6%: многодневная стратегия стала внутридневной. Если тема выбирает низкую полосу,
выписать медиану удержания и долю сделок длиннее одного торгового дня — это плата, которую надо
назвать явно.

На SIBN у этой платы есть вторая сторона, которой не было у соседей: **короткое удержание снижает
экспозицию к дивидендным гэпам** (1.8% дней открываются глубже дефолтного стопа 0.5 ATR). Если
низкая полоса победит, записать и этот эффект — он работает в пользу выбора, а не против.

- [ ] **Step 3: Учесть структурный перекос темы `trail`**

`-min-trades 20` структурно топит ветку `UseRSIExit=0`: без RSI-выхода удержание длиннее, сделок
меньше, открытая позиция блокирует входы. Если все строки с `UseRSIExit=0` ушли под порог, это
процедурная причина, а не вывод о трейле — записать это в `_comment` прямо, а не выдавать за
результат.

- [ ] **Step 4: Записать результаты в оба `_comment`**

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/sibn/cal_exit.json data/params/rsi_pullback/sibn/cal_trail.json
git commit -m "feat(rsi_pullback): SIBN, темы выходов прогнаны"
```

---

### Task 10: Сборка литерала и точечный walk-forward принятой точки

**Files:**
- Create: `data/params/rsi_pullback/sibn/plateau_point.json`

**Interfaces:**
- Consumes: результаты задач 3–9.
- Produces: конкретный набор из восемнадцати полей `core.Params` и его замеры — их прибивает
  Task 11.

- [ ] **Step 1: Собрать точку из лидербордов тем**

Взять по каждой теме её выбор. Где тема мерила ось поверх дефолтов, стоящих вне рабочей зоны
инструмента (случай NVTK, где дефолтный дневной гейт стоял там, где стратегии нет), — проверить ось
точечными прогонами `-params` и записать, что выбор расходится с темой и почему. На SVAV таких
расхождений было три из восьми осей (`RSILower`, `VolMult`, `StopDailyATR`) — это норма, а не сбой.

Наивная сборка «по победителям тем без проверки» ломается предсказуемо: на SVAV она дала 23 сделки
за 36 месяцев при PF 9.18, то есть шум у самой границы стоп-условия. Проверять счёт сделок
собранной точки обязательно.

- [ ] **Step 2: Создать файл точки**

`plateau_point.json` — одна фаза `point`, каждое из восемнадцати полей задано массивом из одного
значения (формат `svav/plateau_point.json`; массив из двух значений уронит
`TestRSIPullbackPlateauFilesArePoints`, `rsi_pullback_grid_test.go:204`). `_comment` обязан
содержать: замеры принятой точки, явную оговорку «для фиксированной точки это НЕ out-of-sample»,
как собиралась каждая ось (выбор темы или точечный прогон), и команду запуска с путём
`data/params/rsi_pullback/sibn/plateau_point.json`.

Поля, которые обязаны быть в файле: `RSIPeriod`, `RSILower`, `RSIUpper`, `EMAFast`, `EMASlow`,
`DailyATRPeriod`, `UseDayATRGate`, `FreshDayATR`, `SpentDayATR`, `StopDailyATR`, `TPDailyATR`,
`UseVolume`, `VolBaseDays`, `VolLookbackBars`, `VolMult`, `UseRSIExit`, `UseTrail`, `TrailDailyATR`.

Отдельно записать, какие поля НИ ОДНОЙ темой не свипуются и остаются дефолтом ядра, а не выбором:
`DailyATRPeriod` и `VolLookbackBars`.

- [ ] **Step 3: Прогнать точку тем же walk-forward**

```bash
go run ./cmd/backtest -ticker SIBN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sibn/plateau_point.json -out ./reports/SIBN_point \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 4: Проверить стоп-условие плана**

Если pooled OOS PF < 1.0 или сделок меньше 20 — остановиться, вынести числа владельцу, задачи 11–14
не выполнять до его решения. На SIBN это условие имеет реальный шанс сработать: baseline даёт
PF 1.027, то есть точка стартует почти от единицы.

- [ ] **Step 5: Замерить плато соседями**

По каждой оси прогнать соседние значения точечно и выписать pooled PF и счёт сделок: плато шириной
в один шаг — это пик, а не полка, и в доке пакета это должно быть названо (случай UGLD, где
`RSILower` 20 роняет точку с 3.627 до 1.580).

- [ ] **Step 6: Проверить, что результат не держится одной неделей**

Прогнать принятую точку через `-params` на OOS-окне и разложить итог: вклад лучшей недели, лучшего
месяца, топ-1 и топ-5 сделок, распределение по полугодиям, число убыточных месяцев. На IVAT 85%
результата сделала одна неделя июля 2026, и без этой проверки pooled PF читается неверно; на SVAV
та же проверка показала обратное (лучшая неделя 9.2%, результат размазан по 46 неделям).

- [ ] **Step 7: Коммит**

```bash
git add data/params/rsi_pullback/sibn/plateau_point.json
git commit -m "feat(rsi_pullback): SIBN, принятая точка и её замеры"
```

---

### Task 11: Литерал в пакете и снимок

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/sibn/sibn.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/sibn/sibn_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry_test.go` (заменить baseline-тест снимком)

**Interfaces:**
- Consumes: набор полей из Task 10.
- Produces: `sibn.DefaultParams()`, возвращающий литерал, — его читают Task 12 (реестр раннера) и
  Task 13 (вселенная).

- [ ] **Step 1: Заменить тест baseline снимком литерала**

В `sibn_test.go` удалить `TestParamsTrackTheBaseline` и написать снимок по образцу `svav_test.go`:
`TestCalibratedLiteralIsPinned` (все восемнадцать полей), `TestParamsDoNotTrackTheBaseline`,
`TestStopIsArmed`, `TestRSIExitIsArmed`, плюс тесты под фактически принятую конфигурацию —
`TestOnlySpentDayBranchIsArmed` / `TestVolumeGateStaysOff` / `TestTrailStaysOff` пишутся под то, что
получилось, а не копируются вслепую. Каждый тест несёт в комментарии замер, объясняющий, почему
поле именно такое.

Обязательные инварианты, которые снимок обязан сторожить независимо от результата калибровки:
`StopDailyATR > 0`, `UseRSIExit == 1` (живой раннер держит RSI-выход обязательным для всех тикеров
реестра — это проверяет `TestRegisteredTickersKeepTheRSIExitArmed`), `RSIUpper > RSILower`,
`TPDailyATR > 0`, и при `UseTrail == 0` — `TrailDailyATR == 0`.

- [ ] **Step 2: Запустить и убедиться, что падает**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/sibn/ -v`
Expected: FAIL — `DefaultParams()` ещё возвращает baseline.

- [ ] **Step 3: Поставить литерал**

В `sibn.go` заменить `return core.DefaultParams()` литералом из Task 10 и переписать доку пакета:
раздел «СОСТОЯНИЕ: калибровка не проводилась» → разбор калибровки (результат девяти тем, вердикт по
планке пункт за пунктом, разбор каждого поля литерала, граница приёма). Замеры инструмента из
прежней редакции доки сохраняются целиком — включая априор, дивидендные гэпы и низкую
волатильность, потому что они не устаревают от прогонов.

- [ ] **Step 4: Заменить сторожевой тест в реестре бэктеста**

`TestRSIPullbackSIBNTracksBaseline` → `TestRSIPullbackSIBNIsRegisteredAndCalibrated` по образцу
теста IVAT (`rsi_pullback_registry_test.go:449`): проверяет наличие в карте, несовпадение с
baseline, равенство литералу пакета и `Ticker()`. Комментарий теста обязан назвать вердикт по
планке — взята она или нет, с числами обеих ключевых тем.

- [ ] **Step 5: Запустить тесты и линт**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/ && ./bin/golangci-lint run ./internal/service/...`
Expected: PASS, 0 issues.

- [ ] **Step 6: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/sibn internal/service/backtest/rsi_pullback_registry_test.go
git commit -m "feat(rsi_pullback): SIBN откалиброван — литерал вместо отслеживания baseline"
```

---

### Task 12: Реестр живого раннера

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/live/registry.go`

**Interfaces:**
- Consumes: `sibn.Ticker`, `sibn.DefaultParams()` из Task 11.
- Produces: `ParamsFor("SIBN")` и `StrategyFor("SIBN")`, без которых раннер тикер не увидит.

- [ ] **Step 1: Добавить импорт и запись в карту**

Импорт `"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/sibn"` рядом с остальными
и строка карты (образец — строка 110, запись SVAV):

```go
	sibn.Ticker:  sibn.DefaultParams(),
```

- [ ] **Step 2: Дописать комментарий карты**

Абзац про SIBN: штатная схема прогонов, вердикт по планке с числами обеих ключевых тем, самый
слабый априор каталога (baseline PF 1.027, holdout скринера 0.20) и чем он закончился, дневной ATR
2.52% как самый низкий в каталоге, ликвидность 519 млн ₽/день медианой как лучшая, дивидендные
гэпы как риск, который прогонами не закрывается.

- [ ] **Step 3: Запустить тесты пакета**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/ -v`
Expected: PASS — включая `TestRegisteredTickersKeepTheRSIExitArmed`, который обходит всю карту, и
`TestBaselineTrackingTickersStayOutOfTheDefaultUniverse`.

- [ ] **Step 4: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/live/registry.go
git commit -m "feat(rsi_pullback): SIBN в реестре живого раннера"
```

---

### Task 13: Заведение в боевую вселенную

**Files:**
- Modify: `internal/config/rsi_pullback.go` (дефолт `Tickers` + комментарий)
- Modify: `internal/config/rsi_pullback_test.go:54` (ожидание дефолта)
- Modify: `env/prod.env:20`, `env/prod.env.example:30`, `env/local.env.example:27`
- Modify: `docs/rsi_pullback/live.md` (таблица §8, раздел про реестр, §9 порядок выката)

**Interfaces:**
- Consumes: литерал из Task 11, запись реестра из Task 12.
- Produces: боевую вселенную из тринадцати тикеров.

- [ ] **Step 1: Проверить стоп-условие**

Если Task 10 остановился на стоп-условии — эта задача не выполняется. Иначе продолжать.

- [ ] **Step 2: Обновить тест дефолта**

```go
	want := []string{"UGLD", "T", "GAZP", "DOMRF", "FESH", "WUSH", "LENT", "RENI", "NVTK", "LSNGP", "IVAT", "SVAV", "SIBN"}
```

- [ ] **Step 3: Запустить тест и убедиться, что падает**

Run: `go test ./internal/config/ -run TestNewRSIPullbackConfig_Defaults -v`
Expected: FAIL — дефолт ещё из двенадцати тикеров.

- [ ] **Step 4: Обновить дефолт и env-файлы**

`Tickers: []string{"UGLD", "T", "GAZP", "DOMRF", "FESH", "WUSH", "LENT", "RENI", "NVTK", "LSNGP", "IVAT", "SVAV", "SIBN"}`
и та же строка в `RSI_PULLBACK_TICKERS` трёх env-файлов. В комментарий функции дописать абзац про
SIBN с типом его риска: самый слабый априор каталога, дивидендные гэпы при удержании через ночь,
низкая волатильность (стоп 0.5 ATR = 1.26% цены при `Fraction=1`) и лучшая ликвидность каталога
как единственный смягчающий фактор.

- [ ] **Step 5: Обновить live.md**

Таблица §8 (дефолт переменной), раздел про реестр («знает тринадцать пакетов»), §9 пункт 1 —
добавить SIBN в список сверки `pullparity`.

- [ ] **Step 6: Запустить тесты**

Run: `go test ./internal/config/ ./internal/service/trading_strategy/rsi_pullback/... -v`
Expected: PASS — `TestEveryDefaultTickerIsRegistered` и
`TestBaselineTrackingTickersStayOutOfTheDefaultUniverse` читают вселенную из конфига и покроют новый
состав автоматически.

- [ ] **Step 7: Коммит**

```bash
git add internal/config env docs/rsi_pullback/live.md
git commit -m "feat(rsi_pullback): завести SIBN в боевую вселенную"
```

---

### Task 14: Документация — разбор калибровки и принятый риск

**Files:**
- Modify: `docs/rsi_pullback/strategy.md` (§8.0.1 строка каталога + раздел с разбором прогонов)
- Modify: `docs/rsi_pullback/live.md` (§10, риск 16 — следующий за риском 15 про SVAV, строка 570)

**Interfaces:**
- Consumes: числа задач 3–10, решение задачи 13.
- Produces: справочник, по которому тикер сопровождают в живой торговле.

- [ ] **Step 1: Дописать §8.0.1 `strategy.md`**

В ячейку «откалиброван» добавить `sibn` с датой, схемой прогонов (штатная), вердиктом по планке,
замерами принятой точки и ссылкой на риск 16 в `live.md`. Отдельно назвать, чем закончился самый
слабый априор каталога — это первый случай, когда тикер брали в работу с baseline PF около единицы,
и результат должен быть записан как прецедент в обе стороны.

- [ ] **Step 2: Дописать раздел с разбором прогонов**

По образцу разделов SVAV, NVTK и UGLD: рамки данных, режим, вердикт по планке пункт за пунктом,
разбор каждого поля литерала, граница приёма («для фиксированной точки это НЕ out-of-sample»).
Отдельными абзацами — два свойства, которых у соседей по каталогу не было: расширенная ось
`RSIPeriod` (что дал край 8) и дивидендные гэпы (что видно по распределению убытков принятой точки:
сверить крупнейшие убыточные сделки с пятью гэп-датами из Global Constraints).

- [ ] **Step 3: Дописать риск 16 в `live.md` §10**

Замеры, практические следствия для наблюдения (распределение выходов, медиана удержания, просадка),
и четыре ограничения:

1. самый слабый априор каталога — с чем тикер входил в калибровку;
2. дивидендные гэпы: 1.8% дней открываются глубже дефолтного стопа, худший гэп −4.69 дневного ATR,
   стоп на таких днях не работает; наблюдать даты отсечек и сверять с открытыми позициями;
3. низкая волатильность: цель 0.5 ATR = 1.26% цены, круг издержек съедает 7.9% от неё;
4. мягкий холдаут (−5.9%) — доказательная нагрузка OOS-числа ниже, чем у SVAV.

Плюс явная строка, что риск исполнения у этого тикера снят замером ликвидности (p10 = 206 млн
₽/день) — единственный случай в каталоге, и это тоже нужно зафиксировать.

- [ ] **Step 4: Проверить, что доки не противоречат числам**

Run: `grep -rn "SIBN" docs/rsi_pullback/*.md internal/service/trading_strategy/rsi_pullback/strategy/sibn/*.go internal/config/rsi_pullback.go`
Сверить каждое число с отчётами прогонов в `./reports/SIBN_*`.

- [ ] **Step 5: Коммит**

```bash
git add docs/rsi_pullback
git commit -m "docs(rsi_pullback): разбор калибровки SIBN и принятый риск"
```

---

### Task 15: Финальная проверка

**Files:** нет изменений, только проверки.

- [ ] **Step 1: Полный гейт качества**

Run: `./bin/mage ci`
Expected: lint + `go test -race ./...` + проверка дрейфа моков — зелёные.

- [ ] **Step 2: Сверка живой сборки с бэктестом**

```bash
go run ./cmd/pullparity -tickers SIBN -months 24
```
Expected: ноль расхождений. **24 месяца, а не 36:** живой сборщик тянет дневные свечи окном
`dailyFetchDays = 730` (`live/marketdata/marketdata.go:47`), и на большем горизонте появляются
ожидаемые расхождения длины `Daily*` рядов (`maxDailyHorizonMonths`, выяснено на IVAT).
Расхождение на 24 месяцах означает, что живой раннер и бэктест считают сигнал по-разному, и
заведение в прод откатывается до выяснения.

- [ ] **Step 3: Сообщить владельцу результат**

Короткий отчёт: вердикт по планке пункт за пунктом, замеры принятой точки, что заведено в прод,
какие риски записаны, что осталось (первые живые сделки, условия пересмотра). Отдельной строкой —
чем закончился самый слабый априор каталога: это ответ на вопрос, стоит ли впредь тратить прогоны
на кандидатов с `PFmed HO` ниже единицы.

---

## Self-review

**Покрытие спеки.** Априор → Global Constraints, дока пакета (Task 2, Task 11), Task 14 Step 1 и
Task 15 Step 3; рамки данных → Global Constraints; штатный протокол → Global Constraints и каждая
команда прогона; свойство 1 (низкий ATR) → Task 1 (тест строки 0.3), Task 8 Step 3, риск 16;
свойство 2 (ликвидность) → дока пакета Task 2 и явная строка риска 16 Step 3; свойство 3
(дивидендные гэпы) → дока пакета Task 2, Task 9 Step 2, Task 14 Step 2 и риск 16; свойство 4
(плоская трендовая ось) → сторожевой тест Task 1, Step 2 задачи 5; режим → дока пакета и риск 16;
оси девяти сеток → Task 1; расширение `RSIPeriod` до 8 → Task 1 (сетка + два сторожевых
утверждения) и Task 4 Step 3; сдвиг края дневного гейта → Task 1 и Task 6 Step 3; отказ от
`VolMult` 3.0 → Task 1 и Task 7 Step 2; планка → Global Constraints, вердикт выносится в Task 11 и
Task 14; правило прода и стоп-условие → Task 10 Step 4 и Task 13 Step 1; артефакты 1-6 спеки →
задачи 1, 2, 11, 12, 13, 14; порядок работы спеки → порядок задач; риск 6 спеки (первое
использование `RSIPeriod = 8`) → Task 4 Step 3.

**Плейсхолдеры.** В задачах 3–10 значения результатов прогонов не выдуманы: там, где число станет
известно только после запуска, стоит явное указание выписать его из отчёта — это данные, которых до
прогона не существует, а не плейсхолдер плана. Код сторожевого теста осей, теста baseline, доки
пакета и все девять сеток даны целиком. Снимок литерала (Task 11) задан списком обязательных
инвариантов, потому что его значения — результат Task 10.

**Согласованность типов.** `sibn.Ticker` (строка `"SIBN"`) и `sibn.DefaultParams() core.Params`
объявлены в Task 2 и используются под теми же именами в задачах 11, 12, 13. Хелпер `sibnGrid`
объявлен в Task 1; `containsValue`, `sameSet` и `rsiPullbackTickerGrid` берутся из существующих
файлов пакета и НЕ переобъявляются (это отдельно оговорено в Task 1 — иначе пакет не соберётся).
Имена тестов, заменяемых на следующем шаге (`TestParamsTrackTheBaseline` → снимок,
`TestRSIPullbackSIBNTracksBaseline` → `TestRSIPullbackSIBNIsRegisteredAndCalibrated`), названы в
обеих задачах одинаково.
