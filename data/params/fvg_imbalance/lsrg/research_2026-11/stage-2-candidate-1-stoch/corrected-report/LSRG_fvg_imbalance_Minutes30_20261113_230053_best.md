# Бэктест LSRG (Minutes30)

- Период: 2023-11-13 23:00 — 2025-11-13 23:00
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.3 |
| MaxWaitBars | 30 |
| StochPeriod | 9 |
| UseStochExit | 1 |
| StopATR | 2 |
| TakeProfitATR | 1 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 24 |
| Выигрышных / проигрышных | 18 / 6 |
| Win rate | 75.00% |
| Gross profit / loss | 21162.88 / 8612.19 |
| Profit factor | 2.457 |
| Чистый PnL | 12550.69 (12.55%) |
| Стартовый / финальный капитал | 100000.00 / 112550.69 |
| Макс. просадка | 5809.20 (5.47%) |
| Средняя прибыль / убыток | 1175.72 / 1435.37 |
| Expectancy | 522.95 |
| Лучшая / худшая сделка | 3385.56 / -2451.81 |
| Exposure | 2.25% |
| CAGR | 6.08% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2024-01-09 11:30 | 675.2000 | 630.6838 | 2024-01-09 16:00 | 679.0000 | STOCH_EXIT | 9 | 462.19 | 0.46% | 665.8000 | 697.4581 | 22.2581 | M30 bullish FVG [659.2000, 665.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 2 | 2024-01-17 18:00 | 715.0000 | 669.7246 | 2024-01-18 12:30 | 718.8000 | STOCH_EXIT | 8 | 431.63 | 0.43% | 712.0000 | 737.6377 | 22.6377 | M30 bullish FVG [703.2000, 712.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 3 | 2024-01-23 10:30 | 745.6000 | 700.7554 | 2024-01-24 17:30 | 747.4000 | STOCH_EXIT | 33 | 142.22 | 0.14% | 739.4000 | 768.0223 | 22.4223 | M30 bullish FVG [729.8000, 739.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 4 | 2024-02-05 15:00 | 757.0000 | 718.9374 | 2024-02-06 18:00 | 760.2000 | STOCH_EXIT | 25 | 324.71 | 0.32% | 750.4000 | 776.0313 | 19.0313 | M30 bullish FVG [735.8000, 750.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 5 | 2024-02-08 12:30 | 767.2000 | 728.0860 | 2024-02-10 11:00 | 755.6000 | STOCH_EXIT | 33 | -1631.70 | -1.61% | 767.0000 | 786.7570 | 19.5570 | M30 bullish FVG [759.6000, 767.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 6 | 2024-02-14 17:00 | 811.0000 | 768.7444 | 2024-02-15 11:00 | 832.1278 | TP | 7 | 2477.36 | 2.50% | 809.8000 | 832.1278 | 21.1278 | M30 bullish FVG [789.4000, 809.8000], stochastic crossed below 20 | FVG target 832.1278 |
| 7 | 2024-02-15 16:00 | 816.4000 | 770.8198 | 2024-02-16 18:00 | 817.4000 | STOCH_EXIT | 23 | 22.89 | 0.02% | 824.2000 | 839.1901 | 22.7901 | M30 bullish FVG [813.0000, 824.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 8 | 2024-02-27 11:30 | 868.0000 | 808.3614 | 2024-02-27 12:00 | 897.8193 | TP | 1 | 3385.56 | 3.33% | 862.8000 | 897.8193 | 29.8193 | M30 bullish FVG [854.6000, 862.8000], stochastic crossed below 20 | FVG target 897.8193 |
| 9 | 2024-02-28 16:00 | 1015.4000 | 945.4499 | 2024-03-01 12:30 | 992.6000 | STOCH_EXIT | 31 | -2451.81 | -2.34% | 1023.8000 | 1050.3750 | 34.9750 | M30 bullish FVG [974.0000, 1023.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 10 | 2024-05-28 16:30 | 790.6000 | 726.0572 | 2024-05-29 11:00 | 802.0000 | STOCH_EXIT | 8 | 1378.48 | 1.34% | 797.6000 | 822.8714 | 32.2714 | M30 bullish FVG [782.6000, 797.6000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 11 | 2024-06-06 13:30 | 806.2000 | 735.7975 | 2024-06-07 10:00 | 812.4000 | STOCH_EXIT | 12 | 695.40 | 0.67% | 830.2000 | 841.4013 | 35.2013 | M30 bullish FVG [818.8000, 830.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 12 | 2024-07-25 11:00 | 763.8000 | 703.3925 | 2024-07-27 15:30 | 748.0000 | STOCH_EXIT | 46 | -2268.16 | -2.17% | 750.6000 | 794.0038 | 30.2038 | M30 bullish FVG [741.8000, 750.6000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 13 | 2024-09-03 15:00 | 595.2000 | 536.3370 | 2024-09-03 20:00 | 598.8000 | STOCH_EXIT | 10 | 516.52 | 0.50% | 614.2000 | 624.6315 | 29.4315 | M30 bullish FVG [602.0000, 614.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 14 | 2024-11-06 16:00 | 730.2000 | 689.0962 | 2024-11-07 17:30 | 730.0000 | STOCH_EXIT | 32 | -131.14 | -0.13% | 744.8000 | 750.7519 | 20.5519 | M30 bullish FVG [736.0000, 744.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 15 | 2024-11-15 09:00 | 783.4000 | 734.9229 | 2024-11-15 18:00 | 774.4000 | STOCH_EXIT | 18 | -1281.04 | -1.25% | 785.8000 | 807.6386 | 24.2386 | M30 bullish FVG [778.0000, 785.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 16 | 2024-11-28 10:30 | 705.0000 | 643.6511 | 2024-11-28 17:30 | 725.6000 | STOCH_EXIT | 14 | 2863.40 | 2.82% | 686.6000 | 735.6744 | 30.6744 | M30 bullish FVG [670.0000, 686.6000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 17 | 2024-12-30 20:00 | 832.6000 | 764.0930 | 2024-12-30 22:30 | 844.6000 | STOCH_EXIT | 5 | 1395.18 | 1.34% | 820.4000 | 866.8535 | 34.2535 | M30 bullish FVG [810.0000, 820.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 18 | 2025-06-18 17:30 | 742.8000 | 692.5919 | 2025-06-18 23:30 | 748.0000 | STOCH_EXIT | 12 | 637.01 | 0.60% | 746.6000 | 767.9040 | 25.1040 | M30 bullish FVG [739.0000, 746.6000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 19 | 2025-07-01 12:00 | 771.0000 | 724.6253 | 2025-07-03 08:00 | 772.4000 | STOCH_EXIT | 62 | 86.71 | 0.08% | 772.4000 | 794.1874 | 23.1874 | M30 bullish FVG [765.6000, 772.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 20 | 2025-07-15 11:00 | 745.2000 | 701.3752 | 2025-07-16 07:30 | 757.4000 | STOCH_EXIT | 27 | 1637.16 | 1.54% | 732.8000 | 767.1124 | 21.9124 | M30 bullish FVG [722.4000, 732.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 21 | 2025-07-16 21:00 | 765.4000 | 722.0198 | 2025-07-17 06:30 | 770.0000 | STOCH_EXIT | 6 | 540.35 | 0.50% | 760.2000 | 787.0901 | 21.6901 | M30 bullish FVG [750.0000, 760.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 22 | 2025-07-28 17:30 | 761.4000 | 719.9545 | 2025-07-29 09:30 | 773.0000 | STOCH_EXIT | 19 | 1549.09 | 1.42% | 762.4000 | 782.1227 | 20.7227 | M30 bullish FVG [752.4000, 762.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 23 | 2025-08-07 17:00 | 776.8000 | 735.5098 | 2025-08-08 08:00 | 771.6000 | STOCH_EXIT | 16 | -848.34 | -0.77% | 780.0000 | 797.4451 | 20.6451 | M30 bullish FVG [771.8000, 780.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 24 | 2025-08-11 19:30 | 806.4000 | 766.2814 | 2025-08-12 07:30 | 826.4593 | TP | 11 | 2617.03 | 2.39% | 809.0000 | 826.4593 | 20.0593 | M30 bullish FVG [794.4000, 809.0000], stochastic crossed below 20 | FVG target 826.4593 |

## Движение капитала

- Старт: 100000.00
- Мин: 98221.32
- Макс: 112550.69
- Финал: 112550.69

Полная кривая — в `*_equity.csv`.
