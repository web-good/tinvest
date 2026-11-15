# Бэктест MRKC (Minutes30)

- Период: 2023-11-15 23:18 — 2025-11-15 23:18
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.1 |
| MaxWaitBars | 30 |
| StochPeriod | 10 |
| UseStochExit | 1 |
| StopATR | 1 |
| TakeProfitATR | 2 |
| HTFTrendEMA | 120 |
| RequireEntryAboveFVGLower | 1 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 70 |
| Выигрышных / проигрышных | 47 / 23 |
| Win rate | 67.14% |
| Gross profit / loss | 43484.59 / 25469.65 |
| Profit factor | 1.707 |
| Чистый PnL | 18014.94 (18.01%) |
| Стартовый / финальный капитал | 100000.00 / 118014.94 |
| Макс. просадка | 11292.99 (10.24%) |
| Средняя прибыль / убыток | 925.20 / 1107.38 |
| Expectancy | 257.36 |
| Лучшая / худшая сделка | 12867.07 / -3609.68 |
| Exposure | 4.18% |
| CAGR | 8.62% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-11-20 18:00 | 0.5972 | 0.5778 | 2023-11-22 09:30 | 0.5982 | STOCH_EXIT | 21 | 67.18 | 0.07% | 0.5980 | 0.6360 | 0.0194 | M30 bullish FVG [0.5948, 0.5980], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 2 | 2023-11-22 17:30 | 0.6012 | 0.5828 | 2023-11-24 16:00 | 0.5952 | STOCH_EXIT | 35 | -1095.30 | -1.10% | 0.6044 | 0.6380 | 0.0184 | M30 bullish FVG [0.5982, 0.6044], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 3 | 2023-12-08 15:30 | 0.5580 | 0.5392 | 2023-12-08 16:30 | 0.5730 | STOCH_EXIT | 2 | 2554.91 | 2.59% | 0.5632 | 0.5956 | 0.0188 | M30 bullish FVG [0.5564, 0.5632], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 4 | 2023-12-19 17:30 | 0.5776 | 0.5577 | 2023-12-20 16:00 | 0.5810 | STOCH_EXIT | 16 | 493.62 | 0.49% | 0.5816 | 0.6173 | 0.0199 | M30 bullish FVG [0.5750, 0.5816], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 5 | 2024-01-08 17:30 | 0.5924 | 0.5773 | 2024-01-09 09:30 | 0.6018 | STOCH_EXIT | 3 | 1514.10 | 1.49% | 0.5912 | 0.6227 | 0.0151 | M30 bullish FVG [0.5892, 0.5912], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 6 | 2024-01-10 13:30 | 0.6146 | 0.5978 | 2024-01-11 15:30 | 0.6062 | STOCH_EXIT | 23 | -1513.75 | -1.47% | 0.6140 | 0.6482 | 0.0168 | M30 bullish FVG [0.6100, 0.6140], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 7 | 2024-01-18 16:30 | 0.6152 | 0.5994 | 2024-01-19 12:30 | 0.6128 | STOCH_EXIT | 11 | -497.31 | -0.49% | 0.6142 | 0.6468 | 0.0158 | M30 bullish FVG [0.6116, 0.6142], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 8 | 2024-02-01 16:30 | 0.6224 | 0.6049 | 2024-02-02 15:00 | 0.6186 | STOCH_EXIT | 16 | -720.54 | -0.71% | 0.6230 | 0.6574 | 0.0175 | M30 bullish FVG [0.6198, 0.6230], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 9 | 2024-03-11 16:00 | 0.5816 | 0.5696 | 2024-03-12 11:00 | 0.5848 | STOCH_EXIT | 9 | 452.71 | 0.45% | 0.5826 | 0.6057 | 0.0120 | M30 bullish FVG [0.5794, 0.5826], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 10 | 2024-03-13 14:30 | 0.5826 | 0.5707 | 2024-03-14 10:30 | 0.5707 | SL | 11 | -2159.29 | -2.14% | 0.5834 | 0.6064 | 0.0119 | M30 bullish FVG [0.5784, 0.5834], stochastic crossed below 20 | FVG stop 0.5707 |
| 11 | 2024-03-19 10:30 | 0.5682 | 0.5545 | 2024-03-19 13:00 | 0.5710 | STOCH_EXIT | 5 | 388.09 | 0.39% | 0.5700 | 0.5955 | 0.0137 | M30 bullish FVG [0.5664, 0.5700], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 12 | 2024-03-27 16:30 | 0.5750 | 0.5630 | 2024-03-28 09:30 | 0.5790 | STOCH_EXIT | 5 | 588.76 | 0.60% | 0.5726 | 0.5990 | 0.0120 | M30 bullish FVG [0.5704, 0.5726], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 13 | 2024-03-29 12:30 | 0.5806 | 0.5691 | 2024-03-29 18:00 | 0.5812 | STOCH_EXIT | 11 | 3.29 | 0.00% | 0.5810 | 0.6035 | 0.0115 | M30 bullish FVG [0.5790, 0.5810], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 14 | 2024-04-11 12:30 | 0.5986 | 0.5882 | 2024-04-11 16:00 | 0.6010 | STOCH_EXIT | 7 | 300.63 | 0.30% | 0.5998 | 0.6193 | 0.0104 | M30 bullish FVG [0.5982, 0.5998], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 15 | 2024-04-17 15:30 | 0.6150 | 0.6046 | 2024-04-17 18:00 | 0.6156 | STOCH_EXIT | 5 | -2.49 | -0.00% | 0.6052 | 0.6358 | 0.0104 | M30 bullish FVG [0.6030, 0.6052], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 16 | 2024-04-22 17:30 | 0.6148 | 0.6038 | 2024-04-23 09:30 | 0.6198 | STOCH_EXIT | 3 | 714.38 | 0.71% | 0.6146 | 0.6367 | 0.0110 | M30 bullish FVG [0.6124, 0.6146], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 17 | 2024-04-26 11:00 | 0.6048 | 0.5935 | 2024-04-26 18:00 | 0.6036 | STOCH_EXIT | 14 | -301.30 | -0.30% | 0.6074 | 0.6273 | 0.0113 | M30 bullish FVG [0.6028, 0.6074], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 18 | 2024-05-03 17:30 | 0.6350 | 0.6215 | 2024-05-04 10:00 | 0.6390 | STOCH_EXIT | 3 | 531.35 | 0.53% | 0.6360 | 0.6619 | 0.0135 | M30 bullish FVG [0.6300, 0.6360], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 19 | 2024-05-08 14:30 | 0.6438 | 0.6290 | 2024-05-08 17:00 | 0.6290 | SL | 5 | -2415.92 | -2.39% | 0.6484 | 0.6733 | 0.0148 | M30 bullish FVG [0.6406, 0.6484], stochastic crossed below 20 | FVG stop 0.6290 |
| 20 | 2024-11-12 23:00 | 0.4562 | 0.4427 | 2024-11-13 10:00 | 0.4634 | STOCH_EXIT | 8 | 1455.88 | 1.48% | 0.4588 | 0.4831 | 0.0135 | M30 bullish FVG [0.4558, 0.4588], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 21 | 2024-11-13 19:30 | 0.4712 | 0.4575 | 2024-11-14 11:00 | 0.4718 | STOCH_EXIT | 17 | 27.24 | 0.03% | 0.4724 | 0.4985 | 0.0137 | M30 bullish FVG [0.4710, 0.4724], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 22 | 2024-12-03 09:30 | 0.4494 | 0.4305 | 2024-12-03 20:00 | 0.4398 | STOCH_EXIT | 21 | -2239.95 | -2.23% | 0.4432 | 0.4871 | 0.0189 | M30 bullish FVG [0.4408, 0.4432], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 23 | 2024-12-23 15:30 | 0.4656 | 0.4472 | 2024-12-23 16:30 | 0.4728 | STOCH_EXIT | 2 | 1413.47 | 1.44% | 0.4568 | 0.5023 | 0.0184 | M30 bullish FVG [0.4546, 0.4568], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 24 | 2024-12-24 10:30 | 0.4726 | 0.4527 | 2024-12-24 15:30 | 0.4738 | STOCH_EXIT | 10 | 152.63 | 0.15% | 0.4718 | 0.5124 | 0.0199 | M30 bullish FVG [0.4660, 0.4718], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 25 | 2025-01-08 22:30 | 0.4838 | 0.4668 | 2025-01-09 09:30 | 0.4898 | STOCH_EXIT | 8 | 1130.21 | 1.14% | 0.4832 | 0.5179 | 0.0170 | M30 bullish FVG [0.4792, 0.4832], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 26 | 2025-01-13 22:30 | 0.4896 | 0.4719 | 2025-01-14 07:00 | 0.4918 | STOCH_EXIT | 3 | 350.41 | 0.35% | 0.4884 | 0.5250 | 0.0177 | M30 bullish FVG [0.4864, 0.4884], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 27 | 2025-01-17 23:30 | 0.4952 | 0.4791 | 2025-01-18 09:30 | 0.4996 | STOCH_EXIT | 6 | 796.13 | 0.79% | 0.5002 | 0.5274 | 0.0161 | M30 bullish FVG [0.4944, 0.5002], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 28 | 2025-01-31 16:30 | 0.4980 | 0.4831 | 2025-02-01 06:00 | 0.5010 | STOCH_EXIT | 19 | 510.10 | 0.50% | 0.4990 | 0.5278 | 0.0149 | M30 bullish FVG [0.4962, 0.4990], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 29 | 2025-02-03 20:30 | 0.5188 | 0.5044 | 2025-02-04 10:00 | 0.5288 | STOCH_EXIT | 13 | 1866.81 | 1.83% | 0.5180 | 0.5475 | 0.0144 | M30 bullish FVG [0.5150, 0.5180], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 30 | 2025-02-12 15:00 | 0.5322 | 0.5180 | 2025-02-12 20:00 | 0.5350 | STOCH_EXIT | 10 | 444.21 | 0.43% | 0.5320 | 0.5605 | 0.0142 | M30 bullish FVG [0.5298, 0.5320], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 31 | 2025-02-13 13:30 | 0.5424 | 0.5287 | 2025-02-13 18:30 | 0.5482 | STOCH_EXIT | 10 | 1014.16 | 0.97% | 0.5512 | 0.5699 | 0.0137 | M30 bullish FVG [0.5420, 0.5512], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 32 | 2025-02-17 15:30 | 0.5610 | 0.5458 | 2025-02-17 18:30 | 0.5628 | STOCH_EXIT | 6 | 232.76 | 0.22% | 0.5604 | 0.5914 | 0.0152 | M30 bullish FVG [0.5524, 0.5604], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 33 | 2025-02-25 20:30 | 0.6688 | 0.6506 | 2025-02-26 14:00 | 0.6506 | SL | 21 | -2981.29 | -2.82% | 0.6500 | 0.7052 | 0.0182 | M30 bullish FVG [0.6330, 0.6500], stochastic crossed below 20 | FVG stop 0.6506 |
| 34 | 2025-03-05 20:00 | 0.6564 | 0.6323 | 2025-03-06 09:30 | 0.6564 | STOCH_EXIT | 13 | -102.40 | -0.10% | 0.6516 | 0.7045 | 0.0241 | M30 bullish FVG [0.6480, 0.6516], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 35 | 2025-03-07 15:30 | 0.6498 | 0.6254 | 2025-03-07 21:00 | 0.6456 | STOCH_EXIT | 11 | -765.94 | -0.75% | 0.6520 | 0.6985 | 0.0244 | M30 bullish FVG [0.6450, 0.6520], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 36 | 2025-03-11 23:00 | 0.6450 | 0.6217 | 2025-03-12 16:00 | 0.6496 | STOCH_EXIT | 20 | 624.53 | 0.61% | 0.6450 | 0.6916 | 0.0233 | M30 bullish FVG [0.6410, 0.6450], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 37 | 2025-03-17 16:00 | 0.6614 | 0.6384 | 2025-03-17 19:00 | 0.6648 | STOCH_EXIT | 6 | 424.22 | 0.41% | 0.6632 | 0.7073 | 0.0230 | M30 bullish FVG [0.6596, 0.6632], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 38 | 2025-03-24 16:00 | 0.6302 | 0.6089 | 2025-03-24 18:00 | 0.6358 | STOCH_EXIT | 4 | 809.62 | 0.79% | 0.6336 | 0.6729 | 0.0213 | M30 bullish FVG [0.6268, 0.6336], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 39 | 2025-04-02 19:30 | 0.5560 | 0.5341 | 2025-04-03 23:30 | 0.5498 | STOCH_EXIT | 42 | -1262.79 | -1.21% | 0.5602 | 0.5999 | 0.0219 | M30 bullish FVG [0.5558, 0.5602], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 40 | 2025-04-11 23:00 | 0.5574 | 0.5326 | 2025-04-12 11:30 | 0.5662 | STOCH_EXIT | 16 | 1515.83 | 1.48% | 0.5540 | 0.6070 | 0.0248 | M30 bullish FVG [0.5488, 0.5540], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 41 | 2025-04-14 09:30 | 0.5642 | 0.5405 | 2025-04-15 09:30 | 0.5530 | STOCH_EXIT | 34 | -2163.58 | -2.08% | 0.5664 | 0.6116 | 0.0237 | M30 bullish FVG [0.5638, 0.5664], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 42 | 2025-04-17 20:30 | 0.5728 | 0.5514 | 2025-04-17 20:30 | 0.5774 | STOCH_EXIT | 0 | 716.43 | 0.70% | 0.5678 | 0.6155 | 0.0214 | M30 bullish FVG [0.5644, 0.5678], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 43 | 2025-04-21 18:30 | 0.5780 | 0.5577 | 2025-04-21 22:00 | 0.5840 | STOCH_EXIT | 7 | 959.16 | 0.94% | 0.5788 | 0.6185 | 0.0203 | M30 bullish FVG [0.5766, 0.5788], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 44 | 2025-05-20 09:30 | 0.5964 | 0.5781 | 2025-05-20 12:30 | 0.5984 | STOCH_EXIT | 6 | 244.05 | 0.24% | 0.5966 | 0.6329 | 0.0183 | M30 bullish FVG [0.5946, 0.5966], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 45 | 2025-05-21 10:30 | 0.6088 | 0.5903 | 2025-05-21 21:30 | 0.6054 | STOCH_EXIT | 22 | -681.21 | -0.66% | 0.6116 | 0.6457 | 0.0185 | M30 bullish FVG [0.6082, 0.6116], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 46 | 2025-05-26 20:00 | 0.6018 | 0.5833 | 2025-05-26 21:30 | 0.6024 | STOCH_EXIT | 3 | -0.36 | -0.00% | 0.6012 | 0.6389 | 0.0185 | M30 bullish FVG [0.5988, 0.6012], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 47 | 2025-05-27 22:30 | 0.6272 | 0.6082 | 2025-05-28 16:30 | 0.6328 | STOCH_EXIT | 23 | 815.08 | 0.79% | 0.6240 | 0.6652 | 0.0190 | M30 bullish FVG [0.6206, 0.6240], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 48 | 2025-06-09 13:30 | 0.6610 | 0.6437 | 2025-06-09 20:30 | 0.6614 | STOCH_EXIT | 14 | -41.01 | -0.04% | 0.6650 | 0.6956 | 0.0173 | M30 bullish FVG [0.6580, 0.6650], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 49 | 2025-06-13 16:30 | 0.6832 | 0.6652 | 2025-06-14 08:30 | 0.6928 | STOCH_EXIT | 25 | 1354.62 | 1.30% | 0.6832 | 0.7191 | 0.0180 | M30 bullish FVG [0.6796, 0.6832], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 50 | 2025-06-20 08:30 | 0.6752 | 0.6563 | 2025-06-20 17:00 | 0.6790 | STOCH_EXIT | 17 | 487.17 | 0.46% | 0.6770 | 0.7130 | 0.0189 | M30 bullish FVG [0.6750, 0.6770], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 51 | 2025-06-25 23:30 | 0.5898 | 0.5690 | 2025-06-26 07:30 | 0.5948 | STOCH_EXIT | 3 | 788.98 | 0.75% | 0.5790 | 0.6313 | 0.0208 | M30 bullish FVG [0.5752, 0.5790], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 52 | 2025-07-03 10:00 | 0.6204 | 0.5962 | 2025-07-03 14:30 | 0.6194 | STOCH_EXIT | 9 | -278.62 | -0.26% | 0.6174 | 0.6689 | 0.0242 | M30 bullish FVG [0.6120, 0.6174], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 53 | 2025-07-11 09:00 | 0.6218 | 0.6004 | 2025-07-11 19:00 | 0.6184 | STOCH_EXIT | 20 | -687.44 | -0.65% | 0.6220 | 0.6647 | 0.0214 | M30 bullish FVG [0.6136, 0.6220], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 54 | 2025-07-17 16:30 | 0.6464 | 0.6269 | 2025-07-17 21:00 | 0.6476 | STOCH_EXIT | 9 | 90.14 | 0.09% | 0.6422 | 0.6855 | 0.0195 | M30 bullish FVG [0.6300, 0.6422], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 55 | 2025-07-21 17:00 | 0.6736 | 0.6521 | 2025-07-22 07:00 | 0.6732 | STOCH_EXIT | 14 | -168.52 | -0.16% | 0.6756 | 0.7165 | 0.0215 | M30 bullish FVG [0.6732, 0.6756], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 56 | 2025-07-28 18:30 | 0.7074 | 0.6897 | 2025-07-28 21:30 | 0.7104 | STOCH_EXIT | 6 | 341.37 | 0.32% | 0.7088 | 0.7429 | 0.0177 | M30 bullish FVG [0.7010, 0.7088], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 57 | 2025-07-30 20:00 | 0.6888 | 0.6666 | 2025-07-31 09:00 | 0.6902 | STOCH_EXIT | 13 | 108.71 | 0.10% | 0.6940 | 0.7333 | 0.0222 | M30 bullish FVG [0.6856, 0.6940], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 58 | 2025-07-31 16:00 | 0.6890 | 0.6665 | 2025-07-31 17:30 | 0.6920 | STOCH_EXIT | 3 | 355.66 | 0.34% | 0.6896 | 0.7340 | 0.0225 | M30 bullish FVG [0.6866, 0.6896], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 59 | 2025-08-05 11:00 | 0.7398 | 0.7155 | 2025-08-05 16:30 | 0.7456 | STOCH_EXIT | 11 | 723.19 | 0.68% | 0.7388 | 0.7884 | 0.0243 | M30 bullish FVG [0.7288, 0.7388], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 60 | 2025-08-06 12:30 | 0.7602 | 0.7354 | 2025-08-06 20:30 | 0.7572 | STOCH_EXIT | 16 | -529.98 | -0.49% | 0.7498 | 0.8099 | 0.0248 | M30 bullish FVG [0.7464, 0.7498], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 61 | 2025-09-01 18:30 | 0.7356 | 0.7143 | 2025-09-01 18:30 | 0.7384 | STOCH_EXIT | 0 | 299.13 | 0.28% | 0.7330 | 0.7782 | 0.0213 | M30 bullish FVG [0.7298, 0.7330], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 62 | 2025-09-23 11:30 | 0.7180 | 0.6981 | 2025-09-23 12:00 | 0.7228 | STOCH_EXIT | 1 | 607.86 | 0.57% | 0.7058 | 0.7578 | 0.0199 | M30 bullish FVG [0.7020, 0.7058], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 63 | 2025-09-25 16:00 | 0.7694 | 0.7451 | 2025-09-25 20:00 | 0.7724 | STOCH_EXIT | 8 | 309.84 | 0.29% | 0.7712 | 0.8180 | 0.0243 | M30 bullish FVG [0.7670, 0.7712], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 64 | 2025-09-29 13:30 | 0.7860 | 0.7616 | 2025-09-29 23:30 | 0.7914 | STOCH_EXIT | 20 | 631.75 | 0.59% | 0.7900 | 0.8348 | 0.0244 | M30 bullish FVG [0.7838, 0.7900], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 65 | 2025-10-01 15:30 | 0.8102 | 0.7837 | 2025-10-02 12:30 | 0.8016 | STOCH_EXIT | 29 | -1250.98 | -1.16% | 0.8022 | 0.8633 | 0.0265 | M30 bullish FVG [0.7976, 0.8022], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 66 | 2025-10-07 22:00 | 0.8292 | 0.8017 | 2025-10-08 07:30 | 0.8320 | STOCH_EXIT | 6 | 254.05 | 0.24% | 0.8354 | 0.8842 | 0.0275 | M30 bullish FVG [0.8270, 0.8354], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 67 | 2025-10-09 23:30 | 0.8552 | 0.8272 | 2025-10-10 11:00 | 0.8272 | SL | 10 | -3609.68 | -3.37% | 0.8612 | 0.9113 | 0.0280 | M30 bullish FVG [0.8424, 0.8612], stochastic crossed below 20 | FVG stop 0.8272 |
| 68 | 2025-10-14 09:30 | 0.7118 | 0.6674 | 2025-10-15 13:00 | 0.8007 | TP | 41 | 12867.07 | 12.38% | 0.7052 | 0.8007 | 0.0444 | M30 bullish FVG [0.7008, 0.7052], stochastic crossed below 20 | FVG target 0.8007 |
| 69 | 2025-10-16 10:30 | 0.7774 | 0.7263 | 2025-10-16 16:00 | 0.7830 | STOCH_EXIT | 11 | 722.97 | 0.62% | 0.7780 | 0.8797 | 0.0511 | M30 bullish FVG [0.7642, 0.7780], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 70 | 2025-10-22 14:00 | 0.7686 | 0.7222 | 2025-10-22 18:30 | 0.7722 | STOCH_EXIT | 9 | 430.10 | 0.37% | 0.7692 | 0.8614 | 0.0464 | M30 bullish FVG [0.7632, 0.7692], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 96768.13
- Макс: 118073.62
- Финал: 118014.94

Полная кривая — в `*_equity.csv`.
