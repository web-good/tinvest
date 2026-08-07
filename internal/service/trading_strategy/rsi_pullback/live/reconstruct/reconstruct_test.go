package reconstruct

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/enum"
	imodel "tinvest/internal/model"
	recmocks "tinvest/internal/service/trading_strategy/rsi_pullback/live/reconstruct/mocks"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/gazp"
	grpcmodel "tinvest/pkg/client/grpc/model"
)

// msk anchors every fixture time the same way the strategy and the marketdata assembler do.
var msk = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

func q(f float64) imodel.Quotation { return imodel.Quotation{Units: int64(f)} }

// fakeCandles отдаёт серию по запрошенному интервалу, обрезая её окном запроса. Бар
// попадает в ответ, если его ИНТЕРВАЛ пересекает окно, а не только если открытие лежит
// внутри: свеча 11:30-12:00 приходит и на запрос с from=11:45. Так ведёт себя свечной API —
// и только так тест способен отличить реализацию, которая честно отбрасывает бары, начатые
// до входа, от той, что берёт всё присланное.
type fakeCandles struct {
	m30, day []*imodel.CandleItemTechAnalyse
}

func (f *fakeCandles) GetCandles(_ context.Context, _ *string, interval int32,
	from, to *timestamppb.Timestamp, _ *int32, _ bool) ([]*imodel.CandleItemTechAnalyse, error) {
	src, span := f.m30, 30*time.Minute
	if interval == enum.Day1.ToNumberInvestAPI() {
		src, span = f.day, 24*time.Hour
	}
	var out []*imodel.CandleItemTechAnalyse
	for _, c := range src {
		if c.Time.Add(span).After(from.AsTime()) && !c.Time.After(to.AsTime()) {
			out = append(out, c)
		}
	}
	return out, nil
}

// dailiesWithRange строит n дневных баров с истинным диапазоном ровно rng и close посередине.
// ПОСЛЕДНИЙ бар стоит в MSK-полночь дня входа — то есть это бар САМОГО дня входа, ещё не
// закрывшийся к моменту сделки, и реализация обязана его отбросить. Чтобы это правило было
// различимо, его диапазон намеренно сделан вдесятеро шире (rng*10): оставленный в расчёте,
// он утащит ATR далеко вверх, и тест это увидит. Все остальные бары равной ширины, поэтому
// ATR любой длины равен rng и тест не зависит от подобранного DailyATRPeriod.
func dailiesWithRange(end time.Time, n int, rng int64) []*imodel.CandleItemTechAnalyse {
	e := end.In(msk)
	midnight := time.Date(e.Year(), e.Month(), e.Day(), 0, 0, 0, 0, msk)
	out := make([]*imodel.CandleItemTechAnalyse, n)
	for i := 0; i < n; i++ {
		t := midnight.AddDate(0, 0, -(n - 1 - i))
		w := rng
		if i == n-1 { // бар дня входа: не закрыт на момент сделки, должен быть отброшен
			w = rng * 10
		}
		half := float64(w) / 2
		out[i] = &imodel.CandleItemTechAnalyse{
			Time: t, Open: q(100), High: q(100 + half), Low: q(100 - half), Close: q(100),
			Volume: 1000, IsComplete: true,
		}
	}
	return out
}

// dayBar строит один дневной бар на дату t с серединой mid и истинным диапазоном rng.
func dayBar(t time.Time, mid float64, rng int64) *imodel.CandleItemTechAnalyse {
	half := float64(rng) / 2
	return &imodel.CandleItemTechAnalyse{
		Time: t, Open: q(mid), High: q(mid + half), Low: q(mid - half), Close: q(mid),
		Volume: 1000, IsComplete: true,
	}
}

// flat30m строит n ровных 30-минутных баров с закрытием price, последний — в end.
func flat30m(end time.Time, n int, price int64) []*imodel.CandleItemTechAnalyse {
	out := make([]*imodel.CandleItemTechAnalyse, n)
	for i := 0; i < n; i++ {
		t := end.Add(-time.Duration(n-1-i) * 30 * time.Minute)
		out[i] = &imodel.CandleItemTechAnalyse{
			Time: t, Open: q(float64(price)), High: q(float64(price)), Low: q(float64(price)), Close: q(float64(price)),
			Volume: 1000, IsComplete: true,
		}
	}
	return out
}

// bars30m строит 30-минутные бары начиная со start (шаг 30 минут), с заданными
// закрытиями (H=L=C=close на каждом баре).
func bars30m(start time.Time, closes []float64) []*imodel.CandleItemTechAnalyse {
	out := make([]*imodel.CandleItemTechAnalyse, len(closes))
	for i, c := range closes {
		t := start.Add(time.Duration(i) * 30 * time.Minute)
		out[i] = &imodel.CandleItemTechAnalyse{
			Time: t, Open: q(c), High: q(c), Low: q(c), Close: q(c), Volume: 1000, IsComplete: true,
		}
	}
	return out
}

// EntryATR обязан быть ДНЕВНЫМ ATR по будним дневным свечам, закрывшимся до
// MSK-полуночи дня входа — той же величиной, которой стратегия мерила стоп на входе.
// Часовой или 30-минутный ATR дал бы уровень в разы теснее, и восстановленная позиция
// поехала бы с защитой, которой стратегия никогда не задавала.
func TestEntryRebuildsDailyATRAsOfEntryDay(t *testing.T) {
	entryTime := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)
	now := entryTime.Add(4 * time.Hour)

	tc := recmocks.NewMockTradesClient(t)
	tc.EXPECT().GetInstrumentTrades(mock.Anything, "acc", "uid-GAZP", mock.Anything, mock.Anything).
		Return([]grpcmodel.Trade{{IsBuy: true, Date: entryTime}}, nil)

	// Дневные свечи с истинным диапазоном ровно 10 у каждой: ATR любой длины даёт 10.
	cc := &fakeCandles{
		day: dailiesWithRange(entryTime, 60, 10),
		m30: flat30m(now, 20, 100),
	}

	p := gazp.DefaultParams()
	got, err := Entry(context.Background(), tc, cc, "acc", "uid-GAZP", "GAZP", 100, p, now)
	if err != nil {
		t.Fatalf("Entry: %v", err)
	}
	if got.EntryATR < 9.5 || got.EntryATR > 10.5 {
		t.Fatalf("EntryATR = %v, want ~10 (дневной ATR, не внутридневной)", got.EntryATR)
	}
	if !got.EntryTime.Equal(entryTime) {
		t.Fatalf("EntryTime = %v, want %v (последняя BUY-сделка)", got.EntryTime, entryTime)
	}
	if got.EntryPrice != 100 {
		t.Fatalf("EntryPrice = %v, want 100 (средняя цена покупки от брокера)", got.EntryPrice)
	}
}

// Выходные дневные свечи MOEX в 2.9-3.8 раза уже будних: оставленные в расчёте, они
// занижают ATR на 9-16% и вместе с ним весь защитный уровень. Здесь наоборот — выходным
// барам придан диапазон в разы ШИРЕ будних (40 против 10), так что если реализация их не
// отфильтрует, ATR заметно уйдёт вверх от 10; тест различает правильную и неправильную
// реализацию именно этим контрастом, а не совпадением значений.
func TestEntrySkipsWeekendDailies(t *testing.T) {
	entryTime := time.Date(2026, 3, 10, 11, 30, 0, 0, msk) // Tuesday
	now := entryTime.Add(4 * time.Hour)

	tc := recmocks.NewMockTradesClient(t)
	tc.EXPECT().GetInstrumentTrades(mock.Anything, "acc", "uid-GAZP", mock.Anything, mock.Anything).
		Return([]grpcmodel.Trade{{IsBuy: true, Date: entryTime}}, nil)

	midnight := time.Date(entryTime.Year(), entryTime.Month(), entryTime.Day(), 0, 0, 0, 0, msk)
	var daily []*imodel.CandleItemTechAnalyse
	for i := 1; i <= 40; i++ {
		d := midnight.AddDate(0, 0, -i)
		rng := int64(10)
		if wd := d.Weekday(); wd == time.Saturday || wd == time.Sunday {
			rng = 40
		}
		daily = append(daily, dayBar(d, 100, rng))
	}
	sort.Slice(daily, func(a, b int) bool { return daily[a].Time.Before(daily[b].Time) })

	cc := &fakeCandles{day: daily, m30: flat30m(now, 20, 100)}

	p := gazp.DefaultParams()
	got, err := Entry(context.Background(), tc, cc, "acc", "uid-GAZP", "GAZP", 100, p, now)
	if err != nil {
		t.Fatalf("Entry: %v", err)
	}
	if got.EntryATR < 9.5 || got.EntryATR > 10.5 {
		t.Fatalf("EntryATR = %v, want ~10 (выходные бары должны быть исключены из расчёта)", got.EntryATR)
	}
}

// Цель восстанавливается из параметров тикера, иначе TP по close-модели не сработает
// никогда и сделку дотянет только стоп.
func TestEntryRebuildsTakeProfitFromParams(t *testing.T) {
	entryTime := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)
	now := entryTime.Add(4 * time.Hour)

	tc := recmocks.NewMockTradesClient(t)
	tc.EXPECT().GetInstrumentTrades(mock.Anything, "acc", "uid-GAZP", mock.Anything, mock.Anything).
		Return([]grpcmodel.Trade{{IsBuy: true, Date: entryTime}}, nil)

	cc := &fakeCandles{
		day: dailiesWithRange(entryTime, 60, 10), // ATR = 10
		m30: flat30m(now, 20, 100),
	}

	p := gazp.DefaultParams() // TPDailyATR = 0.7
	got, err := Entry(context.Background(), tc, cc, "acc", "uid-GAZP", "GAZP", 100, p, now)
	if err != nil {
		t.Fatalf("Entry: %v", err)
	}
	want := 100 + p.TPDailyATR*10
	if got.TakeProfit < want-0.5 || got.TakeProfit > want+0.5 {
		t.Fatalf("TakeProfit = %v, want ~%v (entry + TPDailyATR*dailyATR)", got.TakeProfit, want)
	}
}

// MaxFav — максимум ЗАКРЫТИЙ 30m с момента входа: движок марк-ту-маркетит позицию по
// close, и трейл обязан отсчитываться от той же величины. Пик серии закрытий (120)
// намеренно отличается и от EntryPrice (100), и от последнего закрытия (110): тест
// различает реализацию, которая честно сканирует всю историю после входа, от той, что
// молча берёт цену входа или только последний бар.
func TestEntryRebuildsMaxFavFromClosesAfterEntry(t *testing.T) {
	entryTime := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)
	now := entryTime.Add(3 * time.Hour)

	tc := recmocks.NewMockTradesClient(t)
	tc.EXPECT().GetInstrumentTrades(mock.Anything, "acc", "uid-GAZP", mock.Anything, mock.Anything).
		Return([]grpcmodel.Trade{{IsBuy: true, Date: entryTime}}, nil)

	// Сетка баров сдвинута так, что вход (11:30) попадает ВНУТРЬ бара 11:15, и API отдаёт
	// этот бар на запрос с from=11:30 — он пересекает окно. Его закрытие (200) намеренно
	// выше всего, что случилось после входа: бар НАЧАЛСЯ до сделки, позиция его не прожила,
	// и учитывать его нельзя. Реализация, которая берёт всё присланное, поставит MaxFav=200
	// и уведёт трейл на уровень, которого позиция никогда не видела.
	// 9:45=130, 10:15=150, 10:45=140, 11:15=200 (бар входа), затем 105, 120, 115, 112, 110.
	closes := []float64{130, 150, 140, 200, 105, 120, 115, 112, 110}
	cc := &fakeCandles{
		day: dailiesWithRange(entryTime, 60, 10),
		m30: bars30m(time.Date(2026, 3, 10, 9, 45, 0, 0, msk), closes),
	}

	p := gazp.DefaultParams()
	got, err := Entry(context.Background(), tc, cc, "acc", "uid-GAZP", "GAZP", 100, p, now)
	if err != nil {
		t.Fatalf("Entry: %v", err)
	}
	if got.MaxFav != 120 {
		t.Fatalf("MaxFav = %v, want 120 (максимум закрытий с момента входа)", got.MaxFav)
	}
}

// Дневных свечей меньше, чем нужно для ATR(period+1) — восстановить единицу риска нечем.
// Это ошибка, а не «стоп с нулевым ATR»: DesiredStop при нулевом ATR отдаёт (0, ""), то есть
// раннер молча повёл бы многодневную позицию вообще без биржевой защиты, считая, что так и
// задумано. Отказ оставляет решение человеку — уходит алерт, позиция не трогается.
func TestEntryFailsWhenDailyATRCannotBeRebuilt(t *testing.T) {
	entryTime := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)
	now := entryTime.Add(4 * time.Hour)

	tc := recmocks.NewMockTradesClient(t)
	tc.EXPECT().GetInstrumentTrades(mock.Anything, "acc", "uid-GAZP", mock.Anything, mock.Anything).
		Return([]grpcmodel.Trade{{IsBuy: true, Date: entryTime}}, nil)

	p := gazp.DefaultParams() // DailyATRPeriod = 14 -> нужно >= 15 будних баров
	cc := &fakeCandles{
		day: dailiesWithRange(entryTime, 6, 10),
		m30: flat30m(now, 20, 100),
	}

	if _, err := Entry(context.Background(), tc, cc, "acc", "uid-GAZP", "GAZP", 100, p, now); err == nil {
		t.Fatal("Entry: want error when the daily ATR cannot be rebuilt, got nil")
	}
}

// Без BUY-сделки восстанавливать нечего — ошибка, а не молчаливый нулевой стейт.
func TestEntryFailsWithoutABuyFill(t *testing.T) {
	now := time.Date(2026, 3, 10, 15, 30, 0, 0, msk)

	tc := recmocks.NewMockTradesClient(t)
	tc.EXPECT().GetInstrumentTrades(mock.Anything, "acc", "uid-GAZP", mock.Anything, mock.Anything).
		Return([]grpcmodel.Trade{{IsBuy: false, Date: now.Add(-time.Hour)}}, nil)

	cc := &fakeCandles{}
	p := gazp.DefaultParams()
	_, err := Entry(context.Background(), tc, cc, "acc", "uid-GAZP", "GAZP", 100, p, now)
	if err == nil {
		t.Fatal("Entry: want error when there is no BUY fill, got nil")
	}
}
