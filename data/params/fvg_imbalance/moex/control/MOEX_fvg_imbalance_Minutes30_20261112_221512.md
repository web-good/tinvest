# Бэктест MOEX (Minutes30)

- Период: 2023-11-12 22:15 — 2025-11-12 22:15
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.25 |
| MaxWaitBars | 30 |
| StochPeriod | 14 |
| UseStochExit | 1 |
| StopATR | 1 |
| TakeProfitATR | 2 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 26 |
| Выигрышных / проигрышных | 12 / 14 |
| Win rate | 46.15% |
| Gross profit / loss | 10211.26 / 15904.64 |
| Profit factor | 0.642 |
| Чистый PnL | -5693.38 (-5.69%) |
| Стартовый / финальный капитал | 100000.00 / 94306.62 |
| Макс. просадка | 8283.44 (8.28%) |
| Средняя прибыль / убыток | 850.94 / 1136.05 |
| Expectancy | -218.98 |
| Лучшая / худшая сделка | 1932.79 / -3350.79 |
| Exposure | 2.11% |
| CAGR | -2.88% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-12-19 13:30 | 200.9100 | 195.4420 | 2023-12-19 20:30 | 200.4500 | STOCH_EXIT | 14 | -323.73 | -0.33% | 199.0000 | 211.8460 | 5.4680 | M30 bullish FVG [197.4700, 199.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 2 | 2024-01-04 10:30 | 195.0000 | 189.9379 | 2024-01-07 17:00 | 193.1300 | STOCH_EXIT | 88 | -1052.67 | -1.06% | 191.6700 | 205.1242 | 5.0621 | M30 bullish FVG [190.2300, 191.6700], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 3 | 2024-01-19 10:30 | 193.2400 | 189.6304 | 2024-01-19 21:30 | 192.9600 | STOCH_EXIT | 22 | -241.28 | -0.24% | 194.2700 | 200.4592 | 3.6096 | M30 bullish FVG [193.1800, 194.2700], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 4 | 2024-01-23 12:30 | 198.7900 | 194.9756 | 2024-01-24 18:30 | 197.7400 | STOCH_EXIT | 41 | -611.65 | -0.63% | 196.1400 | 206.4187 | 3.8144 | M30 bullish FVG [194.4500, 196.1400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 5 | 2024-01-30 14:00 | 203.8000 | 199.5059 | 2024-01-31 17:30 | 203.8000 | STOCH_EXIT | 36 | -95.79 | -0.10% | 205.1500 | 212.3882 | 4.2941 | M30 bullish FVG [204.0000, 205.1500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 6 | 2024-03-06 19:00 | 205.2100 | 200.6654 | 2024-03-07 10:00 | 206.7200 | STOCH_EXIT | 11 | 612.90 | 0.64% | 202.6600 | 214.2991 | 4.5446 | M30 bullish FVG [200.8500, 202.6600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 7 | 2024-03-20 15:30 | 208.8300 | 205.0233 | 2024-03-20 17:30 | 210.6600 | STOCH_EXIT | 4 | 761.52 | 0.78% | 209.5600 | 216.4434 | 3.8067 | M30 bullish FVG [208.6000, 209.5600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 8 | 2024-06-03 13:30 | 240.3100 | 235.0161 | 2024-06-03 14:00 | 235.0161 | SL | 1 | -2267.92 | -2.30% | 242.1300 | 250.8977 | 5.2939 | M30 bullish FVG [240.0000, 242.1300], stochastic crossed below 20 | FVG stop 235.0161 |
| 9 | 2024-06-05 14:30 | 247.9200 | 241.7990 | 2024-06-05 17:30 | 249.8300 | STOCH_EXIT | 6 | 647.84 | 0.67% | 246.8100 | 260.1621 | 6.1210 | M30 bullish FVG [244.9700, 246.8100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 10 | 2024-06-14 10:00 | 235.4100 | 227.0457 | 2024-06-14 16:00 | 240.0000 | STOCH_EXIT | 12 | 1784.44 | 1.85% | 234.0000 | 252.1385 | 8.3643 | M30 bullish FVG [226.0400, 234.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 11 | 2024-08-01 18:30 | 231.2200 | 224.9455 | 2024-08-02 16:30 | 229.8900 | STOCH_EXIT | 25 | -655.43 | -0.67% | 233.3200 | 243.7689 | 6.2745 | M30 bullish FVG [231.7100, 233.3200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 12 | 2024-09-24 10:30 | 216.3100 | 210.1682 | 2024-09-24 19:30 | 216.4800 | STOCH_EXIT | 18 | -20.88 | -0.02% | 217.0000 | 228.5937 | 6.1418 | M30 bullish FVG [215.4400, 217.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 13 | 2024-11-18 22:30 | 205.5200 | 199.0667 | 2024-11-19 16:30 | 199.0667 | SL | 22 | -3128.14 | -3.24% | 206.4600 | 218.4267 | 6.4533 | M30 bullish FVG [203.4900, 206.4600], stochastic crossed below 20 | FVG stop 199.0667 |
| 14 | 2025-02-12 15:00 | 214.9000 | 209.2940 | 2025-02-12 19:30 | 217.2200 | STOCH_EXIT | 9 | 925.73 | 0.98% | 212.5200 | 226.1120 | 5.6060 | M30 bullish FVG [210.7800, 212.5200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 15 | 2025-02-14 12:30 | 233.8600 | 227.1912 | 2025-02-14 16:00 | 227.1912 | SL | 7 | -2828.72 | -2.95% | 234.0900 | 247.1976 | 6.6688 | M30 bullish FVG [230.1600, 234.0900], stochastic crossed below 20 | FVG stop 227.1912 |
| 16 | 2025-03-04 14:00 | 222.2700 | 214.6255 | 2025-03-05 18:30 | 222.5500 | STOCH_EXIT | 44 | 24.19 | 0.03% | 218.6500 | 237.5590 | 7.6445 | M30 bullish FVG [216.5900, 218.6500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 17 | 2025-03-28 15:00 | 198.1500 | 191.7688 | 2025-03-28 18:00 | 201.6200 | STOCH_EXIT | 6 | 1536.95 | 1.65% | 201.1300 | 210.9124 | 6.3812 | M30 bullish FVG [199.3500, 201.1300], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 18 | 2025-04-03 11:00 | 202.5100 | 195.4246 | 2025-04-03 16:00 | 195.4246 | SL | 10 | -3350.79 | -3.60% | 204.9900 | 216.6807 | 7.0854 | M30 bullish FVG [202.9900, 204.9900], stochastic crossed below 20 | FVG stop 195.4246 |
| 19 | 2025-07-10 07:30 | 168.8800 | 164.8421 | 2025-07-10 14:00 | 172.6300 | STOCH_EXIT | 13 | 1932.79 | 2.12% | 192.9200 | 176.9557 | 4.0379 | M30 bullish FVG [191.5900, 192.9200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 20 | 2025-07-10 19:30 | 172.9900 | 168.9521 | 2025-07-11 09:00 | 174.5900 | STOCH_EXIT | 14 | 770.15 | 0.82% | 192.9200 | 181.0657 | 4.0379 | M30 bullish FVG [191.5900, 192.9200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 21 | 2025-07-15 11:30 | 172.0600 | 166.2600 | 2025-07-16 07:00 | 171.8600 | STOCH_EXIT | 26 | -200.86 | -0.22% | 171.7000 | 183.6600 | 5.8000 | M30 bullish FVG [167.8400, 171.7000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 22 | 2025-08-07 18:30 | 181.0500 | 176.8053 | 2025-08-08 08:00 | 181.3300 | STOCH_EXIT | 14 | 51.38 | 0.05% | 180.8000 | 189.5393 | 4.2447 | M30 bullish FVG [178.7400, 180.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 23 | 2025-09-29 17:30 | 167.9000 | 164.7611 | 2025-09-30 07:30 | 166.3300 | STOCH_EXIT | 15 | -972.78 | -1.03% | 168.8600 | 174.1778 | 3.1389 | M30 bullish FVG [167.6500, 168.8600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 24 | 2025-10-07 07:00 | 161.0600 | 157.4961 | 2025-10-07 08:00 | 162.3300 | STOCH_EXIT | 2 | 631.73 | 0.69% | 160.8000 | 168.1878 | 3.5639 | M30 bullish FVG [159.9200, 160.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 25 | 2025-10-10 10:30 | 158.4000 | 154.3726 | 2025-10-10 11:00 | 159.4600 | STOCH_EXIT | 1 | 531.63 | 0.57% | 159.0000 | 166.4548 | 4.0274 | M30 bullish FVG [156.6900, 159.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 26 | 2025-10-13 22:00 | 156.7000 | 152.6546 | 2025-10-15 08:30 | 156.6000 | STOCH_EXIT | 43 | -153.99 | -0.16% | 157.9100 | 164.7909 | 4.0454 | M30 bullish FVG [155.9500, 157.9100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 91716.56
- Макс: 100000.00
- Финал: 94306.62

Полная кривая — в `*_equity.csv`.
