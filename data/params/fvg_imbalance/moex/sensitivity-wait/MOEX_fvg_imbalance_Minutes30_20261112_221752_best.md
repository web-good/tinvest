# Бэктест MOEX (Minutes30)

- Период: 2023-11-12 22:17 — 2025-11-12 22:17
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.25 |
| MaxWaitBars | 15 |
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
| Выигрышных / проигрышных | 13 / 9 |
| Win rate | 59.09% |
| Gross profit / loss | 13254.98 / 6632.79 |
| Profit factor | 1.998 |
| Чистый PnL | 6622.19 (6.62%) |
| Стартовый / финальный капитал | 100000.00 / 106622.19 |
| Макс. просадка | 3625.00 (3.49%) |
| Средняя прибыль / убыток | 1019.61 / 736.98 |
| Expectancy | 301.01 |
| Лучшая / худшая сделка | 2079.44 / -1580.51 |
| Exposure | 1.08% |
| CAGR | 3.25% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2024-01-18 16:30 | 193.2000 | 188.8187 | 2024-01-19 17:30 | 193.0600 | STOCH_EXIT | 31 | -169.90 | -0.17% | 194.2700 | 196.8511 | 3.6511 | M30 bullish FVG [193.1800, 194.2700], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 2 | 2024-01-30 11:30 | 204.2800 | 199.1271 | 2024-01-30 16:00 | 204.1400 | STOCH_EXIT | 9 | -165.22 | -0.17% | 205.1500 | 208.5741 | 4.2941 | M30 bullish FVG [204.0000, 205.1500], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 3 | 2024-02-23 10:30 | 190.0900 | 184.7735 | 2024-02-23 16:30 | 191.6800 | STOCH_EXIT | 12 | 727.54 | 0.74% | 188.4300 | 194.5204 | 4.4304 | M30 bullish FVG [185.6800, 188.4300], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 4 | 2024-03-06 10:30 | 205.1400 | 199.6865 | 2024-03-06 15:00 | 206.3000 | STOCH_EXIT | 9 | 458.05 | 0.46% | 202.6600 | 209.6846 | 4.5446 | M30 bullish FVG [200.8500, 202.6600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 5 | 2024-03-20 15:00 | 209.3600 | 204.7919 | 2024-03-20 17:30 | 210.6600 | STOCH_EXIT | 5 | 523.20 | 0.52% | 209.5600 | 213.1667 | 3.8067 | M30 bullish FVG [208.6000, 209.5600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 6 | 2024-05-16 23:30 | 236.6300 | 232.0961 | 2024-05-17 10:00 | 237.7300 | STOCH_EXIT | 2 | 362.38 | 0.36% | 237.0000 | 240.4082 | 3.7782 | M30 bullish FVG [236.0000, 237.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 7 | 2024-06-03 13:30 | 240.3100 | 233.9574 | 2024-06-03 20:00 | 238.3600 | STOCH_EXIT | 13 | -919.52 | -0.91% | 242.1300 | 245.6039 | 5.2939 | M30 bullish FVG [240.0000, 242.1300], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 8 | 2024-08-01 18:00 | 232.3900 | 224.8606 | 2024-08-02 16:00 | 229.1000 | STOCH_EXIT | 25 | -1513.92 | -1.51% | 233.3200 | 238.6645 | 6.2745 | M30 bullish FVG [231.7100, 233.3200], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 9 | 2024-11-18 15:30 | 205.1900 | 197.4460 | 2024-11-18 20:00 | 206.8700 | STOCH_EXIT | 9 | 707.51 | 0.72% | 206.4600 | 211.6433 | 6.4533 | M30 bullish FVG [203.4900, 206.4600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 10 | 2024-12-19 17:00 | 175.6100 | 167.9005 | 2024-12-19 18:30 | 179.0400 | STOCH_EXIT | 3 | 1821.50 | 1.85% | 177.4000 | 182.0346 | 6.4246 | M30 bullish FVG [174.8600, 177.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 11 | 2025-02-13 19:00 | 234.0000 | 226.6022 | 2025-02-14 07:30 | 236.4700 | STOCH_EXIT | 5 | 960.95 | 0.95% | 234.0900 | 240.1649 | 6.1649 | M30 bullish FVG [230.1600, 234.0900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 12 | 2025-03-04 11:00 | 223.8500 | 214.6766 | 2025-03-05 18:30 | 222.5500 | STOCH_EXIT | 50 | -685.44 | -0.68% | 218.6500 | 231.4945 | 7.6445 | M30 bullish FVG [216.5900, 218.6500], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 13 | 2025-03-28 14:30 | 199.3600 | 191.7026 | 2025-03-28 18:00 | 201.6200 | STOCH_EXIT | 7 | 1050.35 | 1.03% | 201.1300 | 205.7412 | 6.3812 | M30 bullish FVG [199.3500, 201.1300], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 14 | 2025-04-03 11:00 | 202.5100 | 194.0076 | 2025-04-03 19:30 | 199.5500 | STOCH_EXIT | 17 | -1580.51 | -1.56% | 204.9900 | 209.5954 | 7.0854 | M30 bullish FVG [202.9900, 204.9900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 15 | 2025-07-10 07:30 | 168.8800 | 164.0346 | 2025-07-10 11:00 | 172.5000 | STOCH_EXIT | 7 | 2069.59 | 2.04% | 192.9200 | 172.9179 | 4.0379 | M30 bullish FVG [191.5900, 192.9200], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 16 | 2025-08-07 19:00 | 180.4500 | 175.3564 | 2025-08-07 23:30 | 180.4300 | STOCH_EXIT | 9 | -114.25 | -0.11% | 180.8000 | 184.6947 | 4.2447 | M30 bullish FVG [178.7400, 180.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 17 | 2025-09-26 17:30 | 167.9300 | 164.1191 | 2025-09-26 21:00 | 169.3200 | STOCH_EXIT | 7 | 745.04 | 0.73% | 168.5100 | 171.1057 | 3.1757 | M30 bullish FVG [167.5900, 168.5100], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 18 | 2025-09-29 17:00 | 168.1900 | 164.4233 | 2025-09-29 22:30 | 166.0300 | STOCH_EXIT | 11 | -1419.54 | -1.38% | 168.8600 | 171.3289 | 3.1389 | M30 bullish FVG [167.6500, 168.8600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 19 | 2025-10-06 15:30 | 158.0200 | 153.9894 | 2025-10-06 16:30 | 161.3788 | TP | 2 | 2079.44 | 2.02% | 158.8400 | 161.3788 | 3.3588 | M30 bullish FVG [157.7600, 158.8400], stochastic crossed below 20 | FVG target 161.3788 |
| 20 | 2025-10-09 20:00 | 159.1800 | 154.4923 | 2025-10-10 09:30 | 159.2400 | STOCH_EXIT | 14 | -64.49 | -0.06% | 159.0000 | 163.0864 | 3.9064 | M30 bullish FVG [156.6900, 159.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 21 | 2025-10-13 23:00 | 156.5800 | 151.7255 | 2025-10-14 07:00 | 157.1700 | STOCH_EXIT | 3 | 285.86 | 0.28% | 157.9100 | 160.6254 | 4.0454 | M30 bullish FVG [155.9500, 157.9100], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 22 | 2025-10-17 10:00 | 165.6100 | 160.1854 | 2025-10-17 14:00 | 168.1000 | STOCH_EXIT | 8 | 1463.58 | 1.40% | 163.2600 | 170.1305 | 4.5205 | M30 bullish FVG [158.2200, 163.2600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 98513.37
- Макс: 106687.75
- Финал: 106622.19

Полная кривая — в `*_equity.csv`.
