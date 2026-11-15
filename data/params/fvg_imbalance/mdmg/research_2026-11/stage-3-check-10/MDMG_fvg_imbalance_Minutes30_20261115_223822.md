# Бэктест MDMG (Minutes30)

- Период: 2023-11-15 22:38 — 2025-11-15 22:38
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.1 |
| MaxWaitBars | 30 |
| StochPeriod | 21 |
| UseStochExit | 1 |
| StopATR | 1 |
| TakeProfitATR | 0 |
| HTFTrendEMA | 50 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 75 |
| Выигрышных / проигрышных | 51 / 24 |
| Win rate | 68.00% |
| Gross profit / loss | 75070.46 / 52757.88 |
| Profit factor | 1.423 |
| Чистый PnL | 22312.59 (22.31%) |
| Стартовый / финальный капитал | 100000.00 / 122312.59 |
| Макс. просадка | 12610.16 (9.65%) |
| Средняя прибыль / убыток | 1471.97 / 2198.24 |
| Expectancy | 297.50 |
| Лучшая / худшая сделка | 11094.08 / -4776.67 |
| Exposure | 8.42% |
| CAGR | 10.58% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-12-21 12:00 | 788.5000 | 757.0581 | 2023-12-22 10:00 | 796.6000 | STOCH_EXIT | 15 | 920.74 | 0.93% | 797.6000 | 0.0000 | 31.4419 | M30 bullish FVG [794.0000, 797.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 2 | 2023-12-26 11:30 | 811.0000 | 780.5514 | 2023-12-27 14:00 | 807.9000 | STOCH_EXIT | 24 | -484.77 | -0.48% | 820.4000 | 0.0000 | 30.4486 | M30 bullish FVG [809.0000, 820.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 3 | 2024-01-08 18:30 | 808.1000 | 779.4447 | 2024-01-10 11:00 | 837.1000 | STOCH_EXIT | 23 | 3494.00 | 3.49% | 818.1000 | 0.0000 | 28.6553 | M30 bullish FVG [815.1000, 818.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 4 | 2024-01-22 13:00 | 800.0000 | 774.9856 | 2024-01-22 16:00 | 808.0000 | STOCH_EXIT | 6 | 928.28 | 0.90% | 804.9000 | 0.0000 | 25.0144 | M30 bullish FVG [800.0000, 804.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 5 | 2024-01-23 17:00 | 807.7000 | 783.1152 | 2024-01-25 10:00 | 818.9000 | STOCH_EXIT | 24 | 1339.88 | 1.29% | 814.0000 | 0.0000 | 24.5848 | M30 bullish FVG [809.0000, 814.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 6 | 2024-02-07 11:30 | 979.0000 | 948.2318 | 2024-02-08 10:30 | 948.2318 | SL | 17 | -3427.04 | -3.24% | 974.5000 | 0.0000 | 30.7682 | M30 bullish FVG [946.3000, 974.5000], stochastic crossed below 20 | FVG stop 948.2318 |
| 7 | 2024-02-12 12:30 | 954.3000 | 920.2076 | 2024-02-14 09:30 | 967.0000 | STOCH_EXIT | 29 | 1256.11 | 1.23% | 978.5000 | 0.0000 | 34.0924 | M30 bullish FVG [975.0000, 978.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 8 | 2024-02-16 14:30 | 993.8000 | 959.8828 | 2024-02-19 14:00 | 1015.4000 | STOCH_EXIT | 18 | 2141.92 | 2.07% | 990.1000 | 0.0000 | 33.9172 | M30 bullish FVG [981.8000, 990.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 9 | 2024-02-20 10:30 | 984.2000 | 949.2337 | 2024-02-21 10:00 | 949.2337 | SL | 18 | -3844.84 | -3.65% | 1008.7000 | 0.0000 | 34.9663 | M30 bullish FVG [1003.0000, 1008.7000], stochastic crossed below 20 | FVG stop 949.2337 |
| 10 | 2024-02-28 17:30 | 958.1000 | 922.9186 | 2024-03-01 15:00 | 983.4000 | STOCH_EXIT | 33 | 2578.90 | 2.54% | 991.8000 | 0.0000 | 35.1814 | M30 bullish FVG [978.9000, 991.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 11 | 2024-03-07 13:00 | 1015.7000 | 978.9098 | 2024-03-07 14:00 | 1036.3000 | STOCH_EXIT | 2 | 2016.12 | 1.93% | 1026.8000 | 0.0000 | 36.7902 | M30 bullish FVG [1007.3000, 1026.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 12 | 2024-03-11 17:30 | 1025.6000 | 989.1734 | 2024-03-14 10:00 | 989.1734 | SL | 42 | -3893.14 | -3.65% | 1030.9000 | 0.0000 | 36.4266 | M30 bullish FVG [1016.6000, 1030.9000], stochastic crossed below 20 | FVG stop 989.1734 |
| 13 | 2024-03-20 12:00 | 1001.6000 | 967.5142 | 2024-03-21 11:30 | 1008.9000 | STOCH_EXIT | 18 | 642.06 | 0.63% | 1013.0000 | 0.0000 | 34.0858 | M30 bullish FVG [1008.7000, 1013.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 14 | 2024-03-22 16:30 | 1001.7000 | 968.7015 | 2024-03-29 12:30 | 988.2000 | STOCH_EXIT | 87 | -1492.98 | -1.45% | 1008.2000 | 0.0000 | 32.9985 | M30 bullish FVG [1002.0000, 1008.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 15 | 2024-04-04 13:30 | 1041.0000 | 1008.9511 | 2024-04-04 17:30 | 1008.9511 | SL | 8 | -3241.24 | -3.18% | 1070.0000 | 0.0000 | 32.0489 | M30 bullish FVG [1060.0000, 1070.0000], stochastic crossed below 20 | FVG stop 1008.9511 |
| 16 | 2024-04-05 13:00 | 1012.9000 | 977.7189 | 2024-04-06 10:00 | 1026.0000 | STOCH_EXIT | 12 | 1171.81 | 1.19% | 1019.3000 | 0.0000 | 35.1811 | M30 bullish FVG [1012.6000, 1019.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 17 | 2024-04-08 12:30 | 1020.0000 | 985.9104 | 2024-04-08 15:00 | 1032.3000 | STOCH_EXIT | 5 | 1104.84 | 1.10% | 1029.1000 | 0.0000 | 34.0896 | M30 bullish FVG [1020.0000, 1029.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 18 | 2024-04-09 12:30 | 1012.9000 | 979.8311 | 2024-04-11 12:30 | 1006.5000 | STOCH_EXIT | 38 | -733.56 | -0.73% | 1029.1000 | 0.0000 | 33.0689 | M30 bullish FVG [1020.0000, 1029.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 19 | 2024-04-23 12:30 | 1046.4000 | 1017.5777 | 2024-04-24 13:30 | 1050.6000 | STOCH_EXIT | 21 | 299.39 | 0.30% | 1057.0000 | 0.0000 | 28.8223 | M30 bullish FVG [1053.8000, 1057.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 20 | 2024-05-03 13:30 | 1136.7000 | 1104.3670 | 2024-05-04 15:00 | 1154.8000 | STOCH_EXIT | 21 | 1491.97 | 1.49% | 1133.9000 | 0.0000 | 32.3330 | M30 bullish FVG [1115.0000, 1133.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 21 | 2024-05-15 13:00 | 1179.1000 | 1144.8545 | 2024-05-15 15:00 | 1144.8545 | SL | 4 | -3045.04 | -3.00% | 1194.9000 | 0.0000 | 34.2455 | M30 bullish FVG [1188.5000, 1194.9000], stochastic crossed below 20 | FVG stop 1144.8545 |
| 22 | 2024-05-17 11:30 | 1064.1000 | 1022.1516 | 2024-05-17 18:00 | 1088.1000 | STOCH_EXIT | 13 | 2131.92 | 2.15% | 1074.0000 | 0.0000 | 41.9484 | M30 bullish FVG [1056.7000, 1074.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 23 | 2024-05-20 14:30 | 1061.3000 | 1019.2408 | 2024-06-17 09:30 | 1179.2000 | STOCH_EXIT | 47 | 11094.08 | 11.00% | 1074.0000 | 0.0000 | 42.0592 | M30 bullish FVG [1056.7000, 1074.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 24 | 2024-06-17 16:30 | 1101.2000 | 1055.4482 | 2024-06-17 18:00 | 1055.4482 | SL | 3 | -4776.67 | -4.25% | 1271.7000 | 0.0000 | 45.7518 | M30 bullish FVG [1179.2000, 1271.7000], stochastic crossed below 20 | FVG stop 1055.4482 |
| 25 | 2024-09-24 10:30 | 861.7000 | 838.1602 | 2024-09-25 09:30 | 873.8000 | STOCH_EXIT | 32 | 1392.80 | 1.30% | 879.3000 | 0.0000 | 23.5398 | M30 bullish FVG [876.7000, 879.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 26 | 2024-09-26 22:30 | 875.6000 | 851.4009 | 2024-09-27 11:00 | 886.7000 | STOCH_EXIT | 11 | 1267.14 | 1.17% | 880.9000 | 0.0000 | 24.1991 | M30 bullish FVG [873.5000, 880.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 27 | 2024-09-27 21:00 | 884.1000 | 860.3651 | 2024-09-28 13:00 | 891.0000 | STOCH_EXIT | 18 | 745.54 | 0.68% | 882.4000 | 0.0000 | 23.7349 | M30 bullish FVG [880.0000, 882.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 28 | 2024-10-02 18:30 | 886.8000 | 861.7745 | 2024-10-03 15:30 | 861.7745 | SL | 28 | -3237.47 | -2.92% | 905.0000 | 0.0000 | 25.0255 | M30 bullish FVG [895.8000, 905.0000], stochastic crossed below 20 | FVG stop 861.7745 |
| 29 | 2024-10-04 13:00 | 870.9000 | 844.8112 | 2024-10-05 07:00 | 876.4000 | STOCH_EXIT | 22 | 569.04 | 0.53% | 879.0000 | 0.0000 | 26.0888 | M30 bullish FVG [873.9000, 879.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 30 | 2024-10-07 11:30 | 875.2000 | 849.5104 | 2024-10-08 13:00 | 869.6000 | STOCH_EXIT | 37 | -796.11 | -0.74% | 881.3000 | 0.0000 | 25.6896 | M30 bullish FVG [877.0000, 881.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 31 | 2024-11-13 20:00 | 877.1000 | 849.4862 | 2024-11-14 13:00 | 881.9000 | STOCH_EXIT | 20 | 478.30 | 0.45% | 882.9000 | 0.0000 | 27.6138 | M30 bullish FVG [879.5000, 882.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 32 | 2024-11-21 18:00 | 864.6000 | 836.9635 | 2024-11-22 10:00 | 873.3000 | STOCH_EXIT | 18 | 971.05 | 0.91% | 873.0000 | 0.0000 | 27.6365 | M30 bullish FVG [869.5000, 873.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 33 | 2024-12-27 11:30 | 852.5000 | 833.7478 | 2024-12-27 23:00 | 856.7000 | STOCH_EXIT | 23 | 424.87 | 0.39% | 865.0000 | 0.0000 | 18.7522 | M30 bullish FVG [857.7000, 865.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 34 | 2025-01-10 12:00 | 912.9000 | 886.7883 | 2025-01-10 14:00 | 934.6000 | STOCH_EXIT | 4 | 2472.37 | 2.27% | 930.9000 | 0.0000 | 26.1117 | M30 bullish FVG [923.6000, 930.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 35 | 2025-01-27 23:00 | 991.4000 | 966.0760 | 2025-01-28 18:30 | 1010.0000 | STOCH_EXIT | 25 | 1971.12 | 1.77% | 1008.6000 | 0.0000 | 25.3240 | M30 bullish FVG [1001.9000, 1008.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 36 | 2025-02-10 23:00 | 1002.6000 | 971.9427 | 2025-02-12 07:00 | 1014.7000 | STOCH_EXIT | 36 | 1253.32 | 1.11% | 1015.0000 | 0.0000 | 30.6573 | M30 bullish FVG [1011.0000, 1015.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 37 | 2025-02-27 12:00 | 1001.2000 | 973.5301 | 2025-02-27 17:30 | 1027.8000 | STOCH_EXIT | 11 | 2916.75 | 2.55% | 1031.4000 | 0.0000 | 27.6699 | M30 bullish FVG [1027.5000, 1031.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 38 | 2025-02-28 21:30 | 1008.6000 | 980.9208 | 2025-03-01 12:30 | 1020.5000 | STOCH_EXIT | 26 | 1273.60 | 1.08% | 1022.0000 | 0.0000 | 27.6792 | M30 bullish FVG [1015.4000, 1022.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 39 | 2025-03-04 15:30 | 1036.2000 | 1008.1888 | 2025-03-05 11:00 | 1052.0000 | STOCH_EXIT | 25 | 1696.93 | 1.42% | 1051.3000 | 0.0000 | 28.0112 | M30 bullish FVG [1048.0000, 1051.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 40 | 2025-03-06 21:30 | 1060.9000 | 1033.8388 | 2025-03-07 17:00 | 1033.8388 | SL | 25 | -3204.37 | -2.65% | 1063.1000 | 0.0000 | 27.0612 | M30 bullish FVG [1058.0000, 1063.1000], stochastic crossed below 20 | FVG stop 1033.8388 |
| 41 | 2025-03-12 19:00 | 1027.6000 | 999.4371 | 2025-03-12 23:00 | 1040.6000 | STOCH_EXIT | 8 | 1364.11 | 1.16% | 1035.5000 | 0.0000 | 28.1629 | M30 bullish FVG [1032.0000, 1035.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 42 | 2025-03-13 11:00 | 1017.0000 | 989.2058 | 2025-03-14 10:00 | 1033.8000 | STOCH_EXIT | 32 | 1845.63 | 1.55% | 1035.5000 | 0.0000 | 27.7942 | M30 bullish FVG [1032.0000, 1035.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 43 | 2025-03-27 10:30 | 1067.5000 | 1027.0650 | 2025-03-28 10:00 | 1059.0000 | STOCH_EXIT | 34 | -1080.65 | -0.90% | 1075.1000 | 0.0000 | 40.4350 | M30 bullish FVG [1069.0000, 1075.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 44 | 2025-03-28 21:30 | 1050.9000 | 1011.2461 | 2025-03-29 02:30 | 1066.9000 | STOCH_EXIT | 6 | 1703.29 | 1.42% | 1058.0000 | 0.0000 | 39.6539 | M30 bullish FVG [1041.9000, 1058.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 45 | 2025-03-31 11:30 | 1037.3000 | 997.7714 | 2025-04-01 06:30 | 1027.7000 | STOCH_EXIT | 25 | -1244.00 | -1.02% | 1076.0000 | 0.0000 | 39.5286 | M30 bullish FVG [1067.3000, 1076.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 46 | 2025-06-13 11:00 | 985.9000 | 968.5222 | 2025-06-14 08:30 | 981.1000 | STOCH_EXIT | 39 | -705.59 | -0.59% | 996.3000 | 0.0000 | 17.3778 | M30 bullish FVG [994.0000, 996.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 47 | 2025-06-18 16:30 | 984.1000 | 967.5854 | 2025-06-18 19:00 | 992.3000 | STOCH_EXIT | 5 | 872.63 | 0.73% | 993.0000 | 0.0000 | 16.5146 | M30 bullish FVG [991.0000, 993.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 48 | 2025-06-24 10:30 | 991.6000 | 976.5625 | 2025-06-24 13:00 | 997.5000 | STOCH_EXIT | 5 | 593.56 | 0.49% | 995.0000 | 0.0000 | 15.0375 | M30 bullish FVG [993.3000, 995.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 49 | 2025-06-25 09:00 | 998.8000 | 983.7438 | 2025-06-25 19:30 | 1000.8000 | STOCH_EXIT | 21 | 121.02 | 0.10% | 1003.0000 | 0.0000 | 15.0562 | M30 bullish FVG [997.5000, 1003.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 50 | 2025-06-26 23:00 | 1004.0000 | 988.8978 | 2025-06-27 11:00 | 1013.6000 | STOCH_EXIT | 10 | 1030.94 | 0.86% | 1011.0000 | 0.0000 | 15.1022 | M30 bullish FVG [1005.0000, 1011.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 51 | 2025-07-04 07:30 | 1031.6000 | 1013.4029 | 2025-07-04 11:30 | 1063.1000 | STOCH_EXIT | 8 | 3593.41 | 2.95% | 1072.0000 | 0.0000 | 18.1971 | M30 bullish FVG [1062.0000, 1072.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 52 | 2025-07-04 20:00 | 1046.6000 | 1028.4029 | 2025-07-05 08:30 | 1056.1000 | STOCH_EXIT | 20 | 1013.84 | 0.81% | 1050.4000 | 0.0000 | 18.1971 | M30 bullish FVG [1047.6000, 1050.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 53 | 2025-07-08 08:30 | 1047.3000 | 1025.6938 | 2025-07-08 17:00 | 1060.6000 | STOCH_EXIT | 17 | 1481.77 | 1.17% | 1053.0000 | 0.0000 | 21.6062 | M30 bullish FVG [1046.2000, 1053.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 54 | 2025-07-08 22:00 | 1049.8000 | 1028.1938 | 2025-07-09 23:30 | 1039.7000 | STOCH_EXIT | 38 | -1359.66 | -1.06% | 1058.9000 | 0.0000 | 21.6062 | M30 bullish FVG [1056.1000, 1058.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 55 | 2025-07-11 09:00 | 1059.1000 | 1036.8908 | 2025-07-11 10:30 | 1069.9000 | STOCH_EXIT | 3 | 1158.52 | 0.92% | 1061.9000 | 0.0000 | 22.2092 | M30 bullish FVG [1059.2000, 1061.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 56 | 2025-07-11 14:30 | 1058.0000 | 1035.7908 | 2025-07-11 18:30 | 1035.7908 | SL | 8 | -2813.99 | -2.20% | 1066.4000 | 0.0000 | 22.2092 | M30 bullish FVG [1062.0000, 1066.4000], stochastic crossed below 20 | FVG stop 1035.7908 |
| 57 | 2025-07-17 18:00 | 1103.4000 | 1076.7205 | 2025-07-18 10:30 | 1114.4000 | STOCH_EXIT | 19 | 1117.69 | 0.90% | 1105.5000 | 0.0000 | 26.6795 | M30 bullish FVG [1098.0000, 1105.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 58 | 2025-07-25 18:00 | 1119.0000 | 1091.1412 | 2025-07-26 02:00 | 1138.3000 | STOCH_EXIT | 12 | 2053.36 | 1.62% | 1138.2000 | 0.0000 | 27.8588 | M30 bullish FVG [1132.9000, 1138.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 59 | 2025-07-28 18:30 | 1146.0000 | 1118.0525 | 2025-07-29 09:30 | 1156.5000 | STOCH_EXIT | 16 | 1047.06 | 0.82% | 1158.8000 | 0.0000 | 27.9475 | M30 bullish FVG [1150.4000, 1158.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 60 | 2025-08-14 18:00 | 1249.6000 | 1214.6440 | 2025-08-15 10:00 | 1257.3000 | STOCH_EXIT | 19 | 663.99 | 0.52% | 1254.8000 | 0.0000 | 34.9560 | M30 bullish FVG [1250.0000, 1254.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 61 | 2025-08-15 21:30 | 1257.4000 | 1221.8480 | 2025-08-16 02:00 | 1221.8480 | SL | 5 | -3789.54 | -2.92% | 1253.8000 | 0.0000 | 35.5520 | M30 bullish FVG [1242.5000, 1253.8000], stochastic crossed below 20 | FVG stop 1221.8480 |
| 62 | 2025-08-20 14:00 | 1280.3000 | 1246.5232 | 2025-08-21 17:00 | 1246.5232 | SL | 41 | -3433.94 | -2.74% | 1299.0000 | 0.0000 | 33.7768 | M30 bullish FVG [1295.0000, 1299.0000], stochastic crossed below 20 | FVG stop 1246.5232 |
| 63 | 2025-09-05 15:00 | 1244.8000 | 1213.3631 | 2025-09-06 15:00 | 1246.6000 | STOCH_EXIT | 42 | 54.32 | 0.04% | 1251.1000 | 0.0000 | 31.4369 | M30 bullish FVG [1246.5000, 1251.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 64 | 2025-09-10 22:00 | 1272.8000 | 1243.0272 | 2025-09-12 07:00 | 1265.6000 | STOCH_EXIT | 39 | -813.04 | -0.67% | 1273.5000 | 0.0000 | 29.7728 | M30 bullish FVG [1267.2000, 1273.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 65 | 2025-09-16 13:00 | 1234.8000 | 1204.4087 | 2025-09-17 07:30 | 1241.1000 | STOCH_EXIT | 24 | 501.14 | 0.41% | 1245.2000 | 0.0000 | 30.3913 | M30 bullish FVG [1238.7000, 1245.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 66 | 2025-09-17 13:00 | 1232.2000 | 1202.1509 | 2025-09-18 12:00 | 1219.7000 | STOCH_EXIT | 32 | -1358.87 | -1.11% | 1235.4000 | 0.0000 | 30.0491 | M30 bullish FVG [1230.0000, 1235.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 67 | 2025-09-25 14:00 | 1236.0000 | 1204.5717 | 2025-09-27 19:30 | 1239.0000 | STOCH_EXIT | 87 | 172.73 | 0.14% | 1234.8000 | 0.0000 | 31.4283 | M30 bullish FVG [1231.6000, 1234.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 68 | 2025-09-29 17:00 | 1227.1000 | 1197.1965 | 2025-09-30 10:30 | 1225.0000 | STOCH_EXIT | 22 | -329.28 | -0.27% | 1241.1000 | 0.0000 | 29.9035 | M30 bullish FVG [1234.6000, 1241.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 69 | 2025-10-02 09:30 | 1240.2000 | 1208.8687 | 2025-10-03 11:00 | 1247.8000 | STOCH_EXIT | 38 | 616.53 | 0.51% | 1258.1000 | 0.0000 | 31.3313 | M30 bullish FVG [1252.1000, 1258.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 70 | 2025-10-06 19:00 | 1244.4000 | 1213.4158 | 2025-10-06 23:00 | 1257.7000 | STOCH_EXIT | 8 | 1168.75 | 0.97% | 1257.0000 | 0.0000 | 30.9842 | M30 bullish FVG [1252.0000, 1257.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 71 | 2025-10-14 12:00 | 1269.0000 | 1233.5500 | 2025-10-14 18:00 | 1233.5500 | SL | 12 | -3523.32 | -2.89% | 1285.5000 | 0.0000 | 35.4500 | M30 bullish FVG [1280.0000, 1285.5000], stochastic crossed below 20 | FVG stop 1233.5500 |
| 72 | 2025-10-20 22:00 | 1241.4000 | 1201.3258 | 2025-10-22 07:30 | 1241.3000 | STOCH_EXIT | 41 | -128.77 | -0.11% | 1241.4000 | 0.0000 | 40.0742 | M30 bullish FVG [1233.0000, 1241.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 73 | 2025-11-05 17:30 | 1276.6000 | 1240.0952 | 2025-11-06 06:30 | 1304.9000 | STOCH_EXIT | 13 | 2511.86 | 2.11% | 1297.1000 | 0.0000 | 36.5048 | M30 bullish FVG [1284.9000, 1297.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 74 | 2025-11-06 14:00 | 1289.5000 | 1252.6813 | 2025-11-07 07:00 | 1294.1000 | STOCH_EXIT | 21 | 310.97 | 0.26% | 1297.4000 | 0.0000 | 36.8187 | M30 bullish FVG [1292.9000, 1297.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 75 | 2025-11-13 14:30 | 1296.5000 | 1261.8246 | 2025-11-14 07:30 | 1298.1000 | STOCH_EXIT | 21 | 28.45 | 0.02% | 1311.8000 | 0.0000 | 34.6754 | M30 bullish FVG [1306.1000, 1311.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 98104.46
- Макс: 130671.84
- Финал: 122312.59

Полная кривая — в `*_equity.csv`.
