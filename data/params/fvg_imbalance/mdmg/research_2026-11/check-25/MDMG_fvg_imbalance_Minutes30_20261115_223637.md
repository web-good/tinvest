# Бэктест MDMG (Minutes30)

- Период: 2023-11-15 22:36 — 2025-11-15 22:36
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
| HTFTrendEMA | 100 |
| RequireEntryAboveFVGLower | 1 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 30 |
| Выигрышных / проигрышных | 22 / 8 |
| Win rate | 73.33% |
| Gross profit / loss | 18260.78 / 11377.45 |
| Profit factor | 1.605 |
| Чистый PnL | 6883.32 (6.88%) |
| Стартовый / финальный капитал | 100000.00 / 106883.32 |
| Макс. просадка | 7131.91 (7.09%) |
| Средняя прибыль / убыток | 830.04 / 1422.18 |
| Expectancy | 229.44 |
| Лучшая / худшая сделка | 3186.06 / -3894.02 |
| Exposure | 1.66% |
| CAGR | 3.38% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-12-22 17:30 | 806.8000 | 775.7253 | 2023-12-22 18:30 | 810.0000 | STOCH_EXIT | 2 | 294.17 | 0.30% | 805.5000 | 868.9493 | 31.0747 | M30 bullish FVG [796.5000, 805.5000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 2 | 2023-12-25 15:00 | 820.6000 | 789.1092 | 2023-12-26 16:30 | 807.4000 | STOCH_EXIT | 22 | -1709.71 | -1.71% | 820.4000 | 883.5815 | 31.4908 | M30 bullish FVG [809.0000, 820.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 3 | 2024-02-06 18:00 | 986.6000 | 958.9112 | 2024-02-08 10:00 | 958.9112 | SL | 22 | -2837.50 | -2.90% | 974.5000 | 1041.9776 | 27.6888 | M30 bullish FVG [946.3000, 974.5000], stochastic crossed below 20 | FVG stop 958.9112 |
| 4 | 2024-02-12 10:30 | 956.2000 | 922.1076 | 2024-02-12 16:30 | 959.5000 | STOCH_EXIT | 12 | 234.21 | 0.24% | 962.3000 | 1024.3849 | 34.0924 | M30 bullish FVG [948.8000, 962.3000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 5 | 2024-02-16 12:00 | 993.9000 | 959.9828 | 2024-02-16 17:30 | 996.3000 | STOCH_EXIT | 11 | 134.87 | 0.14% | 990.1000 | 1061.7344 | 33.9172 | M30 bullish FVG [981.8000, 990.1000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 6 | 2024-02-28 15:30 | 965.3000 | 930.1186 | 2024-02-29 11:30 | 971.1000 | STOCH_EXIT | 11 | 478.35 | 0.50% | 976.1000 | 1035.6628 | 35.1814 | M30 bullish FVG [958.7000, 976.1000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 7 | 2024-03-06 15:30 | 1023.7000 | 987.9490 | 2024-03-06 16:30 | 1032.1000 | STOCH_EXIT | 2 | 692.98 | 0.72% | 1026.8000 | 1095.2020 | 35.7510 | M30 bullish FVG [1007.3000, 1026.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 8 | 2024-04-02 10:30 | 1017.9000 | 987.2664 | 2024-04-02 14:00 | 1029.0000 | STOCH_EXIT | 7 | 957.27 | 0.99% | 1020.2000 | 1079.1672 | 30.6336 | M30 bullish FVG [1009.0000, 1020.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 9 | 2024-04-08 18:30 | 1025.2000 | 991.1104 | 2024-04-10 09:30 | 1017.6000 | STOCH_EXIT | 20 | -819.03 | -0.84% | 1029.1000 | 1093.3792 | 34.0896 | M30 bullish FVG [1020.0000, 1029.1000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 10 | 2024-05-03 13:00 | 1136.9000 | 1104.5670 | 2024-05-03 15:00 | 1153.0000 | STOCH_EXIT | 4 | 1271.18 | 1.31% | 1133.9000 | 1201.5659 | 32.3330 | M30 bullish FVG [1115.0000, 1133.9000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 11 | 2024-06-17 15:00 | 1187.0000 | 1141.2482 | 2024-06-17 16:00 | 1141.2482 | SL | 2 | -3894.02 | -3.95% | 1271.7000 | 1278.5036 | 45.7518 | M30 bullish FVG [1179.2000, 1271.7000], stochastic crossed below 20 | FVG stop 1141.2482 |
| 12 | 2024-06-18 14:00 | 1003.6000 | 936.0876 | 2024-06-19 09:30 | 1027.0000 | STOCH_EXIT | 10 | 2104.16 | 2.23% | 1021.7000 | 1138.6248 | 67.5124 | M30 bullish FVG [1000.0000, 1021.7000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 13 | 2024-09-30 16:00 | 902.3000 | 878.8533 | 2024-09-30 19:00 | 908.9000 | STOCH_EXIT | 6 | 609.30 | 0.63% | 907.0000 | 949.1934 | 23.4467 | M30 bullish FVG [896.9000, 907.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 14 | 2025-01-09 18:00 | 930.9000 | 905.5335 | 2025-01-09 19:00 | 930.9000 | STOCH_EXIT | 2 | -96.81 | -0.10% | 930.9000 | 981.6330 | 25.3665 | M30 bullish FVG [923.6000, 930.9000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 15 | 2025-01-22 16:00 | 948.0000 | 924.2637 | 2025-01-23 11:00 | 980.2000 | STOCH_EXIT | 24 | 3186.06 | 3.29% | 951.6000 | 995.4726 | 23.7363 | M30 bullish FVG [944.6000, 951.6000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 16 | 2025-01-27 17:30 | 991.7000 | 966.3760 | 2025-01-28 08:30 | 1003.1000 | STOCH_EXIT | 16 | 1050.66 | 1.05% | 992.0000 | 1042.3480 | 25.3240 | M30 bullish FVG [979.8000, 992.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 17 | 2025-02-19 23:30 | 1024.6000 | 994.4633 | 2025-02-20 12:00 | 1036.5000 | STOCH_EXIT | 11 | 1076.08 | 1.06% | 1021.3000 | 1084.8735 | 30.1367 | M30 bullish FVG [1013.0000, 1021.3000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 18 | 2025-03-20 23:30 | 1085.0000 | 1057.4579 | 2025-03-21 10:30 | 1094.0000 | STOCH_EXIT | 9 | 743.59 | 0.73% | 1085.2000 | 1140.0842 | 27.5421 | M30 bullish FVG [1070.0000, 1085.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 19 | 2025-03-21 18:30 | 1114.0000 | 1082.7680 | 2025-03-22 03:00 | 1138.0000 | STOCH_EXIT | 13 | 2104.41 | 2.05% | 1134.1000 | 1176.4639 | 31.2320 | M30 bullish FVG [1109.0000, 1134.1000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 20 | 2025-03-28 15:30 | 1061.4000 | 1021.7461 | 2025-03-28 23:30 | 1060.3000 | STOCH_EXIT | 16 | -213.92 | -0.20% | 1058.0000 | 1140.7079 | 39.6539 | M30 bullish FVG [1041.9000, 1058.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 21 | 2025-04-10 17:00 | 997.3000 | 958.5020 | 2025-04-10 20:30 | 1001.7000 | STOCH_EXIT | 7 | 357.05 | 0.34% | 988.4000 | 1074.8960 | 38.7980 | M30 bullish FVG [977.0000, 988.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 22 | 2025-05-16 07:00 | 985.2000 | 964.4131 | 2025-05-16 08:00 | 994.8000 | STOCH_EXIT | 2 | 921.27 | 0.87% | 986.9000 | 1026.7737 | 20.7869 | M30 bullish FVG [980.8000, 986.9000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 23 | 2025-06-25 09:00 | 998.8000 | 983.7438 | 2025-06-25 17:30 | 998.9000 | STOCH_EXIT | 17 | -95.28 | -0.09% | 1003.0000 | 1028.9125 | 15.0562 | M30 bullish FVG [997.5000, 1003.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 24 | 2025-06-26 21:30 | 1005.6000 | 990.4978 | 2025-06-27 09:00 | 1009.0000 | STOCH_EXIT | 9 | 251.23 | 0.24% | 1011.0000 | 1035.8044 | 15.1022 | M30 bullish FVG [1005.0000, 1011.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 25 | 2025-07-03 20:30 | 1061.4000 | 1045.3416 | 2025-07-04 07:00 | 1045.3416 | SL | 8 | -1711.18 | -1.61% | 1046.0000 | 1093.5168 | 16.0584 | M30 bullish FVG [1040.6000, 1046.0000], stochastic crossed below 20 | FVG stop 1045.3416 |
| 26 | 2025-07-07 21:00 | 1049.3000 | 1028.8241 | 2025-07-08 15:30 | 1058.9000 | STOCH_EXIT | 24 | 854.59 | 0.81% | 1053.0000 | 1090.2517 | 20.4759 | M30 bullish FVG [1046.2000, 1053.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 27 | 2025-07-17 14:00 | 1110.6000 | 1083.9205 | 2025-07-18 10:30 | 1114.4000 | STOCH_EXIT | 27 | 255.31 | 0.24% | 1105.5000 | 1163.9590 | 26.6795 | M30 bullish FVG [1098.0000, 1105.5000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 28 | 2025-07-23 07:30 | 1133.2000 | 1104.7040 | 2025-07-23 10:00 | 1137.5000 | STOCH_EXIT | 5 | 294.31 | 0.28% | 1113.3000 | 1190.1921 | 28.4960 | M30 bullish FVG [1105.0000, 1113.3000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 29 | 2025-08-15 19:30 | 1260.0000 | 1224.4480 | 2025-08-15 23:30 | 1261.4000 | STOCH_EXIT | 8 | 11.70 | 0.01% | 1253.8000 | 1331.1040 | 35.5520 | M30 bullish FVG [1242.5000, 1253.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 30 | 2025-10-09 20:30 | 1249.9000 | 1218.3735 | 2025-10-09 21:30 | 1255.6000 | STOCH_EXIT | 2 | 378.02 | 0.36% | 1245.3000 | 1312.9529 | 31.5265 | M30 bullish FVG [1236.4000, 1245.3000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 93514.80
- Макс: 106977.50
- Финал: 106883.32

Полная кривая — в `*_equity.csv`.
