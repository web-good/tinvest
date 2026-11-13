# Бэктест UGLD (Minutes30)

- Период: 2023-11-12 23:00 — 2025-11-12 23:00
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
| Всего сделок | 32 |
| Выигрышных / проигрышных | 17 / 15 |
| Win rate | 53.12% |
| Gross profit / loss | 16919.34 / 42225.03 |
| Profit factor | 0.401 |
| Чистый PnL | -25305.69 (-25.31%) |
| Стартовый / финальный капитал | 100000.00 / 74694.31 |
| Макс. просадка | 30359.52 (29.48%) |
| Средняя прибыль / убыток | 995.26 / 2815.00 |
| Expectancy | -790.80 |
| Лучшая / худшая сделка | 2999.59 / -4146.65 |
| Exposure | 2.29% |
| CAGR | -13.56% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2024-02-14 15:00 | 0.7328 | 0.7086 | 2024-02-15 11:00 | 0.7556 | STOCH_EXIT | 11 | 2999.59 | 3.01% | 0.7591 | 0.7812 | 0.0242 | M30 bullish FVG [0.7487, 0.7591], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 2 | 2024-02-21 14:30 | 0.7598 | 0.7311 | 2024-02-21 14:30 | 0.7311 | SL | 0 | -3968.85 | -3.87% | 0.8070 | 0.8171 | 0.0287 | M30 bullish FVG [0.8000, 0.8070], stochastic crossed below 20 | FVG stop 0.7311 |
| 3 | 2024-02-22 16:30 | 0.7851 | 0.7530 | 2024-02-26 10:00 | 0.7530 | SL | 6 | -4138.75 | -4.18% | 0.7747 | 0.8493 | 0.0321 | M30 bullish FVG [0.7642, 0.7747], stochastic crossed below 20 | FVG stop 0.7530 |
| 4 | 2024-02-26 17:30 | 0.7619 | 0.7301 | 2024-02-28 11:00 | 0.7301 | SL | 25 | -4036.73 | -4.27% | 0.7801 | 0.8255 | 0.0318 | M30 bullish FVG [0.7698, 0.7801], stochastic crossed below 20 | FVG stop 0.7301 |
| 5 | 2024-03-18 10:30 | 0.7700 | 0.7438 | 2024-03-19 11:00 | 0.7721 | STOCH_EXIT | 20 | 155.49 | 0.17% | 0.7748 | 0.8224 | 0.0262 | M30 bullish FVG [0.7624, 0.7748], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 6 | 2024-03-26 17:30 | 0.8021 | 0.7763 | 2024-03-27 14:30 | 0.8033 | STOCH_EXIT | 13 | 44.89 | 0.05% | 0.7992 | 0.8536 | 0.0258 | M30 bullish FVG [0.7894, 0.7992], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 7 | 2024-04-02 15:30 | 0.9394 | 0.9061 | 2024-04-03 09:30 | 0.9600 | STOCH_EXIT | 7 | 1886.43 | 2.09% | 0.9440 | 1.0060 | 0.0333 | M30 bullish FVG [0.9352, 0.9440], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 8 | 2024-04-03 16:30 | 0.9399 | 0.9050 | 2024-04-05 12:00 | 0.9050 | SL | 29 | -3512.54 | -3.81% | 0.9440 | 1.0097 | 0.0349 | M30 bullish FVG [0.9352, 0.9440], stochastic crossed below 20 | FVG stop 0.9050 |
| 9 | 2024-04-11 17:30 | 1.0840 | 1.0417 | 2024-04-11 18:00 | 1.0417 | SL | 1 | -3555.23 | -4.00% | 1.1020 | 1.1686 | 0.0423 | M30 bullish FVG [1.0649, 1.1020], stochastic crossed below 20 | FVG stop 1.0417 |
| 10 | 2024-05-07 11:00 | 0.9274 | 0.8838 | 2024-05-07 17:00 | 0.9410 | STOCH_EXIT | 12 | 1165.25 | 1.37% | 0.9373 | 1.0146 | 0.0436 | M30 bullish FVG [0.9200, 0.9373], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 11 | 2024-08-01 21:00 | 0.8506 | 0.8203 | 2024-08-02 09:30 | 0.8648 | STOCH_EXIT | 6 | 1360.91 | 1.57% | 0.8607 | 0.9113 | 0.0303 | M30 bullish FVG [0.8499, 0.8607], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 12 | 2024-09-05 17:30 | 0.6500 | 0.6110 | 2024-09-06 09:00 | 0.6468 | STOCH_EXIT | 17 | -519.53 | -0.59% | 0.6745 | 0.7280 | 0.0390 | M30 bullish FVG [0.6476, 0.6745], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 13 | 2024-09-27 08:30 | 0.7715 | 0.7327 | 2024-09-28 15:00 | 0.7512 | STOCH_EXIT | 47 | -2379.93 | -2.73% | 0.7677 | 0.8492 | 0.0388 | M30 bullish FVG [0.7428, 0.7677], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 14 | 2024-10-16 18:00 | 0.6908 | 0.6613 | 2024-10-17 07:30 | 0.6939 | STOCH_EXIT | 13 | 296.14 | 0.35% | 0.7000 | 0.7499 | 0.0295 | M30 bullish FVG [0.6900, 0.7000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 15 | 2024-10-23 11:30 | 0.7014 | 0.6724 | 2024-10-23 12:00 | 0.7150 | STOCH_EXIT | 1 | 1572.80 | 1.84% | 0.7100 | 0.7594 | 0.0290 | M30 bullish FVG [0.7018, 0.7100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 16 | 2024-11-13 10:00 | 0.6714 | 0.6402 | 2024-11-14 07:00 | 0.6402 | SL | 28 | -4146.65 | -4.75% | 0.6417 | 0.7339 | 0.0312 | M30 bullish FVG [0.5995, 0.6417], stochastic crossed below 20 | FVG stop 0.6402 |
| 17 | 2024-11-27 19:30 | 0.6570 | 0.6265 | 2024-11-28 19:30 | 0.6524 | STOCH_EXIT | 34 | -662.09 | -0.80% | 0.6659 | 0.7180 | 0.0305 | M30 bullish FVG [0.6477, 0.6659], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 18 | 2024-12-20 10:00 | 0.6798 | 0.6557 | 2024-12-20 10:00 | 0.6870 | STOCH_EXIT | 0 | 788.51 | 0.96% | 0.6630 | 0.7279 | 0.0241 | M30 bullish FVG [0.6558, 0.6630], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 19 | 2024-12-24 10:30 | 0.7842 | 0.7526 | 2024-12-24 17:00 | 0.7526 | SL | 13 | -3433.53 | -4.13% | 0.7980 | 0.8474 | 0.0316 | M30 bullish FVG [0.7510, 0.7980], stochastic crossed below 20 | FVG stop 0.7526 |
| 20 | 2025-02-06 11:30 | 0.8993 | 0.8694 | 2025-02-06 14:00 | 0.8694 | SL | 5 | -2710.06 | -3.42% | 0.9106 | 0.9591 | 0.0299 | M30 bullish FVG [0.8988, 0.9106], stochastic crossed below 20 | FVG stop 0.8694 |
| 21 | 2025-02-13 11:30 | 0.7746 | 0.7373 | 2025-02-14 07:00 | 0.7373 | SL | 19 | -3769.02 | -4.91% | 0.7979 | 0.8492 | 0.0373 | M30 bullish FVG [0.7842, 0.7979], stochastic crossed below 20 | FVG stop 0.7373 |
| 22 | 2025-03-05 22:30 | 0.7597 | 0.7267 | 2025-03-06 08:30 | 0.7681 | STOCH_EXIT | 7 | 733.07 | 1.00% | 0.7680 | 0.8258 | 0.0330 | M30 bullish FVG [0.7498, 0.7680], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 23 | 2025-03-31 15:30 | 0.7312 | 0.7005 | 2025-04-01 07:00 | 0.7454 | STOCH_EXIT | 18 | 1359.63 | 1.84% | 0.7438 | 0.7927 | 0.0307 | M30 bullish FVG [0.7336, 0.7438], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 24 | 2025-04-21 13:30 | 0.7331 | 0.7007 | 2025-04-22 06:30 | 0.7330 | STOCH_EXIT | 21 | -84.97 | -0.11% | 0.7491 | 0.7978 | 0.0324 | M30 bullish FVG [0.7396, 0.7491], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 25 | 2025-06-05 21:30 | 0.6674 | 0.6447 | 2025-06-06 07:00 | 0.6797 | STOCH_EXIT | 6 | 1302.16 | 1.74% | 0.6663 | 0.7127 | 0.0227 | M30 bullish FVG [0.6600, 0.6663], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 26 | 2025-07-01 22:00 | 0.6721 | 0.6530 | 2025-07-02 06:30 | 0.6771 | STOCH_EXIT | 4 | 493.10 | 0.64% | 0.6712 | 0.7103 | 0.0191 | M30 bullish FVG [0.6647, 0.6712], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 27 | 2025-09-09 19:00 | 0.6125 | 0.5956 | 2025-09-10 18:00 | 0.6171 | STOCH_EXIT | 33 | 502.14 | 0.65% | 0.6062 | 0.6462 | 0.0169 | M30 bullish FVG [0.5957, 0.6062], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 28 | 2025-09-12 11:30 | 0.6336 | 0.6146 | 2025-09-13 13:00 | 0.6369 | STOCH_EXIT | 29 | 325.10 | 0.42% | 0.6414 | 0.6716 | 0.0190 | M30 bullish FVG [0.6335, 0.6414], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 29 | 2025-09-15 12:00 | 0.6418 | 0.6219 | 2025-09-15 12:30 | 0.6219 | SL | 1 | -2486.47 | -3.20% | 0.6485 | 0.6816 | 0.0199 | M30 bullish FVG [0.6431, 0.6485], stochastic crossed below 20 | FVG stop 0.6219 |
| 30 | 2025-10-01 10:00 | 0.5740 | 0.5530 | 2025-10-01 10:30 | 0.5530 | SL | 1 | -2820.66 | -3.75% | 0.6011 | 0.6159 | 0.0210 | M30 bullish FVG [0.5863, 0.6011], stochastic crossed below 20 | FVG stop 0.5530 |
| 31 | 2025-10-01 21:00 | 0.5158 | 0.4948 | 2025-10-02 06:30 | 0.5260 | STOCH_EXIT | 6 | 1355.07 | 1.88% | 0.6011 | 0.5577 | 0.0210 | M30 bullish FVG [0.5863, 0.6011], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 32 | 2025-10-09 11:30 | 0.5198 | 0.4923 | 2025-10-10 06:30 | 0.5244 | STOCH_EXIT | 25 | 579.06 | 0.78% | 0.5422 | 0.5748 | 0.0275 | M30 bullish FVG [0.5330, 0.5422], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 72640.07
- Макс: 102999.59
- Финал: 74694.31

Полная кривая — в `*_equity.csv`.
