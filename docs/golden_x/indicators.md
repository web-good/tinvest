# Математика индикаторов Golden X

Здесь — формулы и алгоритмы всех используемых индикаторов. Запись `$x$` означает inline-формулу, `$$x$$` — блочную (рендерится GitHub-Markdown'ом).

## 1. RSI Wilder

Источник: `rsi.go:14-60` (`computeRSISeries`, `wilderRSI`).

**Идея.** Relative Strength Index по Уайлдеру измеряет относительную силу роста к падению на скользящем окне длиной $p$ (period).

### Шаг 1. Приращения

Для последовательности цен закрытия $c_0, c_1, \dots, c_n$:

$$\Delta_i = c_i - c_{i-1}$$

$$g_i = \max(\Delta_i, 0), \quad l_i = \max(-\Delta_i, 0)$$

### Шаг 2. Seed (первое сглаженное значение)

Первые $p$ значений $g_i$ и $l_i$ усредняются простым SMA:

$$\overline{g}_p = \frac{1}{p}\sum_{i=1}^{p} g_i, \quad \overline{l}_p = \frac{1}{p}\sum_{i=1}^{p} l_i$$

### Шаг 3. Wilder smoothing (рекуррентная EMA с $\alpha = 1/p$)

Для каждого следующего значения $k > p$:

$$\overline{g}_k = \frac{\overline{g}_{k-1} \cdot (p-1) + g_k}{p}$$

$$\overline{l}_k = \frac{\overline{l}_{k-1} \cdot (p-1) + l_k}{p}$$

### Шаг 4. Сам RSI

$$RS_k = \frac{\overline{g}_k}{\overline{l}_k}$$

$$RSI_k = 100 - \frac{100}{1 + RS_k}$$

Округляется до 2 знаков после запятой (`wilderRSI` в `rsi.go:53`).

### Граничный случай

При $\overline{l}_k = 0$ деление невалидно. Чтобы не сломать пайплайн и сохранить паритет с исходным калькулятором проекта, код принудительно выставляет:

- $RS_k = 1 \implies RSI_k = 50$

Это компромисс: формально при отсутствии падений RSI должен стремиться к 100, но текущая реализация ведёт к нейтральному значению. Контракт зафиксирован в комментарии `rsi.go` и обязателен для совместимости.

### Почему Wilder, а не «обычный» RSI

Wilder использует экспоненциальное сглаживание с малым $\alpha = 1/p$, что даёт более «инерционные» значения. Это даёт меньше шума на коротких таймфреймах, но и отсроченную реакцию — отсюда и `RSILength` 7..11 для Golden X (на длинных периодах он становится слишком медленным для недельного режима).

---

## 2. Адаптивные перцентили (метод R-7)

Источник: `percentile.go:13-29` (`percentile`).

**Идея.** Зоны «перепроданности» и «перекупленности» подбираются индивидуально по историческому распределению RSI этой же акции. Это и есть «адаптивность».

### Метод R-7 (NumPy / Excel / `pandas.quantile`)

Дан отсортированный по возрастанию массив $x_0 \le x_1 \le \dots \le x_{n-1}$ и перцентиль $p \in [0, 100]$:

1. Перевод в индекс по непрерывной шкале: $h = \frac{(n-1) \cdot p}{100}$
2. Округлить вниз/вверх: $\text{lo} = \lfloor h \rfloor, \quad \text{hi} = \lceil h \rceil$
3. Линейная интерполяция:

$$\text{percentile}(x, p) = x_{\text{lo}} + (h - \text{lo}) \cdot (x_{\text{hi}} - x_{\text{lo}})$$

Если $\text{lo} = \text{hi}$ — возвращается просто $x_{\text{lo}}$.

### Применение в Golden X

`adaptiveThresholds()` и `adaptiveSellThresholds()` (`percentile.go:49, 62`):

- Берётся последний RSI-ряд длиной $[$`AdaptiveWindowMin`, `AdaptiveWindowMax`$] = [100, 200]$.
- Сортируется по возрастанию.
- Вычисляются перцентили:
  - Покупка: P5 (`BuyGreen`), P15 (`BuyYellow`).
  - Продажа: P80 (`SellYellow`), P90 (`SellOrange`), P95 (`SellRed`).

Финальное решение по tier — в `tierFromAdaptive()` и `sellTierFromAdaptive()` (см. [strategy.md](strategy.md)).

### Почему адаптивные пороги, а не 30/70

Фиксированные «магические» уровни 30/70 — это эвристика из учебника, которая работает плохо на бумагах с разной волатильностью. Пример: у акции, чей RSI исторически крутится в диапазоне 40-65, классический уровень 30 не сработает ни разу за год. Адаптивный P5 пересчитывает «локально редкое значение» и подстраивается под характер тикера.

---

## 3. EMA200 — фильтр тренда

Источник: `trend_filter.go:20-37` (`computeEMA`), `trend_filter.go:50` (`trendStatusFromClosed`).

**Идея.** Подтвердить, что покупка идёт «по тренду»: цена выше своей долгосрочной средней. Используется только в Growth.

### Seed

Первое значение EMA — обычная SMA первых $p$ значений:

$$EMA_{p-1} = \frac{1}{p}\sum_{i=0}^{p-1} c_i$$

### Рекуррентная формула

Множитель сглаживания:

$$\alpha = \frac{2}{p + 1}$$

Для $k \ge p$:

$$EMA_k = (c_k - EMA_{k-1}) \cdot \alpha + EMA_{k-1}$$

Эквивалентная запись:

$$EMA_k = \alpha \cdot c_k + (1 - \alpha) \cdot EMA_{k-1}$$

### Решение фильтра тренда

`trendStatusFromClosed()`:

| Условие | TrendStatus | Mark |
|---|---|---|
| `c_last > EMA_last` | `TrendWith` | ✅ |
| `c_last ≤ EMA_last` | `TrendAgainst` | 🚫 |
| Истории меньше `TrendEMAPeriod` | ошибка `ErrInsufficientHistory` | (сообщение не уходит) |

Для Growth-стратегии 🚫 не блокирует алерт — он всё равно показывается, но с маркером «тренд против».

---

## 4. Бычья дивергенция по фракталам Williams

Источник: `divergence.go:10-50`.

**Идея.** Найти момент, когда цена обновляет минимум, но RSI этого не подтверждает — классический признак истощения нисходящего движения.

### Фрактал Williams (k=2)

Индекс $i$ называется фрактальным минимумом порядка $k$, если

$$\text{low}_i < \text{low}_j \quad \forall j \in [i - k, i + k], \ j \ne i$$

То есть точка $i$ строго ниже всех $k$ соседей слева и $k$ соседей справа. В Golden X используется $k = 2$ (константа `divergenceFractalK` в `detector.go`) — классическое значение.

`findRecentPivotLow()` (`divergence.go:10`) ищет **самый свежий** такой минимум в окне `DivergenceLookbackWeeks` (по умолчанию 52). Граница окна симметрична: ищется в диапазоне $[k, n - 1 - k]$ — крайние индексы не могут быть фракталом, у них нет $k$ соседей справа.

### Условие бычьей дивергенции

Найден pivot с индексом $p$, последняя точка имеет индекс $\text{last}$:

$$\text{Bullish} \iff \text{low}_{\text{last}} < \text{low}_{p} \quad \text{И} \quad RSI_{\text{last}} > RSI_{p}$$

Если pivot не найден или массивы рассинхронизированы — `false`.

При выполнении → флаг `DivergenceOK = true` в `Signal` → 📈 в сообщении.

---

## 5. ATR — стоп-лосс

Источник: `pkg/indicators` (используется в `stop.go`, `detector.go:73`).

**Идея.** Стоп-лосс ставится не в фиксированных процентах, а в кратных Average True Range — статистике дневной/недельной волатильности тикера.

### True Range

Для каждой свечи $k$:

$$TR_k = \max\Big(\text{high}_k - \text{low}_k,\ |\text{high}_k - \text{close}_{k-1}|,\ |\text{low}_k - \text{close}_{k-1}|\Big)$$

Учёт `close_{k-1}` ловит гэпы между сессиями.

### ATR (Wilder smoothing)

$$ATR_k = \frac{ATR_{k-1} \cdot (p-1) + TR_k}{p}$$

Период $p = $ `ATRPeriod`, по умолчанию 14.

### Расчёт стопа

`stopFromATR(lastClose, atr, K)` (`stop.go:19`):

$$\text{StopPrice} = \text{lastClose} - K \cdot ATR$$

$$\text{DistancePct} = \frac{\text{lastClose} - \text{StopPrice}}{\text{lastClose}} \cdot 100$$

Коэффициент $K$ зависит от типа стратегии (`stop.go:9`, `kForKind`):

| Тип | $K$ | Источник |
|---|---:|---|
| Growth | 1.5 | `ATRMultiplierGrowth` |
| Dividend / Unknown | 2.0 | `ATRMultiplierDividend` |

Логика: для дивидендных бумаг (низкая волатильность, долгие удержания) — стоп шире, чтобы выдерживать шум. Для растущих (резкие движения, активный выход) — стоп уже.

### Вырожденные случаи

`stopFromATR()` возвращает пустой `Stop{}` если `atr ≤ 0`, `lastClose ≤ 0`, или `StopPrice ≤ 0`. В сообщении строка `Stop:` тогда не печатается (см. `stopLine()` в `notification/notifications.go:80`).

---

## 6. VolumeConfirmed

Источник: `pkg/indicators.VolumeConfirmed()` (используется в `detector.go:63`).

**Идея.** Подтвердить сигнал «свежим» всплеском объёма по сравнению с недавним средним.

### Формула

Дан массив объёмов $v_0, v_1, \dots, v_n$, окно SMA `VolumeSMALookback` $= L$ (по умолчанию 20), множитель `VolumeMultiplier` $= M$ (по умолчанию 1.5).

Скользящее среднее объёма предшествующих $L$ свечей:

$$\overline{v} = \frac{1}{L}\sum_{i=n-L}^{n-1} v_i$$

Подтверждение:

$$\text{VolumeConfirmed} \iff v_n > M \cdot \overline{v}$$

При `true` → флаг `VolumeOK` → 🔊 в сообщении.

### Зачем

Покупка в локальном минимуме без подскока объёма часто оказывается «ловушкой» — рынок продолжает падение. Всплеск объёма — признак того, что в зону заходят покупатели.

---

## Сводка: что во что подаётся

```
свечи (Close, High, Low, Volume)
   ↓
[Close] ──► computeRSISeries() ──► RSI-ряд ──► percentile(R-7) ──► P5/P15/P80/P90/P95
                                       │
                                       └────► tierFromAdaptive() ──► 🟢/🟡 или нет
                                                                        │
                                                                        └────►
[Close]              ──► computeEMA(200) ──► EMA200 ──► trendStatusFromClosed() ──► ✅/🚫
[Low + RSI]          ──► findRecentPivotLow() ──► bullishDivergence() ──► 📈
[Volume]             ──► VolumeConfirmed() ──► 🔊
[High,Low,Close]     ──► ATR(14) ──► stopFromATR(K) ──► Stop {Price, DistancePct}
                                                                        │
                                                                        ↓
                                                                  dto.Signal
```
