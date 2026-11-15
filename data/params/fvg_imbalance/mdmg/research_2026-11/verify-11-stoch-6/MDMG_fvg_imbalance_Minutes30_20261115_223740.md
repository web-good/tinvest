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
| StochPeriod | 9 |
| UseStochExit | 1 |
| StopATR | 2 |
| TakeProfitATR | 1 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 1 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 36 |
| Выигрышных / проигрышных | 28 / 8 |
| Win rate | 77.78% |
| Gross profit / loss | 25210.62 / 14779.72 |
| Profit factor | 1.706 |
| Чистый PnL | 10430.90 (10.43%) |
| Стартовый / финальный капитал | 100000.00 / 110430.90 |
| Макс. просадка | 10693.44 (10.12%) |
| Средняя прибыль / убыток | 900.38 / 1847.46 |
| Expectancy | 289.75 |
| Лучшая / худшая сделка | 2646.79 / -8060.10 |
| Exposure | 1.75% |
| CAGR | 5.08% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-12-22 17:30 | 806.8000 | 744.6507 | 2023-12-22 18:30 | 810.0000 | STOCH_EXIT | 2 | 294.17 | 0.30% | 805.5000 | 837.8747 | 31.0747 | M30 bullish FVG [796.5000, 805.5000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 2 | 2023-12-25 15:00 | 820.6000 | 757.6185 | 2023-12-26 16:30 | 807.4000 | STOCH_EXIT | 22 | -1709.71 | -1.71% | 820.4000 | 852.0908 | 31.4908 | M30 bullish FVG [809.0000, 820.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 3 | 2024-02-06 18:00 | 986.6000 | 931.2224 | 2024-02-07 10:00 | 1014.2888 | TP | 3 | 2642.15 | 2.70% | 974.5000 | 1014.2888 | 27.6888 | M30 bullish FVG [946.3000, 974.5000], stochastic crossed below 20 | FVG target 1014.2888 |
| 4 | 2024-02-12 10:30 | 956.2000 | 888.0151 | 2024-02-12 16:30 | 959.5000 | STOCH_EXIT | 12 | 245.93 | 0.24% | 962.3000 | 990.2924 | 34.0924 | M30 bullish FVG [948.8000, 962.3000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 5 | 2024-02-16 12:00 | 993.9000 | 926.0656 | 2024-02-16 17:30 | 996.3000 | STOCH_EXIT | 11 | 143.30 | 0.14% | 990.1000 | 1027.8172 | 33.9172 | M30 bullish FVG [981.8000, 990.1000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 6 | 2024-02-28 15:30 | 965.3000 | 894.9372 | 2024-02-29 11:30 | 971.1000 | STOCH_EXIT | 11 | 507.34 | 0.50% | 976.1000 | 1000.4814 | 35.1814 | M30 bullish FVG [958.7000, 976.1000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 7 | 2024-03-06 15:30 | 1023.7000 | 952.1980 | 2024-03-06 16:30 | 1032.1000 | STOCH_EXIT | 2 | 729.84 | 0.72% | 1026.8000 | 1059.4510 | 35.7510 | M30 bullish FVG [1007.3000, 1026.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 8 | 2024-04-02 10:30 | 1017.9000 | 956.6328 | 2024-04-02 14:00 | 1029.0000 | STOCH_EXIT | 7 | 1007.66 | 0.99% | 1020.2000 | 1048.5336 | 30.6336 | M30 bullish FVG [1009.0000, 1020.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 9 | 2024-04-08 18:30 | 1025.2000 | 957.0208 | 2024-04-10 09:30 | 1017.6000 | STOCH_EXIT | 20 | -870.76 | -0.84% | 1029.1000 | 1059.2896 | 34.0896 | M30 bullish FVG [1020.0000, 1029.1000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 10 | 2024-05-03 13:00 | 1136.9000 | 1072.2341 | 2024-05-03 15:00 | 1153.0000 | STOCH_EXIT | 4 | 1345.95 | 1.31% | 1133.9000 | 1169.2330 | 32.3330 | M30 bullish FVG [1115.0000, 1133.9000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 11 | 2024-06-17 15:00 | 1187.0000 | 1095.4964 | 2024-06-17 16:00 | 1095.4964 | SL | 2 | -8060.10 | -7.80% | 1271.7000 | 1232.7518 | 45.7518 | M30 bullish FVG [1179.2000, 1271.7000], stochastic crossed below 20 | FVG stop 1095.4964 |
| 12 | 2024-06-18 14:00 | 1003.6000 | 868.5752 | 2024-06-19 09:30 | 1027.0000 | STOCH_EXIT | 10 | 2126.55 | 2.23% | 1021.7000 | 1071.1124 | 67.5124 | M30 bullish FVG [1000.0000, 1021.7000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 13 | 2024-07-31 15:00 | 875.4000 | 808.0710 | 2024-07-31 18:00 | 874.2000 | STOCH_EXIT | 6 | -232.38 | -0.24% | 840.0000 | 909.0645 | 33.6645 | M30 bullish FVG [830.6000, 840.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 14 | 2024-08-01 15:00 | 881.5000 | 808.1088 | 2024-08-01 17:00 | 889.1000 | STOCH_EXIT | 4 | 745.33 | 0.76% | 890.3000 | 918.1956 | 36.6956 | M30 bullish FVG [880.0000, 890.3000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 15 | 2024-09-26 16:00 | 880.0000 | 831.6017 | 2024-09-26 20:00 | 883.6000 | STOCH_EXIT | 8 | 304.44 | 0.31% | 880.9000 | 904.1991 | 24.1991 | M30 bullish FVG [873.5000, 880.9000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 16 | 2024-09-30 16:00 | 902.3000 | 855.4066 | 2024-09-30 19:00 | 908.9000 | STOCH_EXIT | 6 | 620.69 | 0.63% | 907.0000 | 925.7467 | 23.4467 | M30 bullish FVG [896.9000, 907.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 17 | 2024-10-30 14:30 | 862.8000 | 815.0193 | 2024-10-30 14:30 | 886.6903 | TP | 0 | 2646.79 | 2.67% | 869.0000 | 886.6903 | 23.8903 | M30 bullish FVG [843.2000, 869.0000], stochastic crossed below 20 | FVG target 886.6903 |
| 18 | 2025-01-09 18:00 | 930.9000 | 880.1670 | 2025-01-09 19:00 | 930.9000 | STOCH_EXIT | 2 | -102.40 | -0.10% | 930.9000 | 956.2665 | 25.3665 | M30 bullish FVG [923.6000, 930.9000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 19 | 2025-01-22 16:00 | 948.0000 | 900.5274 | 2025-01-23 11:00 | 971.7363 | TP | 24 | 2437.08 | 2.40% | 951.6000 | 971.7363 | 23.7363 | M30 bullish FVG [944.6000, 951.6000], stochastic crossed below 20 | FVG target 971.7363 |
| 20 | 2025-01-27 17:30 | 991.7000 | 941.0520 | 2025-01-28 08:30 | 1003.1000 | STOCH_EXIT | 16 | 1092.27 | 1.05% | 992.0000 | 1017.0240 | 25.3240 | M30 bullish FVG [979.8000, 992.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 21 | 2025-02-19 23:30 | 1024.6000 | 964.3265 | 2025-02-20 12:00 | 1036.5000 | STOCH_EXIT | 11 | 1119.55 | 1.06% | 1021.3000 | 1054.7367 | 30.1367 | M30 bullish FVG [1013.0000, 1021.3000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 22 | 2025-03-20 23:30 | 1085.0000 | 1029.9158 | 2025-03-21 10:30 | 1094.0000 | STOCH_EXIT | 9 | 775.23 | 0.73% | 1085.2000 | 1112.5421 | 27.5421 | M30 bullish FVG [1070.0000, 1085.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 23 | 2025-03-21 18:30 | 1114.0000 | 1051.5361 | 2025-03-22 03:00 | 1138.0000 | STOCH_EXIT | 13 | 2195.90 | 2.05% | 1134.1000 | 1145.2320 | 31.2320 | M30 bullish FVG [1109.0000, 1134.1000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 24 | 2025-03-28 15:30 | 1061.4000 | 982.0921 | 2025-03-28 23:30 | 1060.3000 | STOCH_EXIT | 16 | -222.57 | -0.20% | 1058.0000 | 1101.0539 | 39.6539 | M30 bullish FVG [1041.9000, 1058.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 25 | 2025-04-10 17:00 | 997.3000 | 919.7040 | 2025-04-10 20:30 | 1001.7000 | STOCH_EXIT | 7 | 374.06 | 0.34% | 988.4000 | 1036.0980 | 38.7980 | M30 bullish FVG [977.0000, 988.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 26 | 2025-05-16 07:00 | 985.2000 | 943.6263 | 2025-05-16 08:00 | 994.8000 | STOCH_EXIT | 2 | 955.71 | 0.87% | 986.9000 | 1005.9869 | 20.7869 | M30 bullish FVG [980.8000, 986.9000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 27 | 2025-05-27 10:30 | 968.9000 | 924.3498 | 2025-05-27 16:30 | 973.3000 | STOCH_EXIT | 12 | 390.89 | 0.35% | 972.9000 | 991.1751 | 22.2751 | M30 bullish FVG [961.9000, 972.9000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 28 | 2025-05-27 22:00 | 977.7000 | 933.1498 | 2025-05-28 08:30 | 982.2000 | STOCH_EXIT | 8 | 397.77 | 0.36% | 972.2000 | 999.9751 | 22.2751 | M30 bullish FVG [965.2000, 972.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 29 | 2025-06-25 09:00 | 998.8000 | 968.6875 | 2025-06-25 17:30 | 998.9000 | STOCH_EXIT | 17 | -99.77 | -0.09% | 1003.0000 | 1013.8562 | 15.0562 | M30 bullish FVG [997.5000, 1003.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 30 | 2025-06-26 21:30 | 1005.6000 | 975.3956 | 2025-06-27 09:00 | 1009.0000 | STOCH_EXIT | 9 | 265.59 | 0.24% | 1011.0000 | 1020.7022 | 15.1022 | M30 bullish FVG [1005.0000, 1011.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 31 | 2025-07-03 20:30 | 1061.4000 | 1029.2832 | 2025-07-04 07:00 | 1029.2832 | SL | 8 | -3482.03 | -3.12% | 1046.0000 | 1077.4584 | 16.0584 | M30 bullish FVG [1040.6000, 1046.0000], stochastic crossed below 20 | FVG stop 1029.2832 |
| 32 | 2025-07-07 21:00 | 1049.3000 | 1008.3483 | 2025-07-08 15:30 | 1058.9000 | STOCH_EXIT | 24 | 880.23 | 0.81% | 1053.0000 | 1069.7759 | 20.4759 | M30 bullish FVG [1046.2000, 1053.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 33 | 2025-07-17 14:00 | 1110.6000 | 1057.2410 | 2025-07-18 10:30 | 1114.4000 | STOCH_EXIT | 27 | 263.38 | 0.24% | 1105.5000 | 1137.2795 | 26.6795 | M30 bullish FVG [1098.0000, 1105.5000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 34 | 2025-07-23 07:30 | 1133.2000 | 1076.2079 | 2025-07-23 10:00 | 1137.5000 | STOCH_EXIT | 5 | 303.81 | 0.28% | 1113.3000 | 1161.6960 | 28.4960 | M30 bullish FVG [1105.0000, 1113.3000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 35 | 2025-08-15 19:30 | 1260.0000 | 1188.8960 | 2025-08-15 23:30 | 1261.4000 | STOCH_EXIT | 8 | 12.12 | 0.01% | 1253.8000 | 1295.5520 | 35.5520 | M30 bullish FVG [1242.5000, 1253.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 36 | 2025-10-09 20:30 | 1249.9000 | 1186.8471 | 2025-10-09 21:30 | 1255.6000 | STOCH_EXIT | 2 | 386.91 | 0.36% | 1245.3000 | 1281.4265 | 31.5265 | M30 bullish FVG [1236.4000, 1245.3000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 94974.08
- Макс: 112252.27
- Финал: 110430.90

Полная кривая — в `*_equity.csv`.
