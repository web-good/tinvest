package live

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"tinvest/internal/config"
	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/rsi_pullback/live/dto"
	rsizonelive "tinvest/internal/service/trading_strategy/rsi_zone/live"
	"tinvest/internal/service/trading_strategy/scalping/model"
)

func boostOn(c *config.RSIPullbackConfig) {
	c.LagBoostEnabled, c.LagBoostMult, c.LagBoostPct, c.LagBoostDays = true, 2, 5, 5
}

// closesDailies — завершённые дневки по закрытиям (oldest-first), последняя — вчера от end.
func closesDailies(end time.Time, closes ...float64) []*imodel.CandleItemTechAnalyse {
	day := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, msk)
	n := len(closes)
	out := make([]*imodel.CandleItemTechAnalyse, 0, n)
	for i, c := range closes {
		out = append(out, &imodel.CandleItemTechAnalyse{
			Time: day.AddDate(0, 0, -(n - i)),
			Open: q(int64(c)), High: q(int64(c)), Low: q(int64(c)), Close: q(int64(c)),
			Volume: 100000, IsComplete: true,
		})
	}
	return out
}

func isUID(uid string) any {
	return mock.MatchedBy(func(p *string) bool { return p != nil && *p == uid })
}

// Тикер-фикстура: 60 дневок по 100, последняя — падение до 90 (−10%); индекс ровный.
func laggingTicker(now time.Time) []*imodel.CandleItemTechAnalyse {
	c := make([]float64, 60)
	for i := range c {
		c[i] = 100
	}
	c[59] = 90
	return closesDailies(now, c...)
}

func flatIndex(now time.Time) []*imodel.CandleItemTechAnalyse {
	c := make([]float64, 20)
	for i := range c {
		c[i] = 2000
	}
	return closesDailies(now, c...)
}

// expectTickerCandles — свечи тикера только по его UID: запрос индекса идёт тем же
// CandleClient, и общий mock.Anything по UID отдал бы индексу дневки тикера.
func (e *env) expectTickerCandles(ticker string, m30, day []*imodel.CandleItemTechAnalyse) {
	e.market.EXPECT().GetCandles(mock.Anything, isUID("uid-"+ticker), mock.MatchedBy(isM30),
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(m30, nil)
	e.market.EXPECT().GetCandles(mock.Anything, isUID("uid-"+ticker), mock.MatchedBy(isDay1),
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(day, nil)
}

func runPass(t *testing.T, e *env) []string {
	t.Helper()
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil).Maybe()
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil).Maybe()
	var sent []string
	e.tg.EXPECT().SendMessage(mock.Anything).RunAndReturn(func(s string) error {
		sent = append(sent, s)
		return nil
	}).Maybe()
	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	return sent
}

// Выключено — ни UID индекса, ни свечей IMOEX: моки без ожиданий упадут на любом вызове.
func TestLagBoostOffDoesNotTouchIndex(t *testing.T) {
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, nil)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectTickerCandles("GAZP", pullback30m(sharedLastBar, 400), laggingTicker(sharedNow))
	runPass(t, e)
	if got := e.state(t)["GAZP"].Quantity; got != 500 {
		t.Fatalf("Quantity = %d, want 500 (5%% без буста)", got)
	}
}

// Отставший тикер — доля ×2, в уведомлении пометка.
func TestLagBoostDoublesLaggingTicker(t *testing.T) {
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, boostOn)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.instruments.EXPECT().IndicativeUIDByTicker(mock.Anything, "IMOEX").Return("uid-imoex", nil).Once()
	e.expectTickerCandles("GAZP", pullback30m(sharedLastBar, 400), laggingTicker(sharedNow))
	e.market.EXPECT().GetCandles(mock.Anything, isUID("uid-imoex"), mock.MatchedBy(isDay1),
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(flatIndex(sharedNow), nil).Once()
	sent := runPass(t, e)
	if got := e.state(t)["GAZP"].Quantity; got != 1000 {
		t.Fatalf("Quantity = %d, want 1000 (5%%·2)", got)
	}
	if !anyContains(sent, "Размер ×2: отстал от IMOEX на 10.0% за 5д") {
		t.Fatalf("нет пометки буста, отправлено: %q", sent)
	}
	if anyContains(sent, "(урезано кэшем)") {
		t.Fatalf("пометка урезания при достаточном кэше: %q", sent)
	}
}

// Индекс недоступен — вход обычным размером, вход состоялся.
func TestLagBoostIndexFailureKeepsBaseSize(t *testing.T) {
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, boostOn)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.instruments.EXPECT().IndicativeUIDByTicker(mock.Anything, "IMOEX").Return("", errors.New("down")).Once()
	e.expectTickerCandles("GAZP", pullback30m(sharedLastBar, 400), laggingTicker(sharedNow))
	runPass(t, e)
	if got := e.state(t)["GAZP"].Quantity; got != 500 {
		t.Fatalf("Quantity = %d, want 500", got)
	}
}

// Два входа в одном пассе при упавших свечах индекса — один запрос, оба ×1.
func TestLagBoostIndexFetchedOncePerPass(t *testing.T) {
	e := newSharedEnv(t, sharedNow, []string{"GAZP", "T"}, boostOn)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP", "T"), nil)
	e.instruments.EXPECT().IndicativeUIDByTicker(mock.Anything, "IMOEX").Return("uid-imoex", nil).Once()
	e.expectTickerCandles("GAZP", pullback30m(sharedLastBar, 400), laggingTicker(sharedNow))
	e.expectTickerCandles("T", pullback30m(sharedLastBar, 400), laggingTicker(sharedNow))
	e.market.EXPECT().GetCandles(mock.Anything, isUID("uid-imoex"), mock.MatchedBy(isDay1),
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("down")).Once()
	runPass(t, e)
	st := e.state(t)
	if st["GAZP"].Quantity != 500 || st["T"].Quantity != 500 {
		t.Fatalf("Quantity = %d/%d, want 500/500", st["GAZP"].Quantity, st["T"].Quantity)
	}
}

// Кэша хватает только на базовую долю — лоты урезаны, уведомление это говорит.
func TestLagBoostCappedByCashIsReported(t *testing.T) {
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, boostOn)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.instruments.EXPECT().IndicativeUIDByTicker(mock.Anything, "IMOEX").Return("uid-imoex", nil)
	e.expectTickerCandles("GAZP", pullback30m(sharedLastBar, 400), laggingTicker(sharedNow))
	e.market.EXPECT().GetCandles(mock.Anything, isUID("uid-imoex"), mock.MatchedBy(isDay1),
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(flatIndex(sharedNow), nil)
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(60_000.0, nil) // 600 штук по 100
	var sent []string
	e.tg.EXPECT().SendMessage(mock.Anything).RunAndReturn(func(s string) error {
		sent = append(sent, s)
		return nil
	}).Maybe()
	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := e.state(t)["GAZP"].Quantity; got != 600 {
		t.Fatalf("Quantity = %d, want 600 (урезано кэшем)", got)
	}
	if !anyContains(sent, "(урезано кэшем)") {
		t.Fatalf("нет пометки урезания, отправлено: %q", sent)
	}
}

// gap_fade буст не получает, даже если тикер отстал.
func TestLagBoostSkipsGapFade(t *testing.T) {
	gap := zoneFake(model.SignalBuy, "GAZP")
	gap.name = "gap_fade"
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, boostOn, gap)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectTickerCandles("GAZP", flatAt30m(sharedLastBar, 400, 100), laggingTicker(sharedNow))
	runPass(t, e)
	if got := e.state(t)["GAZP"]; got.Strategy != "gap_fade" || got.Quantity != 500 {
		t.Fatalf("запись = %+v, want gap_fade и 500 (без буста; BuyPct фейка 5)", got)
	}
}

// Неверный конфиг — опция выключена (индекс не запрашивается), раннер работает.
func TestLagBoostInvalidConfigDisables(t *testing.T) {
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, func(c *config.RSIPullbackConfig) {
		boostOn(c)
		c.LagBoostMult = 20
	})
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectTickerCandles("GAZP", pullback30m(sharedLastBar, 400), laggingTicker(sharedNow))
	runPass(t, e)
	if got := e.state(t)["GAZP"].Quantity; got != 500 {
		t.Fatalf("Quantity = %d, want 500", got)
	}
}

// lagBoostStrategies держит имя rsi_zone строкой; при переименовании гостя буст молча
// отвалился бы — этот тест ловит расхождение.
func TestLagBoostStrategiesMatchesZoneName(t *testing.T) {
	if !lagBoostStrategies[rsizonelive.Name] {
		t.Fatalf("lagBoostStrategies не содержит %q", rsizonelive.Name)
	}
}

func anyContains(msgs []string, sub string) bool {
	for _, m := range msgs {
		if strings.Contains(m, sub) {
			return true
		}
	}
	return false
}
