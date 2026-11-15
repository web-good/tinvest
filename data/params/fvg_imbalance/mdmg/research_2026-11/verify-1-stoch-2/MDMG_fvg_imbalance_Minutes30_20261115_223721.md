# Бэктест MDMG (Minutes30)

- Период: 2023-11-15 22:37 — 2025-11-15 22:37
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.25 |
| MaxWaitBars | 30 |
| StochPeriod | 21 |
| UseStochExit | 1 |
| StopATR | 2 |
| TakeProfitATR | 0 |
| HTFTrendEMA | 50 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 28 |
| Выигрышных / проигрышных | 20 / 8 |
| Win rate | 71.43% |
| Gross profit / loss | 42225.22 / 18972.67 |
| Profit factor | 2.226 |
| Чистый PnL | 23252.56 (23.25%) |
| Стартовый / финальный капитал | 100000.00 / 123252.56 |
| Макс. просадка | 12317.73 (10.51%) |
| Средняя прибыль / убыток | 2111.26 / 2371.58 |
| Expectancy | 830.45 |
| Лучшая / худшая сделка | 11561.20 / -10838.56 |
| Exposure | 3.07% |
| CAGR | 11.00% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-12-26 11:30 | 811.0000 | 750.1029 | 2023-12-27 14:00 | 807.9000 | STOCH_EXIT | 24 | -480.86 | -0.48% | 820.4000 | 0.0000 | 30.4486 | M30 bullish FVG [809.0000, 820.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 2 | 2024-02-07 11:30 | 979.0000 | 917.4636 | 2024-02-09 12:00 | 969.4000 | STOCH_EXIT | 39 | -1067.99 | -1.08% | 974.5000 | 0.0000 | 30.7682 | M30 bullish FVG [946.3000, 974.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 3 | 2024-02-12 12:30 | 954.3000 | 886.1151 | 2024-02-14 09:30 | 967.0000 | STOCH_EXIT | 29 | 1209.15 | 1.23% | 962.3000 | 0.0000 | 34.0924 | M30 bullish FVG [948.8000, 962.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 4 | 2024-02-16 14:30 | 993.8000 | 925.9656 | 2024-02-19 14:00 | 1015.4000 | STOCH_EXIT | 18 | 2059.54 | 2.07% | 990.1000 | 0.0000 | 33.9172 | M30 bullish FVG [981.8000, 990.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 5 | 2024-02-28 17:30 | 958.1000 | 887.7372 | 2024-03-01 15:00 | 983.4000 | STOCH_EXIT | 33 | 2578.90 | 2.54% | 991.8000 | 0.0000 | 35.1814 | M30 bullish FVG [978.9000, 991.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 6 | 2024-03-07 13:00 | 1015.7000 | 942.1195 | 2024-03-07 14:00 | 1036.3000 | STOCH_EXIT | 2 | 1996.55 | 1.93% | 1026.8000 | 0.0000 | 36.7902 | M30 bullish FVG [1007.3000, 1026.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 7 | 2024-03-11 17:30 | 1025.6000 | 952.7467 | 2024-03-18 10:30 | 995.4000 | STOCH_EXIT | 81 | -3214.68 | -3.04% | 1030.9000 | 0.0000 | 36.4266 | M30 bullish FVG [1016.6000, 1030.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 8 | 2024-04-04 13:30 | 1041.0000 | 976.9023 | 2024-04-06 10:00 | 1026.0000 | STOCH_EXIT | 30 | -1571.28 | -1.54% | 1070.0000 | 0.0000 | 32.0489 | M30 bullish FVG [1060.0000, 1070.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 9 | 2024-04-08 12:30 | 1020.0000 | 951.8208 | 2024-04-08 15:00 | 1032.3000 | STOCH_EXIT | 5 | 1116.11 | 1.10% | 1029.1000 | 0.0000 | 34.0896 | M30 bullish FVG [1020.0000, 1029.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 10 | 2024-04-09 12:30 | 1012.9000 | 946.7622 | 2024-04-11 12:30 | 1006.5000 | STOCH_EXIT | 38 | -748.38 | -0.73% | 1029.1000 | 0.0000 | 33.0689 | M30 bullish FVG [1020.0000, 1029.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 11 | 2024-05-03 13:30 | 1136.7000 | 1072.0341 | 2024-05-04 15:00 | 1154.8000 | STOCH_EXIT | 21 | 1508.93 | 1.49% | 1133.9000 | 0.0000 | 32.3330 | M30 bullish FVG [1115.0000, 1133.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 12 | 2024-05-17 11:30 | 1064.1000 | 980.2032 | 2024-05-17 18:00 | 1088.1000 | STOCH_EXIT | 13 | 2223.62 | 2.15% | 1074.0000 | 0.0000 | 41.9484 | M30 bullish FVG [1056.7000, 1074.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 13 | 2024-05-20 14:30 | 1061.3000 | 977.1815 | 2024-06-17 09:30 | 1179.2000 | STOCH_EXIT | 47 | 11561.20 | 11.00% | 1074.0000 | 0.0000 | 42.0592 | M30 bullish FVG [1056.7000, 1074.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 14 | 2024-06-17 16:30 | 1101.2000 | 1009.6964 | 2024-06-18 09:30 | 1000.0000 | SL | 5 | -10838.56 | -9.28% | 1271.7000 | 0.0000 | 45.7518 | M30 bullish FVG [1179.2000, 1271.7000], stochastic crossed below 20 | FVG stop 1009.6964 |
| 15 | 2024-09-26 22:30 | 875.6000 | 827.2017 | 2024-09-27 11:00 | 886.7000 | STOCH_EXIT | 11 | 1236.48 | 1.17% | 880.9000 | 0.0000 | 24.1991 | M30 bullish FVG [873.5000, 880.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 16 | 2024-10-02 18:30 | 886.8000 | 836.7491 | 2024-10-04 07:30 | 881.9000 | STOCH_EXIT | 46 | -699.91 | -0.65% | 905.0000 | 0.0000 | 25.0255 | M30 bullish FVG [895.8000, 905.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 17 | 2024-12-27 11:30 | 852.5000 | 814.9957 | 2024-12-27 23:00 | 856.7000 | STOCH_EXIT | 23 | 418.18 | 0.39% | 865.0000 | 0.0000 | 18.7522 | M30 bullish FVG [857.7000, 865.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 18 | 2025-01-10 12:00 | 912.9000 | 860.6765 | 2025-01-10 14:00 | 934.6000 | STOCH_EXIT | 4 | 2430.82 | 2.27% | 930.9000 | 0.0000 | 26.1117 | M30 bullish FVG [923.6000, 930.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 19 | 2025-01-27 23:00 | 991.4000 | 940.7520 | 2025-01-28 18:30 | 1010.0000 | STOCH_EXIT | 25 | 1935.92 | 1.77% | 1008.6000 | 0.0000 | 25.3240 | M30 bullish FVG [1001.9000, 1008.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 20 | 2025-03-28 21:30 | 1050.9000 | 971.5921 | 2025-03-29 02:30 | 1066.9000 | STOCH_EXIT | 6 | 1583.76 | 1.42% | 1058.0000 | 0.0000 | 39.6539 | M30 bullish FVG [1041.9000, 1058.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 21 | 2025-06-25 09:00 | 998.8000 | 968.6875 | 2025-06-25 19:30 | 1000.8000 | STOCH_EXIT | 21 | 113.02 | 0.10% | 1003.0000 | 0.0000 | 15.0562 | M30 bullish FVG [997.5000, 1003.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 22 | 2025-06-26 23:00 | 1004.0000 | 973.7956 | 2025-06-27 11:00 | 1013.6000 | STOCH_EXIT | 10 | 962.21 | 0.86% | 1011.0000 | 0.0000 | 15.1022 | M30 bullish FVG [1005.0000, 1011.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 23 | 2025-07-04 07:30 | 1031.6000 | 995.2058 | 2025-07-04 11:30 | 1063.1000 | STOCH_EXIT | 8 | 3349.79 | 2.95% | 1072.0000 | 0.0000 | 18.1971 | M30 bullish FVG [1062.0000, 1072.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 24 | 2025-07-08 08:30 | 1047.3000 | 1004.0877 | 2025-07-08 17:00 | 1060.6000 | STOCH_EXIT | 17 | 1371.56 | 1.17% | 1053.0000 | 0.0000 | 21.6062 | M30 bullish FVG [1046.2000, 1053.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 25 | 2025-07-17 18:00 | 1103.4000 | 1050.0410 | 2025-07-18 10:30 | 1114.4000 | STOCH_EXIT | 19 | 1058.35 | 0.90% | 1105.5000 | 0.0000 | 26.6795 | M30 bullish FVG [1098.0000, 1105.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 26 | 2025-07-28 18:30 | 1146.0000 | 1090.1050 | 2025-07-29 09:30 | 1156.5000 | STOCH_EXIT | 16 | 972.27 | 0.82% | 1158.8000 | 0.0000 | 27.9475 | M30 bullish FVG [1150.4000, 1158.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 27 | 2025-08-15 21:30 | 1257.4000 | 1186.2960 | 2025-08-16 03:30 | 1255.0000 | STOCH_EXIT | 8 | -351.00 | -0.29% | 1253.8000 | 0.0000 | 35.5520 | M30 bullish FVG [1242.5000, 1253.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 28 | 2025-11-05 17:30 | 1276.6000 | 1203.5904 | 2025-11-06 06:30 | 1304.9000 | STOCH_EXIT | 13 | 2538.87 | 2.11% | 1297.1000 | 0.0000 | 36.5048 | M30 bullish FVG [1284.9000, 1297.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 95227.70
- Макс: 123252.56
- Финал: 123252.56

Полная кривая — в `*_equity.csv`.
