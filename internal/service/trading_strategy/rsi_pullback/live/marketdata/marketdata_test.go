package marketdata

import (
	"context"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/enum"
	imodel "tinvest/internal/model"
)

var msk = func() *time.Location {
	l, _ := time.LoadLocation("Europe/Moscow")
	return l
}()

// fakeClient отдаёт свечи из заранее построенных серий, обрезая их запрошенным окном —
// как делает настоящий API. Записывает окна запросов, чтобы тест мог проверить чанкование.
type fakeClient struct {
	m30, day []*imodel.CandleItemTechAnalyse
	windows  []int32 // интервалы запросов в порядке вызова
}

func (f *fakeClient) GetCandles(_ context.Context, _ *string, interval int32,
	from, to *timestamppb.Timestamp, limit *int32, _ bool) ([]*imodel.CandleItemTechAnalyse, error) {
	f.windows = append(f.windows, interval)
	src := f.m30
	if interval == enum.Day1.ToNumberInvestAPI() {
		src = f.day
	}
	var out []*imodel.CandleItemTechAnalyse
	for _, c := range src {
		if !c.Time.Before(from.AsTime()) && !c.Time.After(to.AsTime()) {
			out = append(out, c)
		}
	}
	return out, nil
}

// Quotation в CandleItemTechAnalyse — значение, не указатель.
func q(v int64) imodel.Quotation { return imodel.Quotation{Units: v} }

// tradingHour reports whether t falls in the MSK session window (09:00-23:00) the fake
// candle tape trades in. No day-of-week check: per the brief, MOEX weekend bars exist in
// the candle cache and must NOT be filtered by the assembler, so the fixture must produce
// them too (see TestAssembleKeepsWeekendBarsInTheWindow).
func tradingHour(t time.Time) bool {
	h := t.In(msk).Hour()
	return h >= 9 && h < 23
}

// series30m builds n 30-minute bars ending at `end` (the newest bar's open-time), stepping
// back one slot at a time and keeping only slots inside the MSK trading session — about 28
// bars/day, so 403 bars span roughly two calendar weeks, matching real UGLD density (see
// the task brief: "403 бара 30m — это около двух календарных недель"). A dense, gapless
// 48-bars/day tape would let a single chunkDays-wide request satisfy any registered
// ticker's lookback outright, defeating the point of TestAssembleChunksUntilLookbackIsFilled.
// Weekend days are not excluded, only off-session hours — see tradingHour.
func series30m(end time.Time, n int) []*imodel.CandleItemTechAnalyse {
	rev := make([]*imodel.CandleItemTechAnalyse, 0, n)
	for t := end; len(rev) < n; t = t.Add(-30 * time.Minute) {
		if !tradingHour(t) {
			continue
		}
		rev = append(rev, &imodel.CandleItemTechAnalyse{
			Time: t, Open: q(100), High: q(101), Low: q(99), Close: q(100),
			Volume: 1000, IsComplete: true,
		})
	}
	out := make([]*imodel.CandleItemTechAnalyse, n)
	for i, c := range rev {
		out[n-1-i] = c
	}
	return out
}

// setExtent overwrites the High/Low of the bar at exactly time t (comparing instants, so
// callers may build t in any location). Fails the test if no bar has that open-time.
func setExtent(t *testing.T, bars []*imodel.CandleItemTechAnalyse, at time.Time, high, low int64) {
	t.Helper()
	for _, b := range bars {
		if b.Time.Equal(at) {
			b.High = q(high)
			b.Low = q(low)
			return
		}
	}
	t.Fatalf("setExtent: no bar at %v", at)
}

// seriesDaily строит n дневных баров, последний открывается в MSK-полночь дня end.
func seriesDaily(end time.Time, n int) []*imodel.CandleItemTechAnalyse {
	e := end.In(msk)
	midnight := time.Date(e.Year(), e.Month(), e.Day(), 0, 0, 0, 0, msk)
	out := make([]*imodel.CandleItemTechAnalyse, 0, n)
	for i := n - 1; i >= 0; i-- {
		out = append(out, &imodel.CandleItemTechAnalyse{
			Time: midnight.AddDate(0, 0, -i), Open: q(100), High: q(105), Low: q(95), Close: q(100),
			Volume: 100000, IsComplete: true,
		})
	}
	return out
}

// Окно на 403 бара (UGLD) не помещается в один запрос 30m-свечей: API ограничивает
// окно примерно тремя неделями, а каникулы MOEX растягивают эти бары за кап. Сборщик
// обязан догружать чанками, а не молча возвращать короткое окно.
func TestAssembleChunksUntilLookbackIsFilled(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, msk)
	f := &fakeClient{m30: series30m(now.Add(-30*time.Minute), 900), day: seriesDaily(now, 90)}
	md, err := Assemble(context.Background(), f, "uid", 403, now)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if len(md.Closes) != 403 {
		t.Fatalf("len(Closes) = %d, want 403", len(md.Closes))
	}
	m30calls := 0
	for _, iv := range f.windows {
		if iv == enum.Minutes30.ToNumberInvestAPI() {
			m30calls++
		}
	}
	if m30calls < 2 {
		t.Fatalf("30m requests = %d, want >= 2 (chunked fetch)", m30calls)
	}
}

// Недобор баров — это ошибка, а не короткое окно: ядро на окне короче EMASlow
// возвращает нулевую серию EMA и молча заваливает трендовый гейт на весь прогон.
func TestAssembleFailsWhenHistoryIsShort(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, msk)
	f := &fakeClient{m30: series30m(now.Add(-30*time.Minute), 50), day: seriesDaily(now, 90)}
	if _, err := Assemble(context.Background(), f, "uid", 403, now); err == nil {
		t.Fatal("Assemble returned nil error on a short history, want an error")
	}
}

// Формирующийся бар не должен попадать в окно: решение принимается по закрытым барам.
func TestAssembleDropsIncompleteBar(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, msk)
	bars := series30m(now.Add(-30*time.Minute), 500)
	forming := *bars[len(bars)-1]
	forming.Time = now
	forming.IsComplete = false
	f := &fakeClient{m30: append(bars, &forming), day: seriesDaily(now, 90)}
	md, err := Assemble(context.Background(), f, "uid", 403, now)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if last := md.Times[len(md.Times)-1]; !last.Before(now) {
		t.Fatalf("last bar time = %v, want strictly before now (%v)", last, now)
	}
}

// Дневная свеча текущего дня закрыться не успела: если она попадёт в Daily*, дневной ATR
// и обе границы гейта дня будут посчитаны с заглядыванием в будущее.
func TestAssembleExcludesTodayFromDailies(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, msk)
	f := &fakeClient{m30: series30m(now.Add(-30*time.Minute), 900), day: seriesDaily(now, 90)}
	md, err := Assemble(context.Background(), f, "uid", 403, now)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if len(md.DailyTimes) == 0 {
		t.Fatal("DailyTimes is empty, want the completed dailies")
	}
	last := md.DailyTimes[len(md.DailyTimes)-1].In(msk)
	if last.Year() == now.Year() && last.YearDay() == now.YearDay() {
		t.Fatalf("last daily = %v, want strictly before today", last)
	}
}

// Выходные бары MOEX обязаны остаться в 30m-окне: ядро отсеивает их само (isWeekend),
// а бэктест видит их в окне. Фильтрация здесь сломала бы совпадение с бэктестом.
func TestAssembleKeepsWeekendBarsInTheWindow(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, msk)
	f := &fakeClient{m30: series30m(now.Add(-30*time.Minute), 900), day: seriesDaily(now, 90)}
	md, err := Assemble(context.Background(), f, "uid", 403, now)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	weekend := 0
	for _, tm := range md.Times {
		if wd := tm.In(msk).Weekday(); wd == time.Saturday || wd == time.Sunday {
			weekend++
		}
	}
	if weekend == 0 {
		t.Fatal("no weekend bars in the window: the assembler must not filter them")
	}
}

// TodayHigh/Low обязаны считаться правилом движка (граница MSK-дня), а не похожим кодом,
// который берёт экстремум по всему окну. Вчерашний бар получает заведомо более широкий
// диапазон (500/10), чем любой сегодняшний (105-110/90-97): реализация, которая сканирует
// всё 403-бара окно вместо бары одного MSK-дня, поймает вчерашний экстремум и провалится.
func TestAssembleFillsTodayExtentFromEngine(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, msk)
	bars := series30m(now.Add(-30*time.Minute), 900)

	// Вчера (9 марта) — намеренно шире ожидаемого сегодняшнего диапазона.
	setExtent(t, bars, time.Date(2026, 3, 9, 15, 0, 0, 0, msk), 500, 10)

	// Сегодня (10 марта), все бары с открытия сессии (09:00) до последнего завершённого
	// (11:30) — узкий, точно известный диапазон.
	today := []struct {
		hour, min int
		high, low int64
	}{
		{9, 0, 105, 95},
		{9, 30, 110, 90}, // максимум High=110, минимум Low=90 — ожидаемые TodayHigh/TodayLow
		{10, 0, 103, 97},
		{10, 30, 108, 92},
		{11, 0, 106, 94},
		{11, 30, 104, 96},
	}
	for _, b := range today {
		setExtent(t, bars, time.Date(2026, 3, 10, b.hour, b.min, 0, 0, msk), b.high, b.low)
	}

	f := &fakeClient{m30: bars, day: seriesDaily(now, 90)}
	md, err := Assemble(context.Background(), f, "uid", 403, now)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if md.TodayHigh != 110 || md.TodayLow != 90 {
		t.Fatalf("TodayHigh/TodayLow = %v/%v, want 110/90 (today's MSK-day bars only, not the whole window)",
			md.TodayHigh, md.TodayLow)
	}
}

// cur, переданный в backtest.AssembleMarketData, обязан быть временем открытия ПОСЛЕДНЕГО
// ЗАВЕРШЁННОГО 30m-бара, а не now: now только что перевалило за полночь МСК (00:10 10
// марта), а последний завершённый бар — ещё 22:30 9 марта, т.е. день 9 марта по часам
// последнего бара ЕЩЁ НЕ ЗАКРЫТ. Если бы cur = now, граница дня сдвинулась бы на полночь
// 10 марта, и дневная свеча за 9 марта ошибочно попала бы в Daily* — заглядывание в
// будущее относительно того, что реально видел последний бар.
func TestAssembleUsesLastBarTimeNotNowForDailyBoundary(t *testing.T) {
	now := time.Date(2026, 3, 10, 0, 10, 0, 0, msk)
	lastBar := time.Date(2026, 3, 9, 22, 30, 0, 0, msk)
	f := &fakeClient{m30: series30m(lastBar, 900), day: seriesDaily(now, 90)}
	md, err := Assemble(context.Background(), f, "uid", 403, now)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	mar9 := time.Date(2026, 3, 9, 0, 0, 0, 0, msk)
	for _, dt := range md.DailyTimes {
		if dt.In(msk).Equal(mar9) {
			t.Fatalf("DailyTimes includes the 9 March daily (%v); want it excluded — "+
				"the day is not closed as of the last completed bar (%v)", dt, lastBar)
		}
	}
}
