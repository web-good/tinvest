package backtest

import (
	"strings"
	"testing"
	"time"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestWalkForwardFolds(t *testing.T) {
	tests := []struct {
		name                    string
		from, to                time.Time
		trainMonths, testMonths int
		wantFolds               int
		wantErr                 bool
	}{
		{
			name: "24m/12train/3test -> 4 folds",
			from: date(2024, time.January, 1), to: date(2026, time.January, 1),
			trainMonths: 12, testMonths: 3, wantFolds: 4,
		},
		{
			name: "9m/3train/3test -> 2 folds",
			from: date(2025, time.January, 1), to: date(2025, time.October, 1),
			trainMonths: 3, testMonths: 3, wantFolds: 2,
		},
		{
			name: "train+test exceed window -> error",
			from: date(2025, time.January, 1), to: date(2025, time.April, 1),
			trainMonths: 3, testMonths: 3, wantErr: true,
		},
		{
			name: "zero train -> error",
			from: date(2025, time.January, 1), to: date(2025, time.October, 1),
			trainMonths: 0, testMonths: 3, wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			folds, err := walkForwardFolds(tc.from, tc.to, tc.trainMonths, tc.testMonths)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error, got %d folds", len(folds))
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(folds) != tc.wantFolds {
				t.Fatalf("folds = %d, want %d", len(folds), tc.wantFolds)
			}
		})
	}
}

func TestWalkForwardFoldsBoundaries(t *testing.T) {
	folds, err := walkForwardFolds(date(2025, time.January, 1), date(2025, time.October, 1), 3, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Fold 0: train Jan-Apr, test Apr-Jul. Fold 1: train Apr-Jul, test Jul-Oct.
	if !folds[0].trainFrom.Equal(date(2025, time.January, 1)) || !folds[0].testTo.Equal(date(2025, time.July, 1)) {
		t.Errorf("fold0 = %+v", folds[0])
	}
	if !folds[1].trainFrom.Equal(date(2025, time.April, 1)) || !folds[1].testTo.Equal(date(2025, time.October, 1)) {
		t.Errorf("fold1 = %+v", folds[1])
	}
}

func TestSliceByRange(t *testing.T) {
	mk := func(h int) backtest.Candle {
		return backtest.Candle{Time: date(2025, time.January, 1).Add(time.Duration(h) * time.Hour)}
	}
	candles := []backtest.Candle{mk(0), mk(1), mk(2), mk(3), mk(4)}
	got := sliceByRange(candles, date(2025, time.January, 1).Add(time.Hour), date(2025, time.January, 1).Add(3*time.Hour))
	// half-open [1h, 3h): expect bars at h=1 and h=2.
	if len(got) != 2 || !got[0].Time.Equal(mk(1).Time) || !got[1].Time.Equal(mk(2).Time) {
		t.Fatalf("sliceByRange = %+v", got)
	}
}

func TestTradesEnteredFrom(t *testing.T) {
	tr := func(h int) backtest.Trade {
		return backtest.Trade{EntryTime: date(2025, time.January, 1).Add(time.Duration(h) * time.Hour)}
	}
	trades := []backtest.Trade{tr(0), tr(1), tr(2), tr(3)}
	boundary := date(2025, time.January, 1).Add(2 * time.Hour)
	got := tradesEnteredFrom(trades, boundary)
	// Keep entries at/after boundary: h=2 and h=3.
	if len(got) != 2 || !got[0].EntryTime.Equal(tr(2).EntryTime) {
		t.Fatalf("tradesEnteredFrom = %+v", got)
	}
}

func TestSumPnL(t *testing.T) {
	trades := []backtest.Trade{{PnL: 100}, {PnL: -40}, {PnL: 10}}
	if got := sumPnL(trades); got != 70 {
		t.Fatalf("sumPnL = %v, want 70", got)
	}
}

func TestTradeReplayDrawdownPct(t *testing.T) {
	// Equity from 1000: +200 -> 1200 (peak), -360 -> 840. DD = (1200-840)/1200 = 0.30.
	trades := []backtest.Trade{{PnL: 200}, {PnL: -360}, {PnL: 60}}
	got := tradeReplayDrawdownPct(trades, 1000)
	if got < 0.2999 || got > 0.3001 {
		t.Fatalf("tradeReplayDrawdownPct = %v, want ~0.30", got)
	}
	if tradeReplayDrawdownPct(nil, 1000) != 0 {
		t.Fatalf("empty trades should give 0 drawdown")
	}
	if tradeReplayDrawdownPct(trades, 0) != 0 {
		t.Fatalf("zero cash should give 0 drawdown (guard)")
	}
}

func TestCompoundReturns(t *testing.T) {
	// (1+0.10)(1-0.05)(1+0.20) - 1 = 0.254.
	got := compoundReturns([]float64{0.10, -0.05, 0.20})
	if got < 0.2539 || got > 0.2541 {
		t.Fatalf("compoundReturns = %v, want ~0.254", got)
	}
	if compoundReturns(nil) != 0 {
		t.Fatalf("empty should give 0")
	}
}

func TestParamStability(t *testing.T) {
	folds := []WalkForwardFold{
		{WinnerRows: []backtest.ParamLine{{Name: "RSIPeriod", Value: "6"}, {Name: "StopATRMult", Value: "0.8"}}},
		{WinnerRows: []backtest.ParamLine{{Name: "RSIPeriod", Value: "6"}, {Name: "StopATRMult", Value: "1.2"}}},
		{Note: "0 OOS-сделок"}, // skipped fold: no WinnerRows, must be ignored
	}
	stable, varied := paramStability(folds)
	if stable["RSIPeriod"] != "6" {
		t.Errorf("RSIPeriod should be stable at 6, got %q", stable["RSIPeriod"])
	}
	if _, ok := stable["StopATRMult"]; ok {
		t.Errorf("StopATRMult should not be stable")
	}
	got := varied["StopATRMult"]
	if len(got) != 2 || got[0] != "0.8" || got[1] != "1.2" {
		t.Errorf("StopATRMult varied = %v, want [0.8 1.2]", got)
	}
}

// alternatingStrategy buys whenever flat and sells the next bar — it produces a trade
// roughly every two bars across the whole series, at deterministic entry times. Lookback
// 1 keeps warm-up trivial. It ignores params, so any swept grid value yields the same run.
type alternatingStrategy struct{}

func (alternatingStrategy) Ticker() string { return "TEST" }
func (alternatingStrategy) Lookback() int  { return 1 }
func (alternatingStrategy) Decide(md strategy.MarketData) model.Signal {
	if md.Position == nil {
		return model.Signal{Kind: model.SignalBuy}
	}
	return model.Signal{Kind: model.SignalSell, Reason: "TP"}
}

type fakeParams struct{ Threshold int }

func fakeBinding() Binding {
	return Binding{
		DefaultParams: func() any { return fakeParams{Threshold: 1} },
		Build:         func(any) strategy.Strategy { return alternatingStrategy{} },
		ParseParams:   func([]byte) (any, error) { return fakeParams{}, nil },
	}
}

// genHourly builds 1h candles over [from, to) with a slight up-drift so some trades
// profit (keeps PooledMetrics non-degenerate).
func genHourly(from, to time.Time) []backtest.Candle {
	var out []backtest.Candle
	price := 100.0
	for ts, i := from, 0; ts.Before(to); ts, i = ts.Add(time.Hour), i+1 {
		if i%2 == 0 {
			price += 1
		} else {
			price -= 0.5
		}
		out = append(out, backtest.Candle{Time: ts, Open: price, High: price + 1, Low: price - 1, Close: price, Volume: 1})
	}
	return out
}

func TestRunWalkForward(t *testing.T) {
	from, to := date(2025, time.January, 1), date(2025, time.October, 1) // 9 months
	candles := genHourly(from, to)
	phases := []Phase{{Grid: Grid{"Threshold": {1, 2}}}}
	cfg := backtest.Config{InitialCash: 100000, Fraction: 1, Commission: 0.0005, Lot: 1}

	s, err := RunWalkForward(fakeBinding(), phases, candles, nil, nil, cfg,
		"profit_factor", 0, from, to, 3, 3)
	if err != nil {
		t.Fatalf("RunWalkForward: %v", err)
	}
	if len(s.Folds) != 2 {
		t.Fatalf("folds = %d, want 2", len(s.Folds))
	}
	var oosSum int
	for _, f := range s.Folds {
		if f.Note != "" {
			t.Fatalf("fold %d unexpectedly skipped: %s", f.Index, f.Note)
		}
		if f.OOSTrades == 0 {
			t.Fatalf("fold %d has no OOS trades", f.Index)
		}
		if f.WinnerRows == nil {
			t.Fatalf("fold %d missing winner rows", f.Index)
		}
		oosSum += f.OOSTrades
	}
	if s.PooledOOS.TotalTrades != oosSum {
		t.Fatalf("pooled trades = %d, want sum of folds %d", s.PooledOOS.TotalTrades, oosSum)
	}
}

// hungryStrategy's lookback grows with BOTH params, so a grid-aware bound has to accumulate
// across fields instead of taking the best single override. Trading behaviour matches
// alternatingStrategy so runs stay non-degenerate.
type hungryParams struct{ SlowPeriod, WarmPeriod int }

type hungryStrategy struct{ p hungryParams }

func (hungryStrategy) Ticker() string  { return "TEST" }
func (s hungryStrategy) Lookback() int { return s.p.SlowPeriod + s.p.WarmPeriod }
func (hungryStrategy) Decide(md strategy.MarketData) model.Signal {
	if md.Position == nil {
		return model.Signal{Kind: model.SignalBuy}
	}
	return model.Signal{Kind: model.SignalSell, Reason: "TP"}
}

func hungryBinding() Binding {
	return Binding{
		DefaultParams: func() any { return hungryParams{SlowPeriod: 10, WarmPeriod: 5} },
		Build:         func(p any) strategy.Strategy { return hungryStrategy{p: p.(hungryParams)} },
		ParseParams:   func([]byte) (any, error) { return hungryParams{}, nil },
	}
}

func TestMaxGridLookbackAccumulatesAcrossFieldsAndPhases(t *testing.T) {
	phases := []Phase{
		{Grid: Grid{"SlowPeriod": {10, 200, 40}}},
		{Grid: Grid{"WarmPeriod": {5, 60}}},
	}
	got, err := maxGridLookback(hungryBinding(), phases)
	if err != nil {
		t.Fatalf("maxGridLookback: %v", err)
	}
	// 200+60. Not 15 (defaults only) and not 205 (best single override over the defaults).
	if got != 260 {
		t.Fatalf("maxGridLookback = %d, want 260", got)
	}
}

func TestMaxGridLookbackRejectsUnknownField(t *testing.T) {
	if _, err := maxGridLookback(hungryBinding(), []Phase{{Grid: Grid{"Nope": {1}}}}); err == nil {
		t.Fatal("want an error for a grid field absent from Params")
	}
}

// TestRunWalkForwardGuardsSweptLookback: the train window comfortably feeds the defaults'
// lookback but not the hungriest grid combination. Sizing the guard from the defaults let
// that pass, and every under-fed combo then scored zero trades with no error anywhere.
func TestRunWalkForwardGuardsSweptLookback(t *testing.T) {
	from, to := date(2025, time.January, 1), date(2025, time.July, 1) // 3+3 = one fold
	candles := genHourly(from, to)
	cfg := backtest.Config{InitialCash: 100000, Fraction: 1, Commission: 0.0005, Lot: 1}

	// A 3-month hourly train slice is ~2160 bars: ample for the defaults' 15, impossible for 100000.
	phases := []Phase{{Grid: Grid{"SlowPeriod": {10, 100000}}}}
	_, err := RunWalkForward(hungryBinding(), phases, candles, nil, nil, cfg,
		"profit_factor", 0, from, to, 3, 3)
	if err == nil {
		t.Fatal("want an error: the grid sweeps a lookback the train window cannot feed")
	}
	if !strings.Contains(err.Error(), "lookback") {
		t.Fatalf("error should name the lookback, got: %v", err)
	}
}

func TestRunWalkForwardNoFold(t *testing.T) {
	from, to := date(2025, time.January, 1), date(2025, time.April, 1) // 3 months
	candles := genHourly(from, to)
	cfg := backtest.Config{InitialCash: 100000, Fraction: 1, Commission: 0.0005, Lot: 1}
	_, err := RunWalkForward(fakeBinding(), []Phase{{Grid: Grid{"Threshold": {1}}}}, candles, nil, nil, cfg,
		"profit_factor", 0, from, to, 3, 3)
	if err == nil {
		t.Fatal("want error when no fold fits")
	}
}

func TestRenderWalkForwardMarkdown(t *testing.T) {
	s := WalkForwardSummary{
		Folds: []WalkForwardFold{
			{
				Index:     1,
				TrainFrom: date(2025, time.January, 1), TrainTo: date(2025, time.April, 1),
				TestFrom: date(2025, time.April, 1), TestTo: date(2025, time.July, 1),
				InSamplePF: 2.10, OOS: backtest.Metrics{ProfitFactor: 1.30, TotalTrades: 12},
				OOSNetPnLPct: 0.031, OOSMaxDDPct: 0.04, OOSTrades: 12,
				WinnerRows: []backtest.ParamLine{{Name: "RSIPeriod", Value: "6"}, {Name: "StopATRMult", Value: "0.8"}},
			},
			{
				Index:     2,
				TrainFrom: date(2025, time.April, 1), TrainTo: date(2025, time.July, 1),
				TestFrom: date(2025, time.July, 1), TestTo: date(2025, time.October, 1),
				InSamplePF: 1.90, OOS: backtest.Metrics{ProfitFactor: 0.80, TotalTrades: 9},
				OOSNetPnLPct: -0.012, OOSMaxDDPct: 0.06, OOSTrades: 9,
				WinnerRows: []backtest.ParamLine{{Name: "RSIPeriod", Value: "6"}, {Name: "StopATRMult", Value: "1.2"}},
			},
		},
		PooledOOS:           backtest.Metrics{ProfitFactor: 1.05, TotalTrades: 21, WinRate: 0.48},
		CompoundedReturnPct: 0.0186,
	}
	md := RenderWalkForwardMarkdown("NVTK", "profit_factor", s, 3, 3)

	for _, want := range []string{
		"# Walk-forward NVTK",
		"## Пул сделок (агрегат OOS)",
		"Profit factor",
		"Compounded return",
		"## Результаты по фолдам",
		"## Стабильность параметров",
		"RSIPeriod",   // stable param mentioned
		"StopATRMult", // varied param mentioned
	} {
		if !strings.Contains(md, want) {
			t.Errorf("report missing %q", want)
		}
	}
}

// regimeStrategy держит позицию ровно один бар: Phase=0 входит на чётных часах и забирает
// приращение цены до нечётного бара, Phase=1 — наоборот. Обе комбинации торгуют на всей
// ленте (это важно: иначе «чужая» комбинация даёт 0 OOS-сделок и фолд помечается
// пропущенным вместо того, чтобы дать сравнимые метрики), различаются они только тем, чьи
// приращения им достаются.
type regimeParams struct{ Phase int }

type regimeStrategy struct{ p regimeParams }

func (regimeStrategy) Ticker() string { return "TEST" }
func (regimeStrategy) Lookback() int  { return 1 }
func (s regimeStrategy) Decide(md strategy.MarketData) model.Signal {
	if md.Position != nil {
		return model.Signal{Kind: model.SignalSell, Reason: "TP"}
	}
	// Бары часовые и начинаются с полуночи, поэтому чётность часа = чётность индекса бара.
	if md.Times[len(md.Times)-1].Hour()%2 == s.p.Phase {
		return model.Signal{Kind: model.SignalBuy}
	}
	return model.Signal{Kind: model.SignalNone}
}

func regimeBinding() Binding {
	return Binding{
		DefaultParams: func() any { return regimeParams{} },
		Build:         func(p any) strategy.Strategy { return regimeStrategy{p: p.(regimeParams)} },
		ParseParams:   func([]byte) (any, error) { return regimeParams{}, nil },
	}
}

// regimeCandles задаёт приращения цены циклом по 4 бара, разным до и после split:
//
//	i%4:       1     2     3     4          чей это бар
//	до split: +2    +1    −1    −2          нечёт → Phase=0, чёт → Phase=1
//	после:    +1    +4    −4    −1
//
// Отсюда PF по половинам: до split — Phase0 = 2.0, Phase1 = 0.5; после split — Phase0 =
// 0.25, Phase1 = 4.0. Сумма приращений внутри каждого цикла равна нулю в обеих половинах,
// поэтому цена не дрейфует и размер позиции при Fraction=1 остаётся постоянным — PF-ы
// половин складываются без перекоса. На объединённом окне (половина до + половина после)
// Phase1 выигрывает: 5/3 против 3/5 у Phase0. Именно эта смена победителя и ловит утечку.
func regimeCandles(from, split, to time.Time) []backtest.Candle {
	before := [4]float64{-2, +2, +1, -1} // индексом служит i%4, поэтому 0-й элемент — это i%4==0
	after := [4]float64{-1, +1, +4, -4}
	var out []backtest.Candle
	price := 100.0
	for ts, i := from, 0; ts.Before(to); ts, i = ts.Add(time.Hour), i+1 {
		if i > 0 {
			if ts.Before(split) {
				price += before[i%4]
			} else {
				price += after[i%4]
			}
		}
		out = append(out, backtest.Candle{Time: ts, Open: price, High: price + 1, Low: price - 1, Close: price, Volume: 1})
	}
	return out
}

// Калибровка фолда обязана видеть ТОЛЬКО train-окно. Подмена trainTo на testTo (или любой
// другой способ дотянуть обучение до тестового периода) делает walk-forward бессмысленным:
// именно этой процедурой в проекте принимается решение о выводе стратегии в прод, и её
// планка (pooled OOS PF) перестаёт что-либо значить, если параметры подобраны по тем же
// данным, на которых потом измеряются.
func TestRunWalkForwardCalibratesOnTrainWindowOnly(t *testing.T) {
	from, to := date(2025, time.January, 1), date(2025, time.October, 1)
	split := date(2025, time.April, 1) // = testFrom первого фолда
	candles := regimeCandles(from, split, to)
	phases := []Phase{{Grid: Grid{"Phase": {0, 1}}}}
	// Commission=0: исход должна решать механика ленты, а не комиссия, которая при
	// Fraction=1 съедает обе комбинации до PF<1 и смазывает разницу между ними.
	cfg := backtest.Config{InitialCash: 100000, Fraction: 1, Lot: 1}

	s, err := RunWalkForward(regimeBinding(), phases, candles, nil, nil, cfg,
		"profit_factor", 0, from, to, 3, 3)
	if err != nil {
		t.Fatalf("RunWalkForward: %v", err)
	}
	// Фолд 1 (train до split, test после) — собственно детектор утечки. Фолд 2 (обучение и
	// тест целиком после split) обязан выбрать другого победителя: это отсекает реализацию,
	// которая всегда возвращает первую комбинацию сетки.
	for _, want := range []struct {
		fold  int
		phase int
	}{{1, 0}, {2, 1}} {
		f := s.Folds[want.fold-1]
		if f.Note != "" {
			t.Fatalf("фолд %d пропущен: %s", want.fold, f.Note)
		}
		got, ok := f.WinnerParams.(regimeParams)
		if !ok {
			t.Fatalf("фолд %d: WinnerParams = %T, want regimeParams", want.fold, f.WinnerParams)
		}
		if got.Phase != want.phase {
			t.Fatalf("победитель фолда %d = Phase %d, want %d: на train-окне (%s..%s) выигрывает "+
				"Phase %d, а Phase %d — только если дотянуть обучение до тестового периода",
				want.fold, got.Phase, want.phase, f.TrainFrom.Format("2006-01-02"),
				f.TrainTo.Format("2006-01-02"), want.phase, got.Phase)
		}
	}
}

// В OOS-метрики фолда обязаны попасть только сделки, ОТКРЫТЫЕ внутри тестового окна.
// Прогревочный прогон стартует с trainFrom (индикаторам нужна история), поэтому без фильтра
// в «out-of-sample» попадают сделки самого train-периода — то есть тех данных, на которых
// параметры и подбирались.
func TestRunWalkForwardCountsOnlyTradesOpenedInTheTestWindow(t *testing.T) {
	from, to := date(2025, time.January, 1), date(2025, time.October, 1)
	candles := genHourly(from, to)
	phases := []Phase{{Grid: Grid{"Threshold": {1, 2}}}}
	cfg := backtest.Config{InitialCash: 100000, Fraction: 1, Commission: 0.0005, Lot: 1}

	s, err := RunWalkForward(fakeBinding(), phases, candles, nil, nil, cfg,
		"profit_factor", 0, from, to, 3, 3)
	if err != nil {
		t.Fatalf("RunWalkForward: %v", err)
	}
	for _, f := range s.Folds {
		// Повторяем прогревочный прогон фолда независимо и считаем эталон сами.
		warm := sliceByRange(candles, f.TrainFrom, f.TestTo)
		res := backtest.Run(fakeBinding().Build(f.WinnerParams), warm, nil, nil, cfg)
		want := len(tradesEnteredFrom(res.Trades, f.TestFrom))
		if len(res.Trades) <= want {
			t.Fatalf("фолд %d: прогревочный прогон дал %d сделок, из них в тестовом окне %d — "+
				"фикстура не содержит ни одной train-сделки, тест ничего не проверяет",
				f.Index, len(res.Trades), want)
		}
		if f.OOSTrades != want {
			t.Fatalf("фолд %d: OOSTrades = %d, want %d (сделки, открытые с %s); в метрики попали "+
				"сделки прогревочного train-периода",
				f.Index, f.OOSTrades, want, f.TestFrom.Format("2006-01-02"))
		}
	}
}
