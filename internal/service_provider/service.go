package service_provider

import (
	"tinvest/internal/service/instrument/atr"
	"tinvest/internal/service/instrument/ema"
	"tinvest/internal/service/instrument/macd"
	"tinvest/internal/service/instrument/rsi"
	"tinvest/internal/service/instrument/volatility"
	"tinvest/internal/service/news"
	"tinvest/internal/service/notification/purchase_shares"
	"tinvest/internal/service/portfolio/analyze"
	"tinvest/internal/service/portfolio/yield"
	"tinvest/internal/service/screener/dividend"
	"tinvest/internal/service/telegram_commands"
	"tinvest/internal/service/trading_strategy/bonds"
	"tinvest/internal/service/trading_strategy/golden_x"
	"tinvest/internal/service/trading_strategy/reversion/live"
	rsipullbacklive "tinvest/internal/service/trading_strategy/rsi_pullback/live"
	"tinvest/pkg/client/rss"
	"tinvest/pkg/client/telegram"
)

type service struct {
	purchaseSharesService  purchase_shares.PurchaseShares
	reversionLiveService   live.Service
	rsiPullbackLiveService rsipullbacklive.Service
	goldenXTradingService  golden_x.GoldenX
	bondsTradingService    bonds.Bonds
	emaInstrument          ema.Instrument
	atrInstrument          atr.Instrument
	rsiInstrument          rsi.Instrument
	MACDInstrument         macd.Instrument
	volatilityInstrument   volatility.Instrument
	analyze                analyze.Analyze
	portfolioYield         yield.Yield
	telegramCommands       *telegram_commands.Listener
	newsService            *news.Service
	dividend               *dividendSingleton
}

type dividendSingleton struct {
	screener dividend.Screener
	provider dividend.RankProvider
}

func (s *ServiceProvider) dividendSvc() *dividendSingleton {
	if serviceProvider.service.dividend == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		svc := dividend.NewService(grpcClient.InstrumentsServiceClient())
		serviceProvider.service.dividend = &dividendSingleton{screener: svc, provider: svc}
	}
	return serviceProvider.service.dividend
}

func (s *ServiceProvider) GetDividendScreener() dividend.Screener {
	return s.dividendSvc().screener
}

func (*ServiceProvider) GetPurchaseSharesService() purchase_shares.PurchaseShares {
	if serviceProvider.service.purchaseSharesService == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		serviceProvider.service.purchaseSharesService = purchase_shares.NewService(grpcClient.InstrumentsServiceClient(), grpcClient.MarketDataServiceClient())
	}

	return serviceProvider.service.purchaseSharesService
}

func (*ServiceProvider) GetBondsTradingService() bonds.Bonds {
	if serviceProvider.service.bondsTradingService == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		serviceProvider.service.bondsTradingService = bonds.NewService(
			serviceProvider.RSI(),
			grpcClient.InstrumentsServiceClient(),
			grpcClient.MarketDataServiceClient(),
		)
	}

	return serviceProvider.service.bondsTradingService
}

func (s *ServiceProvider) GetGoldenXTradingService() golden_x.GoldenX {
	if serviceProvider.service.goldenXTradingService == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		tgClient, _ := serviceProvider.GetGoldenXSender()
		serviceProvider.service.goldenXTradingService = golden_x.NewService(
			grpcClient.MarketDataServiceClient(),
			tgClient,
			golden_x.WithRankProvider(s.dividendSvc().provider),
		)
	}

	return serviceProvider.service.goldenXTradingService
}

func (*ServiceProvider) Ema() ema.Instrument {
	if serviceProvider.service.emaInstrument == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		serviceProvider.service.emaInstrument = ema.NewEma(
			grpcClient.MarketDataServiceClient(),
		)
	}

	return serviceProvider.service.emaInstrument
}

func (*ServiceProvider) Atr() atr.Instrument {
	if serviceProvider.service.atrInstrument == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		serviceProvider.service.atrInstrument = atr.NewAtr(
			grpcClient.MarketDataServiceClient(),
		)
	}

	return serviceProvider.service.atrInstrument
}

func (*ServiceProvider) MACD() macd.Instrument {
	if serviceProvider.service.MACDInstrument == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		serviceProvider.service.MACDInstrument = macd.New(
			grpcClient.MarketDataServiceClient(),
		)
	}

	return serviceProvider.service.MACDInstrument
}

func (*ServiceProvider) RSI() rsi.Instrument {
	if serviceProvider.service.rsiInstrument == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		serviceProvider.service.rsiInstrument = rsi.New(
			grpcClient.MarketDataServiceClient(),
		)
	}

	return serviceProvider.service.rsiInstrument
}

func (*ServiceProvider) Volatility() volatility.Instrument {
	if serviceProvider.service.volatilityInstrument == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		serviceProvider.service.volatilityInstrument = volatility.New(
			grpcClient.MarketDataServiceClient(),
		)
	}

	return serviceProvider.service.volatilityInstrument
}

func (*ServiceProvider) GetAnalyze() analyze.Analyze {
	if serviceProvider.service.analyze == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		serviceProvider.service.analyze = analyze.NewService(
			grpcClient.OperationsServiceClient(),
			grpcClient.UserServiceClient(),
			grpcClient.InstrumentsServiceClient(),
		)
	}

	return serviceProvider.service.analyze
}

func (*ServiceProvider) GetPortfolioYield() yield.Yield {
	if serviceProvider.service.portfolioYield == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		serviceProvider.service.portfolioYield = yield.NewService(
			grpcClient.OperationsServiceClient(),
			grpcClient.UserServiceClient(),
			serviceProvider.appConfig.PortfolioYield,
		)
	}

	return serviceProvider.service.portfolioYield
}

func (*ServiceProvider) GetReversionLiveService() live.Service {
	if serviceProvider.service.reversionLiveService == nil {
		grpcClient, _ := serviceProvider.GetReversionGrpcClient()
		tgClient, _ := serviceProvider.GetReversionSender()
		serviceProvider.service.reversionLiveService = live.NewService(
			grpcClient.InstrumentsServiceClient(),
			grpcClient.MarketDataServiceClient(),
			grpcClient.OperationsServiceClient(),
			grpcClient.OrdersServiceClient(),
			grpcClient.StopOrdersServiceClient(),
			tgClient,
			serviceProvider.appConfig.Reversion,
		)
	}

	return serviceProvider.service.reversionLiveService
}

// GetRSIPullbackLiveService wires the live rsi_pullback runner onto its own account: its own
// gRPC client (RSI_PULLBACK_TOKEN) and its own Telegram topic. Sharing either with reversion
// would let one strategy see the other's position in the portfolio and manage it as its own —
// the two overlap on tickers.
func (*ServiceProvider) GetRSIPullbackLiveService() rsipullbacklive.Service {
	if serviceProvider.service.rsiPullbackLiveService == nil {
		grpcClient, _ := serviceProvider.GetRSIPullbackGrpcClient()
		tgClient, _ := serviceProvider.GetRSIPullbackSender()
		serviceProvider.service.rsiPullbackLiveService = rsipullbacklive.NewService(
			grpcClient.InstrumentsServiceClient(),
			grpcClient.MarketDataServiceClient(),
			grpcClient.OperationsServiceClient(),
			grpcClient.OrdersServiceClient(),
			grpcClient.StopOrdersServiceClient(),
			tgClient,
			serviceProvider.appConfig.RSIPullback,
		)
	}

	return serviceProvider.service.rsiPullbackLiveService
}

func (s *ServiceProvider) GetTelegramCommands() (*telegram_commands.Listener, error) {
	if serviceProvider.service.telegramCommands != nil {
		return serviceProvider.service.telegramCommands, nil
	}
	bot, err := s.GetTelegramBot()
	if err != nil {
		return nil, err
	}
	factory := func(chatID int64, threadID int) telegram.Client {
		return telegram.NewTopicSender(bot, chatID, threadID)
	}
	cmds := telegram_commands.New(
		s.GetAnalyze(),
		s.GetPortfolioYield(),
		s.GetBondsTradingService(),
		s.GetDividendScreener(),
		factory,
		s.appConfig.TelegramClient.AllowedUserIDs,
	)
	serviceProvider.service.telegramCommands = telegram_commands.NewListener(bot, cmds)

	return serviceProvider.service.telegramCommands, nil
}

func (s *ServiceProvider) GetNewsService() (*news.Service, error) {
	if serviceProvider.service.newsService != nil {
		return serviceProvider.service.newsService, nil
	}
	sender, err := s.GetNewsSender()
	if err != nil {
		return nil, err
	}
	serviceProvider.service.newsService = news.NewService(
		rss.NewClient(s.appConfig.News.FeedURL),
		sender,
	)

	return serviceProvider.service.newsService, nil
}
