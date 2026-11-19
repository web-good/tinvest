package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"tinvest/internal/service/screener/bybitdensity/model"

	"golang.org/x/sync/errgroup"
)

var errFeedVerified = errors.New("public feed messages verified")

func checkFeed(ctx context.Context, market marketData, op options, out io.Writer) error {
	ctx, cancel := context.WithTimeout(ctx, op.checkTimeout)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)
	for _, category := range []model.Market{model.Spot, model.Linear} {
		group.Go(func() error { return checkMarket(ctx, market, category, op.depth) })
	}
	if err := group.Wait(); err != nil {
		return err
	}
	_, err := fmt.Fprintln(out, "BTCUSDT spot and linear: metadata, snapshot, public trades and closed M1 verified")
	return err
}

func checkMarket(ctx context.Context, market marketData, category model.Market, depth int) error {
	instruments, err := market.Instruments(ctx, category, []string{"BTCUSDT"}, 1)
	if err != nil {
		return fmt.Errorf("%s BTCUSDT metadata: %w", category, err)
	}
	if len(instruments) != 1 {
		return fmt.Errorf("%s BTCUSDT metadata missing", category)
	}
	var book, trade, candle bool
	err = market.Stream(ctx, instruments, depth, func(event model.Event) error {
		if event.Reset {
			book, trade, candle = false, false, false
		}
		book = book || (event.Book != nil && event.Book.Snapshot)
		trade = trade || len(event.Trades) > 0
		candle = candle || event.Candle != nil
		if book && trade && candle {
			return errFeedVerified
		}
		return nil
	})
	if errors.Is(err, errFeedVerified) {
		return nil
	}
	if err == nil {
		err = ctx.Err()
	}
	if err == nil {
		err = errors.New("unexpected end of feed")
	}
	return fmt.Errorf("%s BTCUSDT feed check (snapshot=%t trade=%t candle=%t): %w", category, book, trade, candle, err)
}
