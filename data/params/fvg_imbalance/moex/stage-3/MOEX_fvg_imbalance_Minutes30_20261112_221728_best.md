# Бэктест MOEX (Minutes30)

- Период: 2023-11-12 22:17 — 2025-11-12 22:17
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.3 |
| MaxWaitBars | 30 |
| StochPeriod | 9 |
| UseStochExit | 1 |
| StopATR | 1.2 |
| TakeProfitATR | 1 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 22 |
| Выигрышных / проигрышных | 15 / 7 |
| Win rate | 68.18% |
| Gross profit / loss | 14388.78 / 3919.07 |
| Profit factor | 3.671 |
| Чистый PnL | 10469.71 (10.47%) |
| Стартовый / финальный капитал | 100000.00 / 110469.71 |
| Макс. просадка | 3077.34 (2.82%) |
| Средняя прибыль / убыток | 959.25 / 559.87 |
| Expectancy | 475.90 |
| Лучшая / худшая сделка | 2173.07 / -1489.35 |
| Exposure | 0.88% |
| CAGR | 5.10% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2024-01-23 12:30 | 198.7900 | 194.2128 | 2024-01-24 09:30 | 198.0900 | STOCH_EXIT | 23 | -449.22 | -0.45% | 196.1400 | 202.6044 | 3.8144 | M30 bullish FVG [194.4500, 196.1400], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 2 | 2024-01-30 11:30 | 204.2800 | 199.1271 | 2024-01-30 16:00 | 204.1400 | STOCH_EXIT | 9 | -165.22 | -0.17% | 205.1500 | 208.5741 | 4.2941 | M30 bullish FVG [204.0000, 205.1500], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 3 | 2024-02-23 10:30 | 190.0900 | 184.7735 | 2024-02-23 16:30 | 191.6800 | STOCH_EXIT | 12 | 727.54 | 0.74% | 188.4300 | 194.5204 | 4.4304 | M30 bullish FVG [185.6800, 188.4300], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 4 | 2024-03-06 10:30 | 205.1400 | 199.6865 | 2024-03-06 15:00 | 206.3000 | STOCH_EXIT | 9 | 458.05 | 0.46% | 202.6600 | 209.6846 | 4.5446 | M30 bullish FVG [200.8500, 202.6600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 5 | 2024-06-03 13:30 | 240.3100 | 233.9574 | 2024-06-03 20:00 | 238.3600 | STOCH_EXIT | 13 | -897.63 | -0.91% | 242.1300 | 245.6039 | 5.2939 | M30 bullish FVG [240.0000, 242.1300], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 6 | 2024-06-05 12:30 | 247.8700 | 240.5248 | 2024-06-05 17:30 | 249.8300 | STOCH_EXIT | 10 | 684.46 | 0.69% | 246.8100 | 253.9910 | 6.1210 | M30 bullish FVG [244.9700, 246.8100], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 7 | 2024-06-14 10:00 | 235.4100 | 225.3729 | 2024-06-14 15:30 | 239.4300 | STOCH_EXIT | 11 | 1588.68 | 1.61% | 234.0000 | 243.7743 | 8.3643 | M30 bullish FVG [226.0400, 234.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 8 | 2024-08-26 18:30 | 212.8500 | 206.0134 | 2024-08-26 22:30 | 213.8500 | STOCH_EXIT | 8 | 369.73 | 0.37% | 213.6700 | 218.5472 | 5.6972 | M30 bullish FVG [211.8700, 213.6700], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 9 | 2024-11-18 15:30 | 205.1900 | 197.4460 | 2024-11-18 20:00 | 206.8700 | STOCH_EXIT | 9 | 722.25 | 0.72% | 206.4600 | 211.6433 | 6.4533 | M30 bullish FVG [203.4900, 206.4600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 10 | 2024-12-19 17:00 | 175.6100 | 167.9005 | 2024-12-19 18:30 | 179.0400 | STOCH_EXIT | 3 | 1886.55 | 1.85% | 177.4000 | 182.0346 | 6.4246 | M30 bullish FVG [174.8600, 177.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 11 | 2024-12-20 22:30 | 188.8800 | 181.1640 | 2024-12-21 07:00 | 189.8900 | STOCH_EXIT | 3 | 451.34 | 0.43% | 183.8500 | 195.3100 | 6.4300 | M30 bullish FVG [178.8400, 183.8500], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 12 | 2025-02-12 13:00 | 215.6400 | 208.9128 | 2025-02-12 19:00 | 216.1900 | STOCH_EXIT | 12 | 160.36 | 0.15% | 212.5200 | 221.2460 | 5.6060 | M30 bullish FVG [210.7800, 212.5200], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 13 | 2025-02-13 19:00 | 234.0000 | 226.6022 | 2025-02-14 07:30 | 236.4700 | STOCH_EXIT | 5 | 1005.64 | 0.95% | 234.0900 | 240.1649 | 6.1649 | M30 bullish FVG [230.1600, 234.0900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 14 | 2025-07-10 07:30 | 168.8800 | 164.0346 | 2025-07-10 11:00 | 172.5000 | STOCH_EXIT | 7 | 2173.07 | 2.04% | 192.9200 | 172.9179 | 4.0379 | M30 bullish FVG [191.5900, 192.9200], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 15 | 2025-07-10 19:30 | 172.9900 | 168.1446 | 2025-07-10 23:30 | 173.5000 | STOCH_EXIT | 8 | 208.79 | 0.19% | 192.9200 | 177.0279 | 4.0379 | M30 bullish FVG [191.5900, 192.9200], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 16 | 2025-07-15 11:00 | 172.2900 | 165.3300 | 2025-07-15 22:00 | 171.3000 | STOCH_EXIT | 22 | -731.93 | -0.67% | 171.7000 | 178.0900 | 5.8000 | M30 bullish FVG [167.8400, 171.7000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 17 | 2025-08-07 19:00 | 180.4500 | 175.3564 | 2025-08-07 23:30 | 180.4300 | STOCH_EXIT | 9 | -118.26 | -0.11% | 180.8000 | 184.6947 | 4.2447 | M30 bullish FVG [178.7400, 180.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 18 | 2025-09-29 17:00 | 168.1900 | 164.4233 | 2025-09-29 22:30 | 166.0300 | STOCH_EXIT | 11 | -1489.35 | -1.38% | 168.8600 | 171.3289 | 3.1389 | M30 bullish FVG [167.6500, 168.8600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 19 | 2025-10-06 15:30 | 158.0200 | 153.9894 | 2025-10-06 16:30 | 161.3788 | TP | 2 | 2143.42 | 2.02% | 158.8400 | 161.3788 | 3.3588 | M30 bullish FVG [157.7600, 158.8400], stochastic crossed below 20 | FVG target 161.3788 |
| 20 | 2025-10-09 20:00 | 159.1800 | 154.4923 | 2025-10-10 09:30 | 159.2400 | STOCH_EXIT | 14 | -67.46 | -0.06% | 159.0000 | 163.0864 | 3.9064 | M30 bullish FVG [156.6900, 159.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 21 | 2025-10-13 23:00 | 156.5800 | 151.7255 | 2025-10-14 07:00 | 157.1700 | STOCH_EXIT | 3 | 298.86 | 0.28% | 157.9100 | 160.6254 | 4.0454 | M30 bullish FVG [155.9500, 157.9100], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 22 | 2025-10-17 10:00 | 165.6100 | 160.1854 | 2025-10-17 14:00 | 168.1000 | STOCH_EXIT | 8 | 1510.04 | 1.40% | 163.2600 | 170.1305 | 4.5205 | M30 bullish FVG [158.2200, 163.2600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 98693.29
- Макс: 110537.34
- Финал: 110469.71

Полная кривая — в `*_equity.csv`.
