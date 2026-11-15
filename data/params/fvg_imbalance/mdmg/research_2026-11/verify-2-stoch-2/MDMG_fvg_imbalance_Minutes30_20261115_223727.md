# Бэктест MDMG (Minutes30)

- Период: 2023-11-15 22:37 — 2025-11-15 22:37
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0 |
| MaxWaitBars | 30 |
| StochPeriod | 21 |
| UseStochExit | 1 |
| StopATR | 1.7 |
| TakeProfitATR | 1 |
| HTFTrendEMA | 50 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 155 |
| Выигрышных / проигрышных | 113 / 42 |
| Win rate | 72.90% |
| Gross profit / loss | 153266.24 / 81110.00 |
| Profit factor | 1.890 |
| Чистый PnL | 72156.23 (72.16%) |
| Стартовый / финальный капитал | 100000.00 / 172156.23 |
| Макс. просадка | 13327.52 (7.71%) |
| Средняя прибыль / убыток | 1356.34 / 1931.19 |
| Expectancy | 465.52 |
| Лучшая / худшая сделка | 4739.65 / -10838.56 |
| Exposure | 17.66% |
| CAGR | 31.16% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-12-21 12:00 | 788.5000 | 735.0487 | 2023-12-22 10:00 | 796.6000 | STOCH_EXIT | 15 | 920.74 | 0.93% | 797.6000 | 819.9419 | 31.4419 | M30 bullish FVG [794.0000, 797.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 2 | 2023-12-26 11:30 | 811.0000 | 759.2375 | 2023-12-27 14:00 | 807.9000 | STOCH_EXIT | 24 | -484.77 | -0.48% | 820.4000 | 841.4486 | 30.4486 | M30 bullish FVG [809.0000, 820.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 3 | 2023-12-28 11:30 | 793.8000 | 742.1598 | 2024-01-03 09:30 | 792.6000 | STOCH_EXIT | 34 | -251.14 | -0.25% | 803.2000 | 824.1766 | 30.3766 | M30 bullish FVG [801.0000, 803.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 4 | 2024-01-05 11:30 | 806.5000 | 756.2618 | 2024-01-08 10:00 | 822.9000 | STOCH_EXIT | 16 | 1932.58 | 1.93% | 803.5000 | 836.0519 | 29.5519 | M30 bullish FVG [802.8000, 803.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 5 | 2024-01-08 18:30 | 808.1000 | 759.3859 | 2024-01-09 12:00 | 836.7553 | TP | 6 | 3506.95 | 3.44% | 818.1000 | 836.7553 | 28.6553 | M30 bullish FVG [815.1000, 818.1000], stochastic crossed below 20 | FVG target 836.7553 |
| 6 | 2024-01-16 10:30 | 817.0000 | 769.8433 | 2024-01-17 16:00 | 817.0000 | STOCH_EXIT | 30 | -105.39 | -0.10% | 825.1000 | 844.7392 | 27.7392 | M30 bullish FVG [825.0000, 825.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 7 | 2024-01-22 13:00 | 800.0000 | 757.4755 | 2024-01-22 16:00 | 808.0000 | STOCH_EXIT | 6 | 942.68 | 0.90% | 804.9000 | 825.0144 | 25.0144 | M30 bullish FVG [800.0000, 804.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 8 | 2024-01-23 17:00 | 807.7000 | 765.9058 | 2024-01-25 10:00 | 818.9000 | STOCH_EXIT | 24 | 1360.66 | 1.29% | 814.0000 | 832.2848 | 24.5848 | M30 bullish FVG [809.0000, 814.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 9 | 2024-02-05 12:00 | 883.3000 | 841.5397 | 2024-02-05 14:00 | 907.8649 | TP | 4 | 2887.66 | 2.68% | 891.9000 | 907.8649 | 24.5649 | M30 bullish FVG [891.8000, 891.9000], stochastic crossed below 20 | FVG target 907.8649 |
| 10 | 2024-02-07 11:30 | 979.0000 | 926.6941 | 2024-02-09 12:00 | 969.4000 | STOCH_EXIT | 39 | -1194.88 | -1.08% | 992.2000 | 1009.7682 | 30.7682 | M30 bullish FVG [990.2000, 992.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 11 | 2024-02-12 12:30 | 954.3000 | 896.3429 | 2024-02-14 09:30 | 967.0000 | STOCH_EXIT | 29 | 1338.29 | 1.23% | 978.5000 | 988.3924 | 34.0924 | M30 bullish FVG [975.0000, 978.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 12 | 2024-02-16 14:30 | 993.8000 | 936.1408 | 2024-02-19 14:00 | 1015.4000 | STOCH_EXIT | 18 | 2286.09 | 2.07% | 1011.3000 | 1027.7172 | 33.9172 | M30 bullish FVG [1009.8000, 1011.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 13 | 2024-02-20 10:30 | 984.2000 | 924.7572 | 2024-02-21 10:00 | 924.7572 | SL | 18 | -6885.29 | -6.13% | 1012.9000 | 1019.1663 | 34.9663 | M30 bullish FVG [1009.4000, 1012.9000], stochastic crossed below 20 | FVG stop 924.7572 |
| 14 | 2024-02-28 17:30 | 958.1000 | 898.2916 | 2024-02-29 12:30 | 993.2814 | TP | 9 | 3762.63 | 3.57% | 991.8000 | 993.2814 | 35.1814 | M30 bullish FVG [978.9000, 991.8000], stochastic crossed below 20 | FVG target 993.2814 |
| 15 | 2024-03-07 13:00 | 1015.7000 | 953.1566 | 2024-03-07 14:00 | 1036.3000 | STOCH_EXIT | 2 | 2113.99 | 1.93% | 1028.3000 | 1052.4902 | 36.7902 | M30 bullish FVG [1026.0000, 1028.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 16 | 2024-03-11 17:30 | 1025.6000 | 963.6747 | 2024-03-18 10:30 | 995.4000 | STOCH_EXIT | 81 | -3401.94 | -3.04% | 1030.9000 | 1062.0266 | 36.4266 | M30 bullish FVG [1016.6000, 1030.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 17 | 2024-03-20 12:00 | 1001.6000 | 943.6541 | 2024-03-21 11:30 | 1008.9000 | STOCH_EXIT | 18 | 679.83 | 0.63% | 1013.0000 | 1035.6858 | 34.0858 | M30 bullish FVG [1008.7000, 1013.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 18 | 2024-03-22 16:30 | 1001.7000 | 945.6026 | 2024-03-29 12:30 | 988.2000 | STOCH_EXIT | 87 | -1579.95 | -1.45% | 1013.4000 | 1034.6985 | 32.9985 | M30 bullish FVG [1012.9000, 1013.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 19 | 2024-04-04 13:30 | 1041.0000 | 986.5169 | 2024-04-06 10:00 | 1026.0000 | STOCH_EXIT | 30 | -1651.45 | -1.54% | 1070.0000 | 1073.0489 | 32.0489 | M30 bullish FVG [1060.0000, 1070.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 20 | 2024-04-08 10:00 | 1018.5000 | 960.5477 | 2024-04-08 15:00 | 1032.3000 | STOCH_EXIT | 10 | 1328.56 | 1.25% | 1025.1000 | 1052.5896 | 34.0896 | M30 bullish FVG [1025.0000, 1025.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 21 | 2024-04-09 12:30 | 1012.9000 | 956.6829 | 2024-04-11 12:30 | 1006.5000 | STOCH_EXIT | 38 | -785.43 | -0.73% | 1026.0000 | 1045.9689 | 33.0689 | M30 bullish FVG [1025.0000, 1026.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 22 | 2024-04-17 14:30 | 1036.3000 | 984.7014 | 2024-04-17 18:30 | 1047.8000 | STOCH_EXIT | 8 | 1066.71 | 1.01% | 1047.3000 | 1066.6521 | 30.3521 | M30 bullish FVG [1046.5000, 1047.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 23 | 2024-04-19 12:00 | 1038.7000 | 990.6880 | 2024-04-20 12:00 | 1049.3000 | STOCH_EXIT | 18 | 984.27 | 0.92% | 1045.2000 | 1066.9424 | 28.2424 | M30 bullish FVG [1044.1000, 1045.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 24 | 2024-04-23 12:30 | 1046.4000 | 997.4022 | 2024-04-24 13:30 | 1050.6000 | STOCH_EXIT | 21 | 324.60 | 0.30% | 1057.0000 | 1075.2223 | 28.8223 | M30 bullish FVG [1053.8000, 1057.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 25 | 2024-04-30 10:30 | 1104.1000 | 1053.2070 | 2024-05-01 10:00 | 1134.0371 | TP | 17 | 2824.16 | 2.61% | 1115.5000 | 1134.0371 | 29.9371 | M30 bullish FVG [1114.5000, 1115.5000], stochastic crossed below 20 | FVG target 1134.0371 |
| 26 | 2024-05-01 14:30 | 1100.8000 | 1050.1058 | 2024-05-01 22:30 | 1111.0000 | STOCH_EXIT | 16 | 918.50 | 0.83% | 1100.0000 | 1130.6201 | 29.8201 | M30 bullish FVG [1098.2000, 1100.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 27 | 2024-05-03 13:30 | 1136.7000 | 1081.7340 | 2024-05-04 15:00 | 1154.8000 | STOCH_EXIT | 21 | 1678.47 | 1.49% | 1133.9000 | 1169.0330 | 32.3330 | M30 bullish FVG [1115.0000, 1133.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 28 | 2024-05-07 11:00 | 1122.4000 | 1066.5596 | 2024-05-08 15:30 | 1138.5000 | STOCH_EXIT | 28 | 1511.92 | 1.33% | 1133.0000 | 1155.2473 | 32.8473 | M30 bullish FVG [1130.3000, 1133.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 29 | 2024-05-15 13:00 | 1179.1000 | 1120.8826 | 2024-05-15 18:00 | 1120.8826 | SL | 10 | -5818.00 | -5.03% | 1194.9000 | 1213.3455 | 34.2455 | M30 bullish FVG [1188.5000, 1194.9000], stochastic crossed below 20 | FVG stop 1120.8826 |
| 30 | 2024-05-16 16:00 | 1102.7000 | 1036.5467 | 2024-05-17 18:00 | 1088.1000 | STOCH_EXIT | 23 | -1553.84 | -1.42% | 1118.4000 | 1141.6137 | 38.9137 | M30 bullish FVG [1115.0000, 1118.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 31 | 2024-05-20 14:30 | 1061.3000 | 989.7993 | 2024-05-21 10:00 | 1103.3592 | TP | 10 | 4179.65 | 3.86% | 1064.9000 | 1103.3592 | 42.0592 | M30 bullish FVG [1064.8000, 1064.9000], stochastic crossed below 20 | FVG target 1103.3592 |
| 32 | 2024-05-22 16:00 | 1070.1000 | 991.5044 | 2024-06-17 09:30 | 1116.3327 | TP | 6 | 4739.65 | 4.22% | 1088.3000 | 1116.3327 | 46.2327 | M30 bullish FVG [1087.9000, 1088.3000], stochastic crossed below 20 | FVG target 1116.3327 |
| 33 | 2024-06-17 16:30 | 1101.2000 | 1023.4219 | 2024-06-18 09:30 | 1000.0000 | SL | 5 | -10838.56 | -9.28% | 1271.7000 | 1146.9518 | 45.7518 | M30 bullish FVG [1179.2000, 1271.7000], stochastic crossed below 20 | FVG stop 1023.4219 |
| 34 | 2024-09-23 10:00 | 873.0000 | 833.4609 | 2024-09-23 17:00 | 887.0000 | STOCH_EXIT | 14 | 1600.64 | 1.50% | 873.6000 | 896.2583 | 23.2583 | M30 bullish FVG [873.5000, 873.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 35 | 2024-09-24 10:30 | 861.7000 | 821.6823 | 2024-09-25 09:30 | 873.8000 | STOCH_EXIT | 32 | 1404.03 | 1.30% | 883.2000 | 885.2398 | 23.5398 | M30 bullish FVG [882.9000, 883.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 36 | 2024-09-25 23:30 | 857.6000 | 816.2393 | 2024-09-26 07:00 | 870.3000 | STOCH_EXIT | 1 | 1503.18 | 1.38% | 860.5000 | 881.9298 | 24.3298 | M30 bullish FVG [859.0000, 860.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 37 | 2024-09-26 22:30 | 875.6000 | 834.4615 | 2024-09-27 11:00 | 886.7000 | STOCH_EXIT | 11 | 1287.58 | 1.17% | 879.6000 | 899.7991 | 24.1991 | M30 bullish FVG [877.7000, 879.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 38 | 2024-09-27 21:00 | 884.1000 | 843.7506 | 2024-09-28 13:00 | 891.0000 | STOCH_EXIT | 18 | 763.58 | 0.68% | 890.4000 | 907.8349 | 23.7349 | M30 bullish FVG [888.9000, 890.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 39 | 2024-10-02 18:30 | 886.8000 | 844.2567 | 2024-10-04 07:30 | 881.9000 | STOCH_EXIT | 46 | -734.61 | -0.65% | 904.1000 | 911.8255 | 25.0255 | M30 bullish FVG [903.8000, 904.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 40 | 2024-10-04 13:00 | 870.9000 | 826.5490 | 2024-10-05 07:00 | 876.4000 | STOCH_EXIT | 22 | 596.80 | 0.53% | 882.0000 | 896.9888 | 26.0888 | M30 bullish FVG [880.0000, 882.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 41 | 2024-10-07 11:30 | 875.2000 | 831.5277 | 2024-10-08 13:00 | 869.6000 | STOCH_EXIT | 37 | -834.94 | -0.74% | 881.3000 | 900.8896 | 25.6896 | M30 bullish FVG [877.0000, 881.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 42 | 2024-10-10 11:00 | 868.8000 | 827.6462 | 2024-10-11 16:30 | 875.8000 | STOCH_EXIT | 45 | 790.47 | 0.70% | 870.9000 | 893.0081 | 24.2081 | M30 bullish FVG [870.7000, 870.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 43 | 2024-10-11 22:00 | 870.7000 | 830.7493 | 2024-10-12 23:00 | 868.8000 | STOCH_EXIT | 36 | -357.30 | -0.32% | 872.0000 | 894.2004 | 23.5004 | M30 bullish FVG [871.0000, 872.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 44 | 2024-10-14 08:00 | 869.8000 | 831.5615 | 2024-10-14 09:30 | 873.0000 | STOCH_EXIT | 3 | 300.39 | 0.27% | 870.0000 | 892.2932 | 22.4932 | M30 bullish FVG [869.8000, 870.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 45 | 2024-10-16 10:00 | 882.0000 | 845.2301 | 2024-10-16 11:00 | 891.6000 | STOCH_EXIT | 2 | 1115.29 | 0.99% | 888.5000 | 903.6294 | 21.6294 | M30 bullish FVG [887.0000, 888.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 46 | 2024-10-16 19:30 | 883.2000 | 846.4301 | 2024-10-17 08:30 | 888.9000 | STOCH_EXIT | 12 | 621.00 | 0.54% | 886.3000 | 904.8294 | 21.6294 | M30 bullish FVG [885.4000, 886.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 47 | 2024-10-17 13:00 | 881.9000 | 845.9958 | 2024-10-19 07:00 | 868.3000 | STOCH_EXIT | 56 | -1881.76 | -1.64% | 888.0000 | 903.0201 | 21.1201 | M30 bullish FVG [887.7000, 888.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 48 | 2024-10-31 17:00 | 855.4000 | 800.8452 | 2024-11-01 10:30 | 860.1000 | STOCH_EXIT | 21 | 503.33 | 0.45% | 860.6000 | 887.4910 | 32.0910 | M30 bullish FVG [860.0000, 860.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 49 | 2024-11-13 20:00 | 877.1000 | 830.1565 | 2024-11-14 13:00 | 881.9000 | STOCH_EXIT | 20 | 505.74 | 0.45% | 882.9000 | 904.7138 | 27.6138 | M30 bullish FVG [879.5000, 882.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 50 | 2024-11-14 21:00 | 869.4000 | 824.6439 | 2024-11-15 12:30 | 882.8000 | STOCH_EXIT | 17 | 1628.11 | 1.44% | 880.5000 | 895.7271 | 26.3271 | M30 bullish FVG [879.7000, 880.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 51 | 2024-11-19 12:30 | 875.3000 | 828.8291 | 2024-11-20 10:00 | 884.3000 | STOCH_EXIT | 29 | 1063.75 | 0.93% | 894.6000 | 902.6358 | 27.3358 | M30 bullish FVG [893.8000, 894.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 52 | 2024-11-20 18:30 | 867.5000 | 819.9770 | 2024-11-21 10:00 | 877.1000 | STOCH_EXIT | 17 | 1169.51 | 1.01% | 885.3000 | 895.4547 | 27.9547 | M30 bullish FVG [885.0000, 885.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 53 | 2024-11-21 18:00 | 864.6000 | 817.6179 | 2024-11-22 10:00 | 873.3000 | STOCH_EXIT | 18 | 1065.02 | 0.91% | 873.0000 | 892.2365 | 27.6365 | M30 bullish FVG [869.5000, 873.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 54 | 2024-11-22 20:30 | 867.1000 | 821.1424 | 2024-11-23 14:00 | 874.8000 | STOCH_EXIT | 21 | 928.75 | 0.79% | 857.5000 | 894.1339 | 27.0339 | M30 bullish FVG [857.1000, 857.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 55 | 2024-11-25 18:00 | 872.6000 | 828.2129 | 2024-11-27 14:00 | 868.4000 | STOCH_EXIT | 60 | -694.66 | -0.58% | 885.2000 | 898.7100 | 26.1100 | M30 bullish FVG [883.9000, 885.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 56 | 2024-12-23 15:00 | 868.8000 | 837.6939 | 2024-12-26 09:30 | 859.9000 | STOCH_EXIT | 91 | -1327.95 | -1.12% | 866.2000 | 887.0977 | 18.2977 | M30 bullish FVG [865.3000, 866.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 57 | 2024-12-27 11:30 | 852.5000 | 820.6213 | 2024-12-27 23:00 | 856.7000 | STOCH_EXIT | 23 | 461.67 | 0.39% | 865.0000 | 871.2522 | 18.7522 | M30 bullish FVG [857.7000, 865.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 58 | 2024-12-31 13:00 | 894.7000 | 859.9817 | 2024-12-31 19:00 | 900.9000 | STOCH_EXIT | 12 | 699.89 | 0.59% | 898.0000 | 915.1225 | 20.4225 | M30 bullish FVG [896.5000, 898.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 59 | 2025-01-07 15:00 | 940.9000 | 898.4875 | 2025-01-08 07:30 | 943.8000 | STOCH_EXIT | 19 | 246.66 | 0.21% | 944.8000 | 965.8485 | 24.9485 | M30 bullish FVG [944.6000, 944.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 60 | 2025-01-10 12:00 | 912.9000 | 868.5101 | 2025-01-10 14:00 | 934.6000 | STOCH_EXIT | 4 | 2700.91 | 2.27% | 930.9000 | 939.0117 | 26.1117 | M30 bullish FVG [923.6000, 930.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 61 | 2025-01-15 14:00 | 936.2000 | 892.3634 | 2025-01-15 22:00 | 946.9000 | STOCH_EXIT | 16 | 1268.60 | 1.04% | 940.2000 | 961.9862 | 25.7862 | M30 bullish FVG [937.6000, 940.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 62 | 2025-01-20 23:00 | 928.9000 | 889.8291 | 2025-01-21 19:30 | 940.8000 | STOCH_EXIT | 27 | 1447.40 | 1.18% | 938.5000 | 951.8829 | 22.9829 | M30 bullish FVG [936.5000, 938.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 63 | 2025-01-24 17:30 | 969.9000 | 925.8914 | 2025-01-25 09:30 | 973.5000 | STOCH_EXIT | 28 | 336.42 | 0.27% | 979.3000 | 995.7874 | 25.8874 | M30 bullish FVG [977.0000, 979.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 64 | 2025-01-27 23:00 | 991.4000 | 948.3492 | 2025-01-28 18:30 | 1010.0000 | STOCH_EXIT | 25 | 2199.91 | 1.77% | 1008.6000 | 1016.7240 | 25.3240 | M30 bullish FVG [1001.9000, 1008.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 65 | 2025-02-07 15:30 | 991.6000 | 942.1952 | 2025-02-08 02:00 | 1020.6617 | TP | 17 | 3591.11 | 2.83% | 1003.2000 | 1020.6617 | 29.0617 | M30 bullish FVG [1001.0000, 1003.2000], stochastic crossed below 20 | FVG target 1020.6617 |
| 66 | 2025-02-10 23:00 | 1002.6000 | 950.4827 | 2025-02-12 07:00 | 1014.7000 | STOCH_EXIT | 36 | 1441.88 | 1.11% | 1011.8000 | 1033.2573 | 30.6573 | M30 bullish FVG [1011.0000, 1011.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 67 | 2025-02-13 14:00 | 1026.4000 | 978.3694 | 2025-02-14 07:00 | 1054.6533 | TP | 20 | 3483.24 | 2.65% | 1022.1000 | 1054.6533 | 28.2533 | M30 bullish FVG [1022.0000, 1022.1000], stochastic crossed below 20 | FVG target 1054.6533 |
| 68 | 2025-02-14 14:00 | 1034.7000 | 983.4701 | 2025-02-15 03:00 | 1039.7000 | STOCH_EXIT | 22 | 519.13 | 0.38% | 1033.8000 | 1064.8352 | 30.1352 | M30 bullish FVG [1033.6000, 1033.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 69 | 2025-02-17 10:00 | 1036.2000 | 984.6830 | 2025-02-17 12:00 | 1049.6000 | STOCH_EXIT | 4 | 1618.78 | 1.19% | 1036.6000 | 1066.5041 | 30.3041 | M30 bullish FVG [1036.5000, 1036.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 70 | 2025-02-18 14:30 | 1030.0000 | 979.7828 | 2025-02-19 19:00 | 1028.8000 | STOCH_EXIT | 43 | -296.51 | -0.22% | 1029.2000 | 1059.5396 | 29.5396 | M30 bullish FVG [1026.9000, 1029.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 71 | 2025-02-21 21:00 | 1035.9000 | 986.1984 | 2025-02-22 03:00 | 1045.5000 | STOCH_EXIT | 7 | 1129.83 | 0.83% | 1039.0000 | 1065.1363 | 29.2363 | M30 bullish FVG [1036.6000, 1039.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 72 | 2025-02-27 12:00 | 1001.2000 | 954.1611 | 2025-02-27 17:30 | 1027.8000 | STOCH_EXIT | 11 | 3530.80 | 2.55% | 1031.4000 | 1028.8699 | 27.6699 | M30 bullish FVG [1027.5000, 1031.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 73 | 2025-02-28 21:30 | 1008.6000 | 961.5453 | 2025-03-01 12:30 | 1020.5000 | STOCH_EXIT | 26 | 1523.96 | 1.08% | 1017.8000 | 1036.2792 | 27.6792 | M30 bullish FVG [1016.3000, 1017.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 74 | 2025-03-04 15:30 | 1036.2000 | 988.5810 | 2025-03-05 11:00 | 1052.0000 | STOCH_EXIT | 25 | 2036.31 | 1.42% | 1051.3000 | 1064.2112 | 28.0112 | M30 bullish FVG [1048.0000, 1051.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 75 | 2025-03-05 22:30 | 1047.3000 | 999.7188 | 2025-03-06 09:00 | 1056.0000 | STOCH_EXIT | 7 | 1063.12 | 0.73% | 1051.7000 | 1075.2889 | 27.9889 | M30 bullish FVG [1050.8000, 1051.7000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 76 | 2025-03-06 21:30 | 1060.9000 | 1014.8960 | 2025-03-07 23:30 | 1060.1000 | STOCH_EXIT | 38 | -256.75 | -0.18% | 1072.5000 | 1087.9612 | 27.0612 | M30 bullish FVG [1070.6000, 1072.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 77 | 2025-03-11 13:00 | 1059.9000 | 1012.2632 | 2025-03-12 07:30 | 1053.9000 | STOCH_EXIT | 23 | -973.85 | -0.67% | 1067.0000 | 1087.9216 | 28.0216 | M30 bullish FVG [1066.9000, 1067.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 78 | 2025-03-12 19:00 | 1027.6000 | 979.7230 | 2025-03-12 23:00 | 1040.6000 | STOCH_EXIT | 8 | 1687.19 | 1.16% | 1035.5000 | 1055.7629 | 28.1629 | M30 bullish FVG [1032.0000, 1035.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 79 | 2025-03-13 11:00 | 1017.0000 | 969.7499 | 2025-03-14 10:00 | 1033.8000 | STOCH_EXIT | 32 | 2271.54 | 1.55% | 1040.0000 | 1044.7942 | 27.7942 | M30 bullish FVG [1039.7000, 1040.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 80 | 2025-03-17 10:00 | 1041.0000 | 994.5934 | 2025-03-18 08:30 | 1049.2000 | STOCH_EXIT | 31 | 1023.15 | 0.69% | 1043.0000 | 1068.2980 | 27.2980 | M30 bullish FVG [1042.2000, 1043.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 81 | 2025-03-19 11:30 | 1020.2000 | 971.7121 | 2025-03-19 12:30 | 1031.5000 | STOCH_EXIT | 2 | 1510.30 | 1.01% | 1025.0000 | 1048.7223 | 28.5223 | M30 bullish FVG [1024.1000, 1025.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 82 | 2025-03-19 19:00 | 1023.7000 | 975.2121 | 2025-03-20 08:00 | 1028.8000 | STOCH_EXIT | 12 | 602.91 | 0.40% | 1026.1000 | 1052.2223 | 28.5223 | M30 bullish FVG [1026.0000, 1026.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 83 | 2025-03-24 12:30 | 1117.5000 | 1054.1003 | 2025-03-25 08:30 | 1102.7000 | STOCH_EXIT | 27 | -2163.77 | -1.42% | 1126.1000 | 1154.7940 | 37.2940 | M30 bullish FVG [1123.0000, 1126.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 84 | 2025-03-25 13:00 | 1083.1000 | 1014.7574 | 2025-03-25 23:00 | 1083.0000 | STOCH_EXIT | 20 | -163.26 | -0.11% | 1105.1000 | 1123.3015 | 40.2015 | M30 bullish FVG [1103.5000, 1105.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 85 | 2025-03-26 18:00 | 1070.7000 | 1001.1190 | 2025-03-28 10:00 | 1059.0000 | STOCH_EXIT | 54 | -1787.08 | -1.19% | 1081.0000 | 1111.6300 | 40.9300 | M30 bullish FVG [1079.7000, 1081.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 86 | 2025-03-28 21:30 | 1050.9000 | 983.4883 | 2025-03-29 02:30 | 1066.9000 | STOCH_EXIT | 6 | 2106.70 | 1.42% | 1058.0000 | 1090.5539 | 39.6539 | M30 bullish FVG [1041.9000, 1058.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 87 | 2025-03-31 11:30 | 1037.3000 | 970.1013 | 2025-04-01 06:30 | 1027.7000 | STOCH_EXIT | 25 | -1541.71 | -1.02% | 1076.0000 | 1076.8286 | 39.5286 | M30 bullish FVG [1067.3000, 1076.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 88 | 2025-04-01 11:30 | 1013.8000 | 937.6919 | 2025-04-02 21:00 | 1005.5000 | STOCH_EXIT | 54 | -1359.21 | -0.92% | 1032.1000 | 1058.5695 | 44.7695 | M30 bullish FVG [1027.7000, 1032.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 89 | 2025-04-14 07:00 | 1013.8000 | 952.3618 | 2025-04-15 09:00 | 1003.3000 | STOCH_EXIT | 39 | -1668.74 | -1.13% | 1021.6000 | 1049.9401 | 36.1401 | M30 bullish FVG [1021.0000, 1021.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 90 | 2025-05-09 03:00 | 999.2000 | 963.1013 | 2025-05-09 07:30 | 1003.8000 | STOCH_EXIT | 9 | 525.38 | 0.36% | 999.2000 | 1020.4345 | 21.2345 | M30 bullish FVG [998.8000, 999.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 91 | 2025-05-09 16:00 | 998.1000 | 962.0013 | 2025-05-09 23:00 | 1002.0000 | STOCH_EXIT | 14 | 423.39 | 0.29% | 1000.0000 | 1019.3345 | 21.2345 | M30 bullish FVG [998.1000, 1000.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 92 | 2025-05-12 12:00 | 1008.2000 | 972.1013 | 2025-05-13 07:30 | 1003.7000 | STOCH_EXIT | 25 | -798.36 | -0.55% | 1006.7000 | 1029.4345 | 21.2345 | M30 bullish FVG [1006.0000, 1006.7000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 93 | 2025-05-13 12:00 | 998.8000 | 961.6733 | 2025-05-14 06:30 | 1000.0000 | STOCH_EXIT | 24 | 29.29 | 0.02% | 999.0000 | 1020.6392 | 21.8392 | M30 bullish FVG [998.9000, 999.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 94 | 2025-06-12 06:30 | 987.6000 | 958.0577 | 2025-06-12 21:00 | 992.4000 | STOCH_EXIT | 29 | 560.07 | 0.39% | 989.0000 | 1004.9778 | 17.3778 | M30 bullish FVG [988.8000, 989.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 95 | 2025-06-13 11:00 | 985.9000 | 956.3577 | 2025-06-14 08:30 | 981.1000 | STOCH_EXIT | 39 | -855.96 | -0.59% | 996.3000 | 1003.2778 | 17.3778 | M30 bullish FVG [994.0000, 996.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 96 | 2025-06-17 07:30 | 986.7000 | 957.4726 | 2025-06-17 13:30 | 990.5000 | STOCH_EXIT | 12 | 413.28 | 0.28% | 987.5000 | 1003.8926 | 17.1926 | M30 bullish FVG [987.2000, 987.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 97 | 2025-06-18 16:30 | 984.1000 | 956.0252 | 2025-06-18 19:00 | 992.3000 | STOCH_EXIT | 5 | 1067.35 | 0.73% | 993.0000 | 1000.6146 | 16.5146 | M30 bullish FVG [991.0000, 993.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 98 | 2025-06-19 14:00 | 990.7000 | 963.3556 | 2025-06-19 18:00 | 994.6000 | STOCH_EXIT | 8 | 430.29 | 0.29% | 991.0000 | 1006.7849 | 16.0849 | M30 bullish FVG [989.8000, 991.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 99 | 2025-06-20 19:30 | 993.4000 | 966.5516 | 2025-06-20 23:30 | 997.1000 | STOCH_EXIT | 8 | 400.30 | 0.27% | 996.3000 | 1009.1932 | 15.7932 | M30 bullish FVG [996.1000, 996.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 100 | 2025-06-24 10:30 | 991.6000 | 966.0363 | 2025-06-24 13:00 | 997.5000 | STOCH_EXIT | 5 | 730.91 | 0.49% | 995.0000 | 1006.6375 | 15.0375 | M30 bullish FVG [993.3000, 995.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 101 | 2025-06-25 07:00 | 997.5000 | 971.9044 | 2025-06-25 19:30 | 1000.8000 | STOCH_EXIT | 25 | 342.83 | 0.23% | 997.8000 | 1012.5562 | 15.0562 | M30 bullish FVG [997.5000, 997.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 102 | 2025-06-26 23:00 | 1004.0000 | 978.3262 | 2025-06-27 11:00 | 1013.6000 | STOCH_EXIT | 10 | 1271.50 | 0.86% | 1011.0000 | 1019.1022 | 15.1022 | M30 bullish FVG [1005.0000, 1011.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 103 | 2025-06-30 07:00 | 1017.0000 | 989.9590 | 2025-06-30 10:30 | 1024.2000 | STOCH_EXIT | 7 | 908.37 | 0.61% | 1014.2000 | 1032.9065 | 15.9065 | M30 bullish FVG [1013.4000, 1014.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 104 | 2025-07-01 10:30 | 1024.1000 | 996.3312 | 2025-07-01 15:00 | 1028.1000 | STOCH_EXIT | 9 | 437.16 | 0.29% | 1029.6000 | 1040.4346 | 16.3346 | M30 bullish FVG [1029.1000, 1029.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 105 | 2025-07-04 07:30 | 1031.6000 | 1000.6649 | 2025-07-04 08:00 | 1049.7971 | TP | 1 | 2521.99 | 1.66% | 1072.0000 | 1049.7971 | 18.1971 | M30 bullish FVG [1062.0000, 1072.0000], stochastic crossed below 20 | FVG target 1049.7971 |
| 106 | 2025-07-04 20:00 | 1046.6000 | 1015.6649 | 2025-07-05 08:30 | 1056.1000 | STOCH_EXIT | 20 | 1241.95 | 0.81% | 1050.4000 | 1064.7971 | 18.1971 | M30 bullish FVG [1047.6000, 1050.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 107 | 2025-07-07 07:30 | 1047.3000 | 1012.4910 | 2025-07-07 16:00 | 1053.5000 | STOCH_EXIT | 17 | 762.14 | 0.49% | 1053.9000 | 1067.7759 | 20.4759 | M30 bullish FVG [1053.5000, 1053.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 108 | 2025-07-08 08:30 | 1047.3000 | 1010.5695 | 2025-07-08 17:00 | 1060.6000 | STOCH_EXIT | 17 | 1824.66 | 1.17% | 1053.0000 | 1068.9062 | 21.6062 | M30 bullish FVG [1046.2000, 1053.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 109 | 2025-07-08 22:00 | 1049.8000 | 1013.0695 | 2025-07-09 23:30 | 1039.7000 | STOCH_EXIT | 38 | -1671.71 | -1.06% | 1058.9000 | 1071.4062 | 21.6062 | M30 bullish FVG [1056.1000, 1058.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 110 | 2025-07-11 09:00 | 1059.1000 | 1021.3443 | 2025-07-11 10:30 | 1069.9000 | STOCH_EXIT | 3 | 1431.12 | 0.92% | 1066.5000 | 1081.3092 | 22.2092 | M30 bullish FVG [1064.8000, 1066.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 111 | 2025-07-11 14:30 | 1058.0000 | 1020.2443 | 2025-07-12 05:00 | 1020.2443 | SL | 24 | -5780.43 | -3.66% | 1066.4000 | 1080.2092 | 22.2092 | M30 bullish FVG [1062.0000, 1066.4000], stochastic crossed below 20 | FVG stop 1020.2443 |
| 112 | 2025-07-16 11:00 | 1088.0000 | 1041.5360 | 2025-07-16 12:00 | 1099.3000 | STOCH_EXIT | 2 | 1418.68 | 0.94% | 1097.0000 | 1115.3318 | 27.3318 | M30 bullish FVG [1096.0000, 1097.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 113 | 2025-07-17 18:00 | 1103.4000 | 1058.0448 | 2025-07-18 10:30 | 1114.4000 | STOCH_EXIT | 19 | 1374.86 | 0.90% | 1112.2000 | 1130.0795 | 26.6795 | M30 bullish FVG [1110.0000, 1112.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 114 | 2025-07-21 07:00 | 1142.9000 | 1097.6224 | 2025-07-22 15:30 | 1117.5000 | STOCH_EXIT | 52 | -3581.58 | -2.32% | 1148.0000 | 1169.5339 | 26.6339 | M30 bullish FVG [1147.2000, 1148.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 115 | 2025-07-23 18:00 | 1119.0000 | 1070.5567 | 2025-07-24 10:00 | 1133.1000 | STOCH_EXIT | 19 | 1751.48 | 1.16% | 1144.3000 | 1147.4960 | 28.4960 | M30 bullish FVG [1143.8000, 1144.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 116 | 2025-07-24 15:00 | 1125.2000 | 1076.3677 | 2025-07-24 20:00 | 1131.9000 | STOCH_EXIT | 10 | 757.72 | 0.49% | 1132.8000 | 1153.9249 | 28.7249 | M30 bullish FVG [1132.3000, 1132.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 117 | 2025-07-25 18:00 | 1119.0000 | 1071.6400 | 2025-07-26 02:00 | 1138.3000 | STOCH_EXIT | 12 | 2489.47 | 1.62% | 1138.2000 | 1146.8588 | 27.8588 | M30 bullish FVG [1132.9000, 1138.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 118 | 2025-07-28 18:30 | 1146.0000 | 1098.4893 | 2025-07-29 09:30 | 1156.5000 | STOCH_EXIT | 16 | 1271.43 | 0.82% | 1161.0000 | 1173.9475 | 27.9475 | M30 bullish FVG [1160.0000, 1161.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 119 | 2025-08-05 13:00 | 1162.8000 | 1107.0245 | 2025-08-05 20:30 | 1172.5000 | STOCH_EXIT | 15 | 1151.87 | 0.73% | 1178.0000 | 1195.6091 | 32.8091 | M30 bullish FVG [1176.0000, 1178.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 120 | 2025-08-11 07:00 | 1204.4000 | 1148.8332 | 2025-08-11 09:30 | 1217.5000 | STOCH_EXIT | 5 | 1557.47 | 0.99% | 1206.0000 | 1237.0863 | 32.6863 | M30 bullish FVG [1205.9000, 1206.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 121 | 2025-08-11 21:30 | 1224.0000 | 1168.4332 | 2025-08-12 10:30 | 1242.0000 | STOCH_EXIT | 13 | 2196.48 | 1.37% | 1230.9000 | 1256.6863 | 32.6863 | M30 bullish FVG [1229.8000, 1230.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 122 | 2025-08-14 10:30 | 1251.6000 | 1192.1748 | 2025-08-15 10:00 | 1257.3000 | STOCH_EXIT | 34 | 573.48 | 0.36% | 1267.1000 | 1286.5560 | 34.9560 | M30 bullish FVG [1264.8000, 1267.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 123 | 2025-08-15 21:30 | 1257.4000 | 1196.9616 | 2025-08-16 03:30 | 1255.0000 | STOCH_EXIT | 8 | -471.65 | -0.29% | 1264.0000 | 1292.9520 | 35.5520 | M30 bullish FVG [1261.4000, 1264.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 124 | 2025-08-20 14:00 | 1280.3000 | 1222.8794 | 2025-08-22 09:00 | 1263.6000 | STOCH_EXIT | 60 | -2282.44 | -1.40% | 1299.0000 | 1314.0768 | 33.7768 | M30 bullish FVG [1295.0000, 1299.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 125 | 2025-08-26 18:00 | 1258.0000 | 1196.1837 | 2025-08-27 11:30 | 1272.4000 | STOCH_EXIT | 22 | 1668.12 | 1.04% | 1277.9000 | 1294.3625 | 36.3625 | M30 bullish FVG [1275.0000, 1277.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 126 | 2025-08-28 21:30 | 1247.9000 | 1188.7923 | 2025-08-30 02:30 | 1253.9000 | STOCH_EXIT | 41 | 612.63 | 0.38% | 1282.3000 | 1282.6692 | 34.7692 | M30 bullish FVG [1280.5000, 1282.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 127 | 2025-09-03 17:00 | 1238.3000 | 1180.4232 | 2025-09-03 22:30 | 1246.7000 | STOCH_EXIT | 11 | 937.63 | 0.58% | 1245.8000 | 1272.3452 | 34.0452 | M30 bullish FVG [1245.0000, 1245.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 128 | 2025-09-04 15:30 | 1243.9000 | 1188.0202 | 2025-09-05 07:00 | 1248.1000 | STOCH_EXIT | 17 | 386.97 | 0.24% | 1249.8000 | 1276.7705 | 32.8705 | M30 bullish FVG [1249.6000, 1249.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 129 | 2025-09-05 15:00 | 1244.8000 | 1191.3573 | 2025-09-06 15:00 | 1246.6000 | STOCH_EXIT | 42 | 72.61 | 0.04% | 1251.1000 | 1276.2369 | 31.4369 | M30 bullish FVG [1246.5000, 1251.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 130 | 2025-09-09 13:00 | 1268.1000 | 1216.4960 | 2025-09-10 11:00 | 1275.7000 | STOCH_EXIT | 31 | 816.32 | 0.50% | 1270.0000 | 1298.4553 | 30.3553 | M30 bullish FVG [1269.4000, 1270.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 131 | 2025-09-10 22:00 | 1272.8000 | 1222.1863 | 2025-09-12 07:00 | 1265.6000 | STOCH_EXIT | 39 | -1092.53 | -0.67% | 1279.0000 | 1302.5728 | 29.7728 | M30 bullish FVG [1277.8000, 1279.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 132 | 2025-09-12 10:30 | 1243.3000 | 1192.1300 | 2025-09-13 05:00 | 1251.6000 | STOCH_EXIT | 32 | 923.88 | 0.57% | 1259.2000 | 1273.4000 | 30.1000 | M30 bullish FVG [1258.8000, 1259.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 133 | 2025-09-15 09:30 | 1245.0000 | 1193.9636 | 2025-09-16 10:00 | 1250.2000 | STOCH_EXIT | 36 | 521.72 | 0.32% | 1257.1000 | 1275.0214 | 30.0214 | M30 bullish FVG [1257.0000, 1257.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 134 | 2025-09-16 13:00 | 1234.8000 | 1183.1348 | 2025-09-17 07:30 | 1241.1000 | STOCH_EXIT | 24 | 673.25 | 0.41% | 1245.2000 | 1265.1913 | 30.3913 | M30 bullish FVG [1238.7000, 1245.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 135 | 2025-09-17 13:00 | 1232.2000 | 1181.1166 | 2025-09-18 12:00 | 1219.7000 | STOCH_EXIT | 32 | -1839.28 | -1.11% | 1236.8000 | 1262.2491 | 30.0491 | M30 bullish FVG [1236.7000, 1236.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 136 | 2025-09-19 20:00 | 1219.9000 | 1167.2859 | 2025-09-20 13:00 | 1224.1000 | STOCH_EXIT | 29 | 399.05 | 0.24% | 1226.8000 | 1250.8495 | 30.9495 | M30 bullish FVG [1226.0000, 1226.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 137 | 2025-09-25 14:00 | 1236.0000 | 1182.5719 | 2025-09-27 19:30 | 1239.0000 | STOCH_EXIT | 87 | 234.41 | 0.14% | 1234.8000 | 1267.4283 | 31.4283 | M30 bullish FVG [1231.6000, 1234.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 138 | 2025-09-29 17:00 | 1227.1000 | 1176.2641 | 2025-09-30 10:30 | 1225.0000 | STOCH_EXIT | 22 | -445.69 | -0.27% | 1241.1000 | 1257.0035 | 29.9035 | M30 bullish FVG [1234.6000, 1241.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 139 | 2025-10-02 09:30 | 1240.2000 | 1186.9367 | 2025-10-03 11:00 | 1247.8000 | STOCH_EXIT | 38 | 838.99 | 0.51% | 1255.0000 | 1271.5313 | 31.3313 | M30 bullish FVG [1252.7000, 1255.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 140 | 2025-10-03 17:00 | 1238.3000 | 1185.2105 | 2025-10-03 22:30 | 1246.7000 | STOCH_EXIT | 11 | 951.95 | 0.58% | 1245.6000 | 1269.5291 | 31.2291 | M30 bullish FVG [1243.0000, 1245.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 141 | 2025-10-06 19:00 | 1244.4000 | 1191.7269 | 2025-10-06 23:00 | 1257.7000 | STOCH_EXIT | 8 | 1602.51 | 0.97% | 1257.0000 | 1275.3842 | 30.9842 | M30 bullish FVG [1252.0000, 1257.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 142 | 2025-10-07 13:30 | 1247.1000 | 1194.4736 | 2025-10-07 18:30 | 1258.5000 | STOCH_EXIT | 10 | 1359.72 | 0.81% | 1255.7000 | 1278.0567 | 30.9567 | M30 bullish FVG [1253.9000, 1255.7000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 143 | 2025-10-08 11:00 | 1245.6000 | 1194.6562 | 2025-10-09 10:00 | 1245.6000 | STOCH_EXIT | 33 | -168.16 | -0.10% | 1253.4000 | 1275.5670 | 29.9670 | M30 bullish FVG [1251.4000, 1253.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 144 | 2025-10-09 11:30 | 1226.4000 | 1172.8050 | 2025-10-09 15:30 | 1251.3000 | STOCH_EXIT | 8 | 3241.58 | 1.93% | 1242.6000 | 1257.9265 | 31.5265 | M30 bullish FVG [1240.6000, 1242.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 145 | 2025-10-10 14:00 | 1253.2000 | 1197.1432 | 2025-10-11 14:30 | 1246.1000 | STOCH_EXIT | 45 | -1143.90 | -0.67% | 1261.8000 | 1286.1746 | 32.9746 | M30 bullish FVG [1261.4000, 1261.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 146 | 2025-10-14 12:00 | 1269.0000 | 1208.7350 | 2025-10-14 18:30 | 1208.7350 | SL | 13 | -8241.52 | -4.84% | 1285.5000 | 1304.4500 | 35.4500 | M30 bullish FVG [1280.0000, 1285.5000], stochastic crossed below 20 | FVG stop 1208.7350 |
| 147 | 2025-10-20 22:00 | 1241.4000 | 1173.2738 | 2025-10-22 07:30 | 1241.3000 | STOCH_EXIT | 41 | -175.72 | -0.11% | 1243.7000 | 1281.4742 | 40.0742 | M30 bullish FVG [1243.4000, 1243.7000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 148 | 2025-10-22 22:30 | 1232.2000 | 1166.2752 | 2025-10-23 22:00 | 1218.4000 | STOCH_EXIT | 34 | -1968.31 | -1.22% | 1242.8000 | 1270.9793 | 38.7793 | M30 bullish FVG [1241.9000, 1242.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 149 | 2025-11-03 07:00 | 1260.5000 | 1199.8666 | 2025-11-03 07:00 | 1275.5000 | STOCH_EXIT | 0 | 1743.96 | 1.09% | 1267.0000 | 1296.1667 | 35.6667 | M30 bullish FVG [1266.2000, 1267.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 150 | 2025-11-05 17:30 | 1276.6000 | 1214.5419 | 2025-11-06 06:30 | 1304.9000 | STOCH_EXIT | 13 | 3430.17 | 2.11% | 1297.1000 | 1313.1048 | 36.5048 | M30 bullish FVG [1284.9000, 1297.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 151 | 2025-11-06 14:00 | 1289.5000 | 1226.9082 | 2025-11-07 07:00 | 1294.1000 | STOCH_EXIT | 21 | 423.45 | 0.26% | 1297.4000 | 1326.3187 | 36.8187 | M30 bullish FVG [1292.9000, 1297.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 152 | 2025-11-10 18:30 | 1306.7000 | 1246.5004 | 2025-11-11 08:30 | 1318.8000 | STOCH_EXIT | 15 | 1369.98 | 0.83% | 1311.1000 | 1342.1116 | 35.4116 | M30 bullish FVG [1310.7000, 1311.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 153 | 2025-11-12 15:30 | 1310.6000 | 1253.2896 | 2025-11-13 08:30 | 1313.3000 | STOCH_EXIT | 21 | 176.28 | 0.11% | 1325.4000 | 1344.3120 | 33.7120 | M30 bullish FVG [1323.6000, 1325.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 154 | 2025-11-13 14:30 | 1296.5000 | 1237.5518 | 2025-11-14 07:30 | 1298.1000 | STOCH_EXIT | 21 | 39.05 | 0.02% | 1311.8000 | 1331.1754 | 34.6754 | M30 bullish FVG [1306.1000, 1311.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 155 | 2025-11-14 13:30 | 1273.9000 | 1215.3859 | 2025-11-15 02:00 | 1308.3200 | TP | 21 | 4339.89 | 2.60% | 1295.4000 | 1308.3200 | 34.4200 | M30 bullish FVG [1295.0000, 1295.4000], stochastic crossed below 20 | FVG target 1308.3200 |

## Движение капитала

- Старт: 100000.00
- Мин: 97777.76
- Макс: 172783.17
- Финал: 172156.23

Полная кривая — в `*_equity.csv`.
