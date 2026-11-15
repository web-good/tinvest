# Бэктест MRKC (Minutes30)

- Период: 2023-11-15 23:17 — 2025-11-15 23:17
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.25 |
| MaxWaitBars | 30 |
| StochPeriod | 14 |
| UseStochExit | 1 |
| StopATR | 1 |
| TakeProfitATR | 2 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 45 |
| Выигрышных / проигрышных | 30 / 15 |
| Win rate | 66.67% |
| Gross profit / loss | 36976.56 / 29157.36 |
| Profit factor | 1.268 |
| Чистый PnL | 7819.20 (7.82%) |
| Стартовый / финальный капитал | 100000.00 / 107819.20 |
| Макс. просадка | 10601.66 (10.52%) |
| Средняя прибыль / убыток | 1232.55 / 1943.82 |
| Expectancy | 173.76 |
| Лучшая / худшая сделка | 5032.48 / -4321.82 |
| Exposure | 3.13% |
| CAGR | 3.83% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-11-22 18:00 | 0.6004 | 0.5820 | 2023-11-24 16:00 | 0.5952 | STOCH_EXIT | 34 | -962.43 | -0.97% | 0.6044 | 0.6372 | 0.0184 | M30 bullish FVG [0.5982, 0.6044], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 2 | 2023-11-29 10:00 | 0.5964 | 0.5782 | 2023-11-29 15:00 | 0.6018 | STOCH_EXIT | 10 | 792.15 | 0.80% | 0.5920 | 0.6328 | 0.0182 | M30 bullish FVG [0.5856, 0.5920], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 3 | 2023-12-11 13:00 | 0.5580 | 0.5392 | 2023-12-12 11:00 | 0.5392 | SL | 15 | -3449.05 | -3.47% | 0.5632 | 0.5957 | 0.0188 | M30 bullish FVG [0.5564, 0.5632], stochastic crossed below 20 | FVG stop 0.5392 |
| 4 | 2024-01-10 15:00 | 0.6138 | 0.5970 | 2024-01-12 11:00 | 0.6106 | STOCH_EXIT | 30 | -594.70 | -0.62% | 0.6140 | 0.6474 | 0.0168 | M30 bullish FVG [0.6100, 0.6140], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 5 | 2024-01-23 15:00 | 0.6106 | 0.5949 | 2024-01-24 10:30 | 0.6230 | STOCH_EXIT | 10 | 1838.18 | 1.93% | 0.6214 | 0.6421 | 0.0157 | M30 bullish FVG [0.6174, 0.6214], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 6 | 2024-02-22 10:00 | 0.5568 | 0.5423 | 2024-02-22 10:30 | 0.5626 | STOCH_EXIT | 1 | 917.05 | 0.94% | 0.5564 | 0.5857 | 0.0145 | M30 bullish FVG [0.5524, 0.5564], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 7 | 2024-03-11 17:30 | 0.5804 | 0.5684 | 2024-03-12 11:00 | 0.5848 | STOCH_EXIT | 6 | 645.14 | 0.66% | 0.5826 | 0.6045 | 0.0120 | M30 bullish FVG [0.5794, 0.5826], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 8 | 2024-03-19 10:30 | 0.5682 | 0.5545 | 2024-03-21 11:00 | 0.5738 | STOCH_EXIT | 39 | 875.05 | 0.88% | 0.5700 | 0.5955 | 0.0137 | M30 bullish FVG [0.5664, 0.5700], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 9 | 2024-04-22 17:30 | 0.6148 | 0.6038 | 2024-04-23 09:30 | 0.6198 | STOCH_EXIT | 3 | 710.00 | 0.71% | 0.6130 | 0.6367 | 0.0110 | M30 bullish FVG [0.6098, 0.6130], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 10 | 2024-04-26 11:00 | 0.6048 | 0.5935 | 2024-04-26 18:00 | 0.6036 | STOCH_EXIT | 14 | -299.50 | -0.30% | 0.6074 | 0.6273 | 0.0113 | M30 bullish FVG [0.6028, 0.6074], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 11 | 2024-04-30 12:00 | 0.6280 | 0.6149 | 2024-05-01 10:00 | 0.6149 | SL | 14 | -2177.07 | -2.18% | 0.6304 | 0.6541 | 0.0131 | M30 bullish FVG [0.6262, 0.6304], stochastic crossed below 20 | FVG stop 0.6149 |
| 12 | 2024-05-08 17:00 | 0.6430 | 0.6282 | 2024-05-08 17:00 | 0.6282 | SL | 0 | -2338.86 | -2.39% | 0.6484 | 0.6725 | 0.0148 | M30 bullish FVG [0.6406, 0.6484], stochastic crossed below 20 | FVG stop 0.6282 |
| 13 | 2024-05-30 18:00 | 0.5666 | 0.5473 | 2024-05-31 10:30 | 0.5473 | SL | 4 | -3351.34 | -3.50% | 0.5700 | 0.6051 | 0.0193 | M30 bullish FVG [0.5642, 0.5700], stochastic crossed below 20 | FVG stop 0.5473 |
| 14 | 2024-07-05 23:00 | 0.5016 | 0.4819 | 2024-07-06 19:00 | 0.5066 | STOCH_EXIT | 20 | 827.25 | 0.90% | 0.4992 | 0.5410 | 0.0197 | M30 bullish FVG [0.4942, 0.4992], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 15 | 2024-08-08 14:30 | 0.4440 | 0.4289 | 2024-08-08 19:30 | 0.4289 | SL | 10 | -3262.14 | -3.50% | 0.4486 | 0.4742 | 0.0151 | M30 bullish FVG [0.4444, 0.4486], stochastic crossed below 20 | FVG stop 0.4289 |
| 16 | 2024-08-13 14:30 | 0.4312 | 0.4161 | 2024-08-13 16:30 | 0.4376 | STOCH_EXIT | 4 | 1246.81 | 1.38% | 0.4294 | 0.4613 | 0.0151 | M30 bullish FVG [0.4240, 0.4294], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 17 | 2024-09-20 19:00 | 0.3876 | 0.3756 | 2024-09-21 11:30 | 0.3920 | STOCH_EXIT | 17 | 942.40 | 1.03% | 0.3860 | 0.4115 | 0.0120 | M30 bullish FVG [0.3822, 0.3860], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 18 | 2024-10-07 10:00 | 0.3850 | 0.3746 | 2024-10-08 10:30 | 0.3816 | STOCH_EXIT | 34 | -904.21 | -0.98% | 0.3836 | 0.4059 | 0.0104 | M30 bullish FVG [0.3802, 0.3836], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 19 | 2024-10-28 17:00 | 0.3874 | 0.3797 | 2024-10-28 22:30 | 0.3944 | STOCH_EXIT | 11 | 1553.14 | 1.71% | 0.3924 | 0.4028 | 0.0077 | M30 bullish FVG [0.3896, 0.3924], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 20 | 2024-11-04 13:30 | 0.4146 | 0.4031 | 2024-11-04 22:00 | 0.4146 | STOCH_EXIT | 16 | -92.87 | -0.10% | 0.4200 | 0.4376 | 0.0115 | M30 bullish FVG [0.4140, 0.4200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 21 | 2024-11-11 20:00 | 0.4550 | 0.4430 | 2024-11-11 21:00 | 0.4602 | STOCH_EXIT | 2 | 967.45 | 1.04% | 0.4516 | 0.4790 | 0.0120 | M30 bullish FVG [0.4446, 0.4516], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 22 | 2024-11-13 19:30 | 0.4712 | 0.4575 | 2024-11-14 11:00 | 0.4718 | STOCH_EXIT | 17 | 25.57 | 0.03% | 0.4630 | 0.4985 | 0.0137 | M30 bullish FVG [0.4542, 0.4630], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 23 | 2024-11-19 12:30 | 0.4790 | 0.4636 | 2024-11-19 22:00 | 0.4816 | STOCH_EXIT | 19 | 413.34 | 0.44% | 0.4922 | 0.5098 | 0.0154 | M30 bullish FVG [0.4878, 0.4922], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 24 | 2024-12-02 13:30 | 0.4360 | 0.4172 | 2024-12-02 19:00 | 0.4502 | STOCH_EXIT | 11 | 2971.49 | 3.15% | 0.4480 | 0.4736 | 0.0188 | M30 bullish FVG [0.4424, 0.4480], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 25 | 2024-12-20 23:00 | 0.4508 | 0.4332 | 2024-12-21 09:30 | 0.4594 | STOCH_EXIT | 6 | 1751.15 | 1.81% | 0.4402 | 0.4860 | 0.0176 | M30 bullish FVG [0.4348, 0.4402], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 26 | 2024-12-24 10:30 | 0.4726 | 0.4527 | 2024-12-24 17:00 | 0.4734 | STOCH_EXIT | 13 | 68.34 | 0.07% | 0.4792 | 0.5124 | 0.0199 | M30 bullish FVG [0.4746, 0.4792], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 27 | 2025-01-17 23:30 | 0.4952 | 0.4791 | 2025-01-18 10:30 | 0.4984 | STOCH_EXIT | 8 | 540.64 | 0.55% | 0.5002 | 0.5274 | 0.0161 | M30 bullish FVG [0.4944, 0.5002], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 28 | 2025-02-03 20:30 | 0.5188 | 0.5044 | 2025-02-04 10:00 | 0.5288 | STOCH_EXIT | 13 | 1809.95 | 1.83% | 0.5096 | 0.5475 | 0.0144 | M30 bullish FVG [0.5020, 0.5096], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 29 | 2025-02-13 14:30 | 0.5402 | 0.5265 | 2025-02-13 19:00 | 0.5677 | TP | 9 | 5032.48 | 4.98% | 0.5512 | 0.5677 | 0.0137 | M30 bullish FVG [0.5420, 0.5512], stochastic crossed below 20 | FVG target 0.5677 |
| 30 | 2025-02-17 17:30 | 0.5594 | 0.5442 | 2025-02-17 18:30 | 0.5628 | STOCH_EXIT | 2 | 539.39 | 0.51% | 0.5604 | 0.5898 | 0.0152 | M30 bullish FVG [0.5524, 0.5604], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 31 | 2025-02-26 22:00 | 0.6504 | 0.6298 | 2025-02-27 09:00 | 0.6604 | STOCH_EXIT | 8 | 1532.51 | 1.44% | 0.6654 | 0.6917 | 0.0206 | M30 bullish FVG [0.6600, 0.6654], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 32 | 2025-03-07 16:30 | 0.6456 | 0.6212 | 2025-03-08 13:00 | 0.6398 | STOCH_EXIT | 34 | -1082.37 | -1.00% | 0.6520 | 0.6943 | 0.0244 | M30 bullish FVG [0.6450, 0.6520], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 33 | 2025-05-05 17:30 | 0.5252 | 0.5066 | 2025-05-05 22:30 | 0.5416 | STOCH_EXIT | 10 | 3236.79 | 3.02% | 0.5300 | 0.5623 | 0.0186 | M30 bullish FVG [0.5252, 0.5300], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 34 | 2025-05-27 07:30 | 0.6000 | 0.5810 | 2025-05-27 09:00 | 0.6036 | STOCH_EXIT | 3 | 551.67 | 0.50% | 0.5972 | 0.6380 | 0.0190 | M30 bullish FVG [0.5918, 0.5972], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 35 | 2025-05-28 07:00 | 0.6278 | 0.6074 | 2025-05-28 16:30 | 0.6328 | STOCH_EXIT | 19 | 773.44 | 0.70% | 0.6100 | 0.6686 | 0.0204 | M30 bullish FVG [0.6048, 0.6100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 36 | 2025-06-09 14:30 | 0.6550 | 0.6377 | 2025-06-09 21:00 | 0.6616 | STOCH_EXIT | 13 | 1010.09 | 0.91% | 0.6650 | 0.6896 | 0.0173 | M30 bullish FVG [0.6580, 0.6650], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 37 | 2025-07-11 11:00 | 0.6224 | 0.6010 | 2025-07-11 19:00 | 0.6184 | STOCH_EXIT | 16 | -836.29 | -0.74% | 0.6220 | 0.6653 | 0.0214 | M30 bullish FVG [0.6136, 0.6220], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 38 | 2025-07-17 17:30 | 0.6454 | 0.6259 | 2025-07-17 23:30 | 0.6490 | STOCH_EXIT | 12 | 510.83 | 0.46% | 0.6576 | 0.6845 | 0.0195 | M30 bullish FVG [0.6484, 0.6576], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 39 | 2025-07-29 08:30 | 0.7042 | 0.6842 | 2025-07-29 17:00 | 0.6842 | SL | 17 | -3315.91 | -2.94% | 0.7088 | 0.7443 | 0.0200 | M30 bullish FVG [0.7010, 0.7088], stochastic crossed below 20 | FVG stop 0.6842 |
| 40 | 2025-07-30 20:30 | 0.6858 | 0.6636 | 2025-07-31 09:00 | 0.6902 | STOCH_EXIT | 12 | 590.21 | 0.54% | 0.6940 | 0.7303 | 0.0222 | M30 bullish FVG [0.6856, 0.6940], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 41 | 2025-08-05 13:00 | 0.7300 | 0.7057 | 2025-08-05 17:00 | 0.7490 | STOCH_EXIT | 8 | 2739.07 | 2.50% | 0.7388 | 0.7786 | 0.0243 | M30 bullish FVG [0.7288, 0.7388], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 42 | 2025-09-29 14:30 | 0.7862 | 0.7618 | 2025-09-29 23:30 | 0.7914 | STOCH_EXIT | 18 | 630.80 | 0.56% | 0.7900 | 0.8350 | 0.0244 | M30 bullish FVG [0.7838, 0.7900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 43 | 2025-10-01 15:30 | 0.8102 | 0.7837 | 2025-10-03 09:30 | 0.7954 | STOCH_EXIT | 58 | -2168.79 | -1.92% | 0.7972 | 0.8633 | 0.0265 | M30 bullish FVG [0.7896, 0.7972], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 44 | 2025-10-10 08:30 | 0.8566 | 0.8239 | 2025-10-10 11:00 | 0.8239 | SL | 5 | -4321.82 | -3.91% | 0.8882 | 0.9219 | 0.0327 | M30 bullish FVG [0.8776, 0.8882], stochastic crossed below 20 | FVG stop 0.8239 |
| 45 | 2025-10-16 10:30 | 0.7774 | 0.7263 | 2025-10-16 20:00 | 0.7850 | STOCH_EXIT | 19 | 934.18 | 0.88% | 0.7780 | 0.8797 | 0.0511 | M30 bullish FVG [0.7642, 0.7780], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 90169.71
- Макс: 113764.12
- Финал: 107819.20

Полная кривая — в `*_equity.csv`.
