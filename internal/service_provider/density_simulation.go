package service_provider

import (
	"context"
	"fmt"
	"strings"
	"tinvest/internal/config"
	"tinvest/internal/service/screener/density"
	densitylive "tinvest/internal/service/screener/density/live"
	"tinvest/internal/service/screener/density/paper"
	"tinvest/internal/service/screener/density/simulation"
	marketgrpc "tinvest/pkg/client/grpc"
	"tinvest/pkg/logger"
)

func (s *ServiceProvider) densitySimulationParticipant(c *config.DensityLiveConfig, d *config.DensityConfig, detector density.Config, lc densitylive.Config) *densitylive.Participant {
	p := &densitylive.Participant{FollowScreener: c.FollowScreener, AllowEntry: s.densityEntryFilter(c.Tickers), StateDir: c.StateDir, Tickers: append([]string(nil), c.Tickers...), Config: lc, Detector: detector, HistoryInterval: d.HistoryInterval}
	p.RunPaper = func(ctx context.Context) error {
		if err := densitylive.CheckSimulationState(c.StateDir); err != nil {
			return err
		}
		if s.appConfig.GrpcClient == nil || s.appConfig.GrpcClient.TokenProd == "" {
			return fmt.Errorf("T_BANK market data token is required for density simulation")
		}
		client, err := s.GetGrpcClient()
		if err != nil {
			return err
		}
		bot, err := s.GetTelegramBot()
		if err != nil {
			return err
		}
		cfg := paper.DefaultConfig()
		cfg.RestrictEntryHours = true
		cfg.IgnoreCapital = true
		cfg.Commission = 0
		cfg.Latency = 0
		cfg.MaxEntryLots = 1
		cfg.MaxPositions = len(c.Tickers)
		cfg.BookExecution = c.FollowScreener
		if c.FollowScreener {
			cfg.MaxPositions = detector.MaxBooks
		}
		cfg.EntryTicks = lc.EntryTicks
		cfg.EntryPct = lc.EntryPct
		cfg.StopTicks = lc.StopTicks
		cfg.RewardRisk = lc.RewardRisk
		cfg.MinRemaining = lc.MinRemaining
		cfg.Freshness = lc.Freshness
		cfg.EntryTTL = lc.EntryTTL
		cfg.MaxHold = 0 // Alert simulation holds positions until a strategy exit.
		cfg.Cooldown = lc.Cooldown
		tickers := make([]string, len(c.Tickers))
		for n, t := range c.Tickers {
			tickers[n] = strings.ToUpper(strings.TrimSpace(t))
		}
		service := simulation.Service{FollowScreener: c.FollowScreener, AllowEntry: p.AllowEntry, Market: marketgrpc.NewDensityMarketData(client.Connection(), s.appConfig.GrpcClient.TokenProd), Config: cfg, Detector: detector, Tickers: tickers, Root: c.StateDir, HistoryInterval: d.HistoryInterval, Notify: (densityLiveSender{bot: bot.API(), chatID: d.ChatID, topicID: d.TopicID}).SendText, Warn: func(err error) { logger.ErrorContext(context.Background(), "density simulation: "+err.Error()) }}
		logger.Info("density simulation started: real orders disabled; alerts only on virtual fills")
		return service.Run(ctx)
	}
	return p
}
