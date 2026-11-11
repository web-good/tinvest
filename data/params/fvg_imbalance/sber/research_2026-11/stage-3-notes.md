# Этап 3 — записанное основание сетки

Старт: stage-2 candidate-2 no-stoch rank 1 (MinGapATR=0.3, StopATR=2, TakeProfitATR=2, StochPeriod=10, HTFTrendEMA=0, floor=0, MaxWaitBars=30). Локальные значения берутся только из уже запущенных сеток: MinGapATR 0.25/0.3/0.4; StopATR 1.5/2 (выбранное значение — край сетки; расширение не планируется); TakeProfitATR 1.5/2; RequireEntryAboveFVGLower 0/1. Получится 24 сочетания.

Раздельные проверки чувствительности: StochPeriod 9/10/11; HTFTrendEMA 0/50/100/120/200; MaxWaitBars 15/30/45. Основная метрика expectancy; ограничения качества и годовой порог без изменений из plan.md. В финальный год не заглядывать до фиксации одного кандидата после этих результатов.
