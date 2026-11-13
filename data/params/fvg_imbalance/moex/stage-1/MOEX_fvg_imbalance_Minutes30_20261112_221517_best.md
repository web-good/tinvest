# Бэктест MOEX (Minutes30)

- Период: 2023-11-12 22:15 — 2025-11-12 22:15
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
| StopATR | 1 |
| TakeProfitATR | 2 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 22 |
| Выигрышных / проигрышных | 15 / 7 |
| Win rate | 68.18% |
| Gross profit / loss | 14168.03 / 5868.90 |
| Profit factor | 2.414 |
| Чистый PnL | 8299.13 (8.30%) |
| Стартовый / финальный капитал | 100000.00 / 108299.13 |
| Макс. просадка | 3239.64 (3.01%) |
| Средняя прибыль / убыток | 944.54 / 838.41 |
| Expectancy | 377.23 |
| Лучшая / худшая сделка | 2138.57 / -2267.92 |
| Exposure | 0.80% |
| CAGR | 4.06% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2024-01-23 12:30 | 198.7900 | 194.9756 | 2024-01-24 09:30 | 198.0900 | STOCH_EXIT | 23 | -449.22 | -0.45% | 196.1400 | 206.4187 | 3.8144 | M30 bullish FVG [194.4500, 196.1400], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 2 | 2024-01-30 11:30 | 204.2800 | 199.9859 | 2024-01-30 16:00 | 204.1400 | STOCH_EXIT | 9 | -165.22 | -0.17% | 205.1500 | 212.8682 | 4.2941 | M30 bullish FVG [204.0000, 205.1500], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 3 | 2024-02-23 10:30 | 190.0900 | 185.6596 | 2024-02-23 16:30 | 191.6800 | STOCH_EXIT | 12 | 727.54 | 0.74% | 188.4300 | 198.9508 | 4.4304 | M30 bullish FVG [185.6800, 188.4300], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 4 | 2024-03-06 10:30 | 205.1400 | 200.5954 | 2024-03-06 15:00 | 206.3000 | STOCH_EXIT | 9 | 458.05 | 0.46% | 202.6600 | 214.2291 | 4.5446 | M30 bullish FVG [200.8500, 202.6600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 5 | 2024-06-03 13:30 | 240.3100 | 235.0161 | 2024-06-03 14:00 | 235.0161 | SL | 1 | -2267.92 | -2.30% | 242.1300 | 250.8977 | 5.2939 | M30 bullish FVG [240.0000, 242.1300], stochastic crossed below 20 | FVG stop 235.0161 |
| 6 | 2024-06-05 12:30 | 247.8700 | 241.7490 | 2024-06-05 17:30 | 249.8300 | STOCH_EXIT | 10 | 667.35 | 0.69% | 246.8100 | 260.1121 | 6.1210 | M30 bullish FVG [244.9700, 246.8100], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 7 | 2024-06-14 10:00 | 235.4100 | 227.0457 | 2024-06-14 15:30 | 239.4300 | STOCH_EXIT | 11 | 1588.68 | 1.61% | 234.0000 | 252.1385 | 8.3643 | M30 bullish FVG [226.0400, 234.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 8 | 2024-08-26 18:30 | 212.8500 | 207.1528 | 2024-08-26 22:30 | 213.8500 | STOCH_EXIT | 8 | 369.73 | 0.37% | 213.6700 | 224.2443 | 5.6972 | M30 bullish FVG [211.8700, 213.6700], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 9 | 2024-11-18 15:30 | 205.1900 | 198.7367 | 2024-11-18 20:00 | 206.8700 | STOCH_EXIT | 9 | 722.25 | 0.72% | 206.4600 | 218.0967 | 6.4533 | M30 bullish FVG [203.4900, 206.4600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 10 | 2024-12-19 17:00 | 175.6100 | 169.1854 | 2024-12-19 18:30 | 179.0400 | STOCH_EXIT | 3 | 1854.02 | 1.85% | 177.4000 | 188.4592 | 6.4246 | M30 bullish FVG [174.8600, 177.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 11 | 2024-12-20 22:30 | 188.8800 | 182.4500 | 2024-12-21 07:00 | 189.8900 | STOCH_EXIT | 3 | 443.13 | 0.43% | 183.8500 | 201.7400 | 6.4300 | M30 bullish FVG [178.8400, 183.8500], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 12 | 2025-02-12 13:00 | 215.6400 | 210.0340 | 2025-02-12 19:00 | 216.1900 | STOCH_EXIT | 12 | 160.36 | 0.15% | 212.5200 | 226.8520 | 5.6060 | M30 bullish FVG [210.7800, 212.5200], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 13 | 2025-02-13 19:00 | 234.0000 | 227.8351 | 2025-02-14 07:30 | 236.4700 | STOCH_EXIT | 5 | 983.30 | 0.95% | 234.0900 | 246.3297 | 6.1649 | M30 bullish FVG [230.1600, 234.0900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 14 | 2025-07-10 07:30 | 168.8800 | 164.8421 | 2025-07-10 11:00 | 172.5000 | STOCH_EXIT | 7 | 2138.57 | 2.04% | 192.9200 | 176.9557 | 4.0379 | M30 bullish FVG [191.5900, 192.9200], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 15 | 2025-07-10 19:30 | 172.9900 | 168.9521 | 2025-07-10 23:30 | 173.5000 | STOCH_EXIT | 8 | 205.42 | 0.19% | 192.9200 | 181.0657 | 4.0379 | M30 bullish FVG [191.5900, 192.9200], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 16 | 2025-07-15 11:00 | 172.2900 | 166.4900 | 2025-07-15 22:00 | 171.3000 | STOCH_EXIT | 22 | -720.31 | -0.67% | 171.7000 | 183.8900 | 5.8000 | M30 bullish FVG [167.8400, 171.7000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 17 | 2025-08-07 19:00 | 180.4500 | 176.2053 | 2025-08-07 23:30 | 180.4300 | STOCH_EXIT | 9 | -118.26 | -0.11% | 180.8000 | 188.9393 | 4.2447 | M30 bullish FVG [178.7400, 180.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 18 | 2025-09-29 17:00 | 168.1900 | 165.0511 | 2025-09-29 19:00 | 165.0511 | SL | 4 | -2082.48 | -1.96% | 168.8600 | 174.4678 | 3.1389 | M30 bullish FVG [167.6500, 168.8600], stochastic crossed below 20 | FVG stop 165.0511 |
| 19 | 2025-10-06 15:30 | 158.0200 | 154.6612 | 2025-10-06 16:30 | 161.3200 | STOCH_EXIT | 2 | 2072.62 | 1.99% | 158.8400 | 164.7377 | 3.3588 | M30 bullish FVG [157.7600, 158.8400], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 20 | 2025-10-09 20:00 | 159.1800 | 155.2736 | 2025-10-10 09:30 | 159.2400 | STOCH_EXIT | 14 | -65.48 | -0.06% | 159.0000 | 166.9929 | 3.9064 | M30 bullish FVG [156.6900, 159.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 21 | 2025-10-13 23:00 | 156.5800 | 152.5346 | 2025-10-14 07:00 | 157.1700 | STOCH_EXIT | 3 | 290.19 | 0.28% | 157.9100 | 164.6709 | 4.0454 | M30 bullish FVG [155.9500, 157.9100], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 22 | 2025-10-17 10:00 | 165.6100 | 161.0895 | 2025-10-17 14:00 | 168.1000 | STOCH_EXIT | 8 | 1486.81 | 1.40% | 163.2600 | 174.6510 | 4.5205 | M30 bullish FVG [158.2200, 163.2600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 97861.00
- Макс: 108365.72
- Финал: 108299.13

Полная кривая — в `*_equity.csv`.
