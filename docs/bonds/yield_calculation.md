# Расчёт доходности облигаций (скринер `bonds`)

Документ описывает, как стратегия `internal/service/trading_strategy/bonds` считает
итоговую прибыль и доходность облигации с **фиксированным** купоном. Вся логика
находится в `internal/service/trading_strategy/bonds/computable/calculate_profit.go`
(функция `calculateProfit`).

> Скринер обрабатывает только облигации **без амортизации**, **без плавающего купона**
> и с **НКД > 0** (фильтр `pipeline/finder.go`). Поэтому модель ниже считает, что весь
> номинал возвращается единовременно в дату погашения.

---

## 1. Входные данные

| Переменная | Источник | Смысл |
|---|---|---|
| `Nominal` | API | Номинал облигации, ₽ |
| `closePrice` | последняя дневная свеча | Котировка цены в **% от номинала** (напр. `102` = 102%) |
| `Nkd` | API (`AciValue`) | Накопленный купонный доход на дату покупки, ₽ |
| купоны | `GetBondCoupons` (с 1 января текущего года до погашения) | Список выплат |
| `MaturityDate` | API | Дата погашения |
| `CouponQuantityPerYear` | API | Количество купонов в год |

Из купонов выводятся две суммы:

- `currentYearCoupons` — сумма **всех** купонов, чья дата приходится на текущий
  календарный год (включая уже выплаченные);
- `totalFutureCoupons` — сумма купонов, дата которых **позже текущего момента**
  (то, что ещё предстоит получить).

---

## 2. Пошаговая формула

```text
bondPrice        = closePrice * Nominal / 100          # цена облигации в рублях
totalInvestment  = bondPrice + Nkd                     # сколько реально платим

# --- налоги ---
couponTax        = max( (totalFutureCoupons - Nkd) * 0.13 , 0 )   # 13% на купоны; НКД уменьшает базу
nominalPriceTax  = (Nominal - bondPrice) * 0.13   если Nominal > bondPrice, иначе 0
                                                  # налог на доход от роста к номиналу (только при дисконте)

# --- прибыль ---
totalReturn      = Nominal + totalFutureCoupons        # вернётся: номинал + будущие купоны
finalProfit      = totalReturn - totalInvestment - couponTax - nominalPriceTax

# --- приведение к году ---
daysToMaturity   = max( дней до погашения , 1 )
profitPerYear    = finalProfit * 365 / daysToMaturity  # линейный пересчёт прибыли на год
percentByYear    = 100 * profitPerYear / totalInvestment
```

Поля в уведомлении:

| Поле в Telegram | Переменная | Формула |
|---|---|---|
| **Чистая доходность/год** | `percentByYear` | `(finalProfit / totalInvestment) * (365 / daysToMaturity) * 100` |
| **Купонная доходность в год** | `couponPercentByYear` | `annualCouponIncome * 100 / totalInvestment` |
| **Прибыль/год** | `profitPerYear` | `finalProfit * 365 / daysToMaturity` |
| **Доходность к погашению** | `finalProfit` | итоговая чистая прибыль за весь срок, ₽ |

где `annualCouponIncome = currentYearCoupons`, а если в текущем году купонов нет —
запасной вариант `coupons[0].PayOnBond * CouponQuantityPerYear`.

---

## 3. Ключевая идея: премия и дисконт

Разница между ценой покупки и номиналом учитывается **автоматически**, потому что
в `totalInvestment` входит полная `bondPrice`, а в `totalReturn` возвращается только
`Nominal`. Распишем:

```text
finalProfit = (Nominal - bondPrice) + totalFutureCoupons - Nkd - couponTax - nominalPriceTax
              \_______________/
              капитальный результат
```

- **Цена > номинала (премия):** `Nominal - bondPrice < 0` → убыток уже вычтен из прибыли.
  Дополнительно вычитать разницу **не нужно** — это было бы двойным учётом.
- **Цена < номинала (дисконт):** `Nominal - bondPrice > 0` → прибыль от роста к номиналу,
  и с неё начисляется `nominalPriceTax` (13%).

---

## 4. Проверочные примеры

Дата оценки во всех примерах — условное «сегодня». Числа сверены расчётом
(см. раздел 5).

### Облигация A — срок < 1 года

- `Nominal` = 1000 ₽
- Купон 40 ₽, 2 раза в год (8% годовых)
- `daysToMaturity` = 183 (≈ полгода)
- `totalFutureCoupons` = 40 ₽ (один купон при погашении)
- `currentYearCoupons` = 80 ₽ (два купона в текущем году)
- `Nkd` = 5 ₽

| Шаг | Премия (цена 102%) | Дисконт (цена 98%) |
|---|---:|---:|
| `bondPrice` | 1020.00 | 980.00 |
| `totalInvestment` | 1025.00 | 985.00 |
| `couponTax` = (40−5)·0.13 | 4.55 | 4.55 |
| `nominalPriceTax` | 0.00 *(премия)* | 2.60 *(20·0.13)* |
| `totalReturn` = 1000+40 | 1040.00 | 1040.00 |
| **`finalProfit`** | **10.45** | **47.85** |
| `profitPerYear` = ·365/183 | 20.84 | 95.44 |
| **`percentByYear`** | **2.03%** | **9.69%** |
| `couponPercentByYear` | 7.80% | 8.12% |

### Облигация B — срок > 1 года

- `Nominal` = 1000 ₽
- Купон 45 ₽, 2 раза в год (9% годовых)
- `daysToMaturity` = 1095 (≈ 3 года)
- `totalFutureCoupons` = 270 ₽ (6 купонов до погашения)
- `currentYearCoupons` = 90 ₽ (два купона в текущем году)
- `Nkd` = 10 ₽

| Шаг | Премия (цена 103%) | Дисконт (цена 97%) |
|---|---:|---:|
| `bondPrice` | 1030.00 | 970.00 |
| `totalInvestment` | 1040.00 | 980.00 |
| `couponTax` = (270−10)·0.13 | 33.80 | 33.80 |
| `nominalPriceTax` | 0.00 *(премия)* | 3.90 *(30·0.13)* |
| `totalReturn` = 1000+270 | 1270.00 | 1270.00 |
| **`finalProfit`** | **196.20** | **252.30** |
| `profitPerYear` = ·365/1095 | 65.40 | 84.10 |
| **`percentByYear`** | **6.29%** | **8.58%** |
| `couponPercentByYear` | 8.65% | 9.18% |

### Что подтверждают примеры

1. **Премия снижает доходность, дисконт повышает** — при прочих равных:
   A: 2.03% (премия) < 9.69% (дисконт); B: 6.29% < 8.58%. Капитальный убыток/прибыль
   от схождения цены к номиналу учтён корректно.
2. **Короткая бумага чувствительнее к премии/дисконту**: убыток −20 ₽ у короткой A
   «размазывается» на 183 дня и сильно бьёт по годовой ставке (2.03%), тогда как
   −30 ₽ у длинной B распределяется на 1095 дней (6.29%).
3. **Налог на купон** одинаков для премии и дисконта одной бумаги (зависит только от
   `totalFutureCoupons` и `Nkd`), а `nominalPriceTax` появляется **только при дисконте**.

---

## 5. Воспроизведение расчёта

```python
def calc(nominal, close_pct, nkd, future_coupons, current_year_coupons, days, first_coupon, cpy):
    bond_price = close_pct * nominal / 100
    total_investment = bond_price + nkd
    annual = current_year_coupons if current_year_coupons > 0 else first_coupon * cpy
    coupon_pct_year = annual * 100 / total_investment
    coupon_tax = max((future_coupons - nkd) * 0.13, 0)
    nominal_price_tax = (nominal - bond_price) * 0.13 if (nominal - bond_price) > 0 else 0
    total_return = nominal + future_coupons
    final_profit = total_return - total_investment - coupon_tax - nominal_price_tax
    d = max(days, 1)
    profit_per_year = final_profit * 365 / d
    percent_by_year = 100 * profit_per_year / total_investment
    return final_profit, percent_by_year, coupon_pct_year

# A (премия):  calc(1000, 102, 5,  40,  80,  183, 40, 2) -> finalProfit=10.45, percentByYear=2.03%
# A (дисконт): calc(1000,  98, 5,  40,  80,  183, 40, 2) -> finalProfit=47.85, percentByYear=9.69%
# B (премия):  calc(1000, 103, 10, 270, 90, 1095, 45, 2) -> finalProfit=196.20, percentByYear=6.29%
# B (дисконт): calc(1000,  97, 10, 270, 90, 1095, 45, 2) -> finalProfit=252.30, percentByYear=8.58%
```

---

## 6. Известные упрощения

Это **не баги**, а сознательные допущения текущей модели — важно помнить при сравнении
с цифрами брокера:

1. **Простая, а не эффективная доходность.** `percentByYear` — линейный пересчёт
   (`×365/days`), без реинвестирования купонов. Для срока < 1 года слегка занижает,
   для многолетних бумаг занижение заметнее, чем эффективная (IRR) ставка.
2. **Доходность «чистая» (после налога 13%).** Брокер обычно показывает **грязную**
   доходность к погашению. Разница ≈ величина налога на купоны и на дисконт.
3. **`couponPercentByYear` берёт все купоны года, включая уже выплаченные** — это
   характеристика бумаги, а не доход конкретного покупателя.
4. **Налог на купон вычитает весь `Nkd` сразу**, тогда как формально НКД уменьшает
   базу только ближайшего купона. На итог влияет незначительно.
