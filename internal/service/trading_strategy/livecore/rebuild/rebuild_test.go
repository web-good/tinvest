package rebuild

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	imodel "tinvest/internal/model"
	grpcmodel "tinvest/pkg/client/grpc/model"
	"tinvest/pkg/indicators"
)

var msk = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// fakeTrades отдаёт заданные сделки и запоминает окно запроса.
type fakeTrades struct {
	trades   []grpcmodel.Trade
	err      error
	from, to time.Time
}

func (f *fakeTrades) GetInstrumentTrades(_ context.Context, _, _ string, from, to time.Time) ([]grpcmodel.Trade, error) {
	f.from, f.to = from, to
	return f.trades, f.err
}

// fakeDaily отдаёт фиксированный набор дневных баров и запоминает окно запроса.
type fakeDaily struct {
	bars     []*imodel.CandleItemTechAnalyse
	from, to time.Time
}

func (f *fakeDaily) GetCandles(_ context.Context, _ *string, _ int32,
	from, to *timestamppb.Timestamp, _ *int32, _ bool) ([]*imodel.CandleItemTechAnalyse, error) {
	f.from, f.to = from.AsTime(), to.AsTime()
	return f.bars, nil
}

func q(f float64) imodel.Quotation { return imodel.Quotation{Units: int64(f)} }

func bar(t time.Time, low, high, closePx float64) *imodel.CandleItemTechAnalyse {
	return &imodel.CandleItemTechAnalyse{
		Time: t, Open: q(closePx), High: q(high), Low: q(low), Close: q(closePx),
		Volume: 1000, IsComplete: true,
	}
}

func TestLastBuyTimePicksLatestBuy(t *testing.T) {
	now := time.Date(2026, 3, 10, 15, 0, 0, 0, msk)
	early := now.Add(-48 * time.Hour)
	late := now.Add(-5 * time.Hour)
	sellLater := now.Add(-time.Hour) // SELL позже обеих BUY не должна выиграть

	tc := &fakeTrades{trades: []grpcmodel.Trade{
		{IsBuy: true, Date: early},
		{IsBuy: false, Date: sellLater},
		{IsBuy: true, Date: late},
	}}
	got, err := LastBuyTime(context.Background(), tc, "acc", "uid", now)
	if err != nil {
		t.Fatalf("LastBuyTime: %v", err)
	}
	if !got.Equal(late) {
		t.Fatalf("got %v, want %v (поздняя BUY)", got, late)
	}
	if !tc.to.Equal(now) {
		t.Fatalf("to = %v, want %v", tc.to, now)
	}
	if want := now.AddDate(0, 0, -TradesLookbackDays); !tc.from.Equal(want) {
		t.Fatalf("from = %v, want %v (окно %d дней)", tc.from, want, TradesLookbackDays)
	}
}

func TestLastBuyTimeWithoutBuyReturnsSentinel(t *testing.T) {
	now := time.Date(2026, 3, 10, 15, 0, 0, 0, msk)
	tc := &fakeTrades{trades: []grpcmodel.Trade{{IsBuy: false, Date: now.Add(-time.Hour)}}}
	_, err := LastBuyTime(context.Background(), tc, "acc", "uid", now)
	if !errors.Is(err, ErrNoBuyFill) {
		t.Fatalf("err = %v, want ErrNoBuyFill", err)
	}
}

func TestLastBuyTimeWrapsClientError(t *testing.T) {
	boom := errors.New("boom")
	tc := &fakeTrades{err: boom}
	_, err := LastBuyTime(context.Background(), tc, "acc", "uid", time.Now())
	if !errors.Is(err, boom) || errors.Is(err, ErrNoBuyFill) {
		t.Fatalf("err = %v, want wrapped boom and not ErrNoBuyFill", err)
	}
	if err.Error() != "trades: boom" {
		t.Fatalf("err text = %q, want %q", err.Error(), "trades: boom")
	}
}

// Вход во вторник 2026-03-10. Смещения в днях назад от MSK-полуночи дня входа:
// будни: 1 (пн 9.03), 4 (пт 6.03), 5, 6, 7, 8 (пн 2.03), 11 (пт 27.02), 12;
// выходные: 2, 3, 9, 10; смещение 0 — сам день входа.
// Выходным и дню входа придан диапазон, сильно отличающийся от будних, так что попадание
// любого из них в расчёт сдвигает ATR.
var (
	weekdayOffsets = []int{12, 11, 8, 7, 6, 5, 4, 1} // от старых к новым
	weekendOffsets = []int{10, 9, 3, 2}
)

func TestWeekdayDailyATRBeforeSkipsWeekendsAndEntryDay(t *testing.T) {
	entryTime := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)
	midnight := time.Date(2026, 3, 10, 0, 0, 0, 0, msk)
	day := func(off int) time.Time { return midnight.AddDate(0, 0, -off) }

	var bars []*imodel.CandleItemTechAnalyse
	var highs, lows, closes []float64
	for i, off := range weekdayOffsets {
		// разные диапазоны и закрытия, чтобы TR был нетривиальным
		closePx := 100 + float64(i%3)
		low, high := closePx-float64(3+i), closePx+float64(2+i%4)
		bars = append(bars, bar(day(off), low, high, closePx))
		highs, lows, closes = append(highs, high), append(lows, low), append(closes, closePx)
	}
	for _, off := range weekendOffsets {
		bars = append(bars, bar(day(off), 60, 160, 110))
	}
	bars = append(bars, bar(day(0), 50, 200, 90)) // день входа: не закрыт на момент сделки

	const period = 5
	want := indicators.ATR(highs, lows, closes, period)
	// контроль: если взять вообще все бары (выходные и день входа), результат другой
	allH := append(append([]float64{}, highs...), 160, 160, 160, 160, 200)
	allL := append(append([]float64{}, lows...), 60, 60, 60, 60, 50)
	allC := append(append([]float64{}, closes...), 110, 110, 110, 110, 90)
	allIn := indicators.ATR(allH, allL, allC, period)
	if want == allIn {
		t.Fatalf("фикстура не различает фильтрацию: %v == %v", want, allIn)
	}

	// бары отдаются вперемешку, реализация обязана сама отсортировать по времени
	shuffled := []*imodel.CandleItemTechAnalyse{}
	for i := len(bars) - 1; i >= 0; i-- {
		shuffled = append(shuffled, bars[i])
	}
	fc := &fakeDaily{bars: shuffled}

	got, err := WeekdayDailyATRBefore(context.Background(), fc, "uid", entryTime, period)
	if err != nil {
		t.Fatalf("WeekdayDailyATRBefore: %v", err)
	}
	if got != want {
		t.Fatalf("ATR = %v, want %v (только будни до полуночи дня входа)", got, want)
	}
	if !fc.to.Equal(entryTime) {
		t.Fatalf("to = %v, want %v", fc.to, entryTime)
	}
	if wantFrom := entryTime.AddDate(0, 0, -DailyFetchDays); !fc.from.Equal(wantFrom) {
		t.Fatalf("from = %v, want %v (окно %d дней)", fc.from, wantFrom, DailyFetchDays)
	}
}

func TestWeekdayDailyATRBeforeShortHistoryIsZero(t *testing.T) {
	entryTime := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)
	midnight := time.Date(2026, 3, 10, 0, 0, 0, 0, msk)

	// ровно period будних баров (нужно period+1) плюс выходные, которые не считаются
	const period = 5
	var bars []*imodel.CandleItemTechAnalyse
	for _, off := range []int{8, 7, 6, 5, 4} {
		bars = append(bars, bar(midnight.AddDate(0, 0, -off), 95, 105, 100))
	}
	for _, off := range weekendOffsets {
		bars = append(bars, bar(midnight.AddDate(0, 0, -off), 95, 105, 100))
	}
	fc := &fakeDaily{bars: bars}

	got, err := WeekdayDailyATRBefore(context.Background(), fc, "uid", entryTime, period)
	if err != nil || got != 0 {
		t.Fatalf("got (%v, %v), want (0, nil)", got, err)
	}

	// period <= 0 — ноль без единого запроса свечей
	fc2 := &fakeDaily{bars: bars}
	got, err = WeekdayDailyATRBefore(context.Background(), fc2, "uid", entryTime, 0)
	if err != nil || got != 0 || !fc2.to.IsZero() {
		t.Fatalf("period 0: got (%v, %v), to=%v; want (0, nil) без запроса", got, err, fc2.to)
	}
}
