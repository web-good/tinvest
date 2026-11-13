# Бэктест MOEX (Minutes30)

- Период: 2023-11-12 22:16 — 2025-11-12 22:16
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.25 |
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
| Всего сделок | 33 |
| Выигрышных / проигрышных | 19 / 14 |
| Win rate | 57.58% |
| Gross profit / loss | 16398.89 / 12678.94 |
| Profit factor | 1.293 |
| Чистый PnL | 3719.95 (3.72%) |
| Стартовый / финальный капитал | 100000.00 / 103719.95 |
| Макс. просадка | 4341.66 (4.16%) |
| Средняя прибыль / убыток | 863.10 / 905.64 |
| Expectancy | 112.73 |
| Лучшая / худшая сделка | 2035.09 / -3715.00 |
| Exposure | 1.55% |
| CAGR | 1.84% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-12-19 13:30 | 200.9100 | 195.4420 | 2023-12-19 20:00 | 200.4400 | STOCH_EXIT | 13 | -328.63 | -0.33% | 199.0000 | 211.8460 | 5.4680 | M30 bullish FVG [197.4700, 199.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 2 | 2024-01-04 10:30 | 195.0000 | 189.9379 | 2024-01-04 18:30 | 194.2700 | STOCH_EXIT | 16 | -471.56 | -0.47% | 191.6700 | 205.1242 | 5.0621 | M30 bullish FVG [190.2300, 191.6700], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 3 | 2024-01-18 16:30 | 193.2000 | 189.5489 | 2024-01-19 17:30 | 193.0600 | STOCH_EXIT | 31 | -169.90 | -0.17% | 194.2700 | 200.5022 | 3.6511 | M30 bullish FVG [193.1800, 194.2700], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 4 | 2024-01-23 12:30 | 198.7900 | 194.9756 | 2024-01-24 09:30 | 198.0900 | STOCH_EXIT | 23 | -440.24 | -0.45% | 196.1400 | 206.4187 | 3.8144 | M30 bullish FVG [194.4500, 196.1400], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 5 | 2024-01-30 11:30 | 204.2800 | 199.9859 | 2024-01-30 16:00 | 204.1400 | STOCH_EXIT | 9 | -165.22 | -0.17% | 205.1500 | 212.8682 | 4.2941 | M30 bullish FVG [204.0000, 205.1500], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 6 | 2024-02-23 10:30 | 190.0900 | 185.6596 | 2024-02-23 16:30 | 191.6800 | STOCH_EXIT | 12 | 713.55 | 0.74% | 188.4300 | 198.9508 | 4.4304 | M30 bullish FVG [185.6800, 188.4300], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 7 | 2024-03-06 10:30 | 205.1400 | 200.5954 | 2024-03-06 15:00 | 206.3000 | STOCH_EXIT | 9 | 458.05 | 0.46% | 202.6600 | 214.2291 | 4.5446 | M30 bullish FVG [200.8500, 202.6600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 8 | 2024-03-20 15:00 | 209.3600 | 205.5533 | 2024-03-20 17:30 | 210.6600 | STOCH_EXIT | 5 | 512.30 | 0.52% | 209.5600 | 216.9734 | 3.8067 | M30 bullish FVG [208.6000, 209.5600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 9 | 2024-05-16 23:30 | 236.6300 | 232.8518 | 2024-05-17 10:00 | 237.7300 | STOCH_EXIT | 2 | 362.38 | 0.36% | 237.0000 | 244.1865 | 3.7782 | M30 bullish FVG [236.0000, 237.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 10 | 2024-06-03 13:30 | 240.3100 | 235.0161 | 2024-06-03 14:00 | 235.0161 | SL | 1 | -2267.92 | -2.30% | 242.1300 | 250.8977 | 5.2939 | M30 bullish FVG [240.0000, 242.1300], stochastic crossed below 20 | FVG stop 235.0161 |
| 11 | 2024-06-05 12:30 | 247.8700 | 241.7490 | 2024-06-05 17:30 | 249.8300 | STOCH_EXIT | 10 | 667.35 | 0.69% | 246.8100 | 260.1121 | 6.1210 | M30 bullish FVG [244.9700, 246.8100], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 12 | 2024-06-14 10:00 | 235.4100 | 227.0457 | 2024-06-14 15:30 | 239.4300 | STOCH_EXIT | 11 | 1550.86 | 1.61% | 234.0000 | 252.1385 | 8.3643 | M30 bullish FVG [226.0400, 234.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 13 | 2024-08-01 18:00 | 232.3900 | 226.1155 | 2024-08-02 16:00 | 229.1000 | STOCH_EXIT | 25 | -1513.92 | -1.51% | 233.3200 | 244.9389 | 6.2745 | M30 bullish FVG [231.7100, 233.3200], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 14 | 2024-08-26 18:30 | 212.8500 | 207.1528 | 2024-08-26 22:30 | 213.8500 | STOCH_EXIT | 8 | 361.86 | 0.37% | 213.6700 | 224.2443 | 5.6972 | M30 bullish FVG [211.8700, 213.6700], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 15 | 2024-09-24 10:30 | 216.3100 | 210.1682 | 2024-09-24 17:30 | 216.3900 | STOCH_EXIT | 14 | -61.36 | -0.06% | 217.0000 | 228.5937 | 6.1418 | M30 bullish FVG [215.4400, 217.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 16 | 2024-11-18 15:30 | 205.1900 | 198.7367 | 2024-11-18 20:00 | 206.8700 | STOCH_EXIT | 9 | 707.51 | 0.72% | 206.4600 | 218.0967 | 6.4533 | M30 bullish FVG [203.4900, 206.4600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 17 | 2024-12-19 17:00 | 175.6100 | 169.1854 | 2024-12-19 18:30 | 179.0400 | STOCH_EXIT | 3 | 1821.50 | 1.85% | 177.4000 | 188.4592 | 6.4246 | M30 bullish FVG [174.8600, 177.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 18 | 2024-12-20 22:30 | 188.8800 | 182.4500 | 2024-12-21 07:00 | 189.8900 | STOCH_EXIT | 3 | 434.93 | 0.43% | 183.8500 | 201.7400 | 6.4300 | M30 bullish FVG [178.8400, 183.8500], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 19 | 2025-02-12 13:00 | 215.6400 | 210.0340 | 2025-02-12 19:00 | 216.1900 | STOCH_EXIT | 12 | 157.02 | 0.15% | 212.5200 | 226.8520 | 5.6060 | M30 bullish FVG [210.7800, 212.5200], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 20 | 2025-02-13 19:00 | 234.0000 | 227.8351 | 2025-02-14 07:30 | 236.4700 | STOCH_EXIT | 5 | 960.95 | 0.95% | 234.0900 | 246.3297 | 6.1649 | M30 bullish FVG [230.1600, 234.0900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 21 | 2025-03-04 11:00 | 223.8500 | 216.2055 | 2025-03-05 18:30 | 222.5500 | STOCH_EXIT | 50 | -700.67 | -0.68% | 218.6500 | 239.1390 | 7.6445 | M30 bullish FVG [216.5900, 218.6500], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 22 | 2025-03-28 14:30 | 199.3600 | 192.9788 | 2025-03-28 18:00 | 201.6200 | STOCH_EXIT | 7 | 1050.35 | 1.03% | 201.1300 | 212.1224 | 6.3812 | M30 bullish FVG [199.3500, 201.1300], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 23 | 2025-04-03 11:00 | 202.5100 | 195.4246 | 2025-04-03 16:00 | 195.4246 | SL | 10 | -3715.00 | -3.60% | 204.9900 | 216.6807 | 7.0854 | M30 bullish FVG [202.9900, 204.9900], stochastic crossed below 20 | FVG stop 195.4246 |
| 24 | 2025-07-10 07:30 | 168.8800 | 164.8421 | 2025-07-10 11:00 | 172.5000 | STOCH_EXIT | 7 | 2035.09 | 2.04% | 192.9200 | 176.9557 | 4.0379 | M30 bullish FVG [191.5900, 192.9200], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 25 | 2025-07-10 19:30 | 172.9900 | 168.9521 | 2025-07-10 23:30 | 173.5000 | STOCH_EXIT | 8 | 195.32 | 0.19% | 192.9200 | 181.0657 | 4.0379 | M30 bullish FVG [191.5900, 192.9200], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 26 | 2025-07-15 11:00 | 172.2900 | 166.4900 | 2025-07-15 22:00 | 171.3000 | STOCH_EXIT | 22 | -685.46 | -0.67% | 171.7000 | 183.8900 | 5.8000 | M30 bullish FVG [167.8400, 171.7000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 27 | 2025-08-07 19:00 | 180.4500 | 176.2053 | 2025-08-07 23:30 | 180.4300 | STOCH_EXIT | 9 | -112.25 | -0.11% | 180.8000 | 188.9393 | 4.2447 | M30 bullish FVG [178.7400, 180.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 28 | 2025-09-26 17:30 | 167.9300 | 164.7543 | 2025-09-26 21:00 | 169.3200 | STOCH_EXIT | 7 | 732.82 | 0.73% | 168.5100 | 174.2815 | 3.1757 | M30 bullish FVG [167.5900, 168.5100], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 29 | 2025-09-29 17:00 | 168.1900 | 165.0511 | 2025-09-29 19:00 | 165.0511 | SL | 4 | -1983.32 | -1.96% | 168.8600 | 174.4678 | 3.1389 | M30 bullish FVG [167.6500, 168.8600], stochastic crossed below 20 | FVG stop 165.0511 |
| 30 | 2025-10-06 15:30 | 158.0200 | 154.6612 | 2025-10-06 16:30 | 161.3200 | STOCH_EXIT | 2 | 1978.41 | 1.99% | 158.8400 | 164.7377 | 3.3588 | M30 bullish FVG [157.7600, 158.8400], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 31 | 2025-10-09 20:00 | 159.1800 | 155.2736 | 2025-10-10 09:30 | 159.2400 | STOCH_EXIT | 14 | -63.49 | -0.06% | 159.0000 | 166.9929 | 3.9064 | M30 bullish FVG [156.6900, 159.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 32 | 2025-10-13 23:00 | 156.5800 | 152.5346 | 2025-10-14 07:00 | 157.1700 | STOCH_EXIT | 3 | 281.53 | 0.28% | 157.9100 | 164.6709 | 4.0454 | M30 bullish FVG [155.9500, 157.9100], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 33 | 2025-10-17 10:00 | 165.6100 | 161.0895 | 2025-10-17 14:00 | 168.1000 | STOCH_EXIT | 8 | 1417.12 | 1.40% | 163.2600 | 174.6510 | 4.5205 | M30 bullish FVG [158.2200, 163.2600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 97760.58
- Макс: 104265.84
- Финал: 103719.95

Полная кривая — в `*_equity.csv`.
