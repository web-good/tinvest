package grpc

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
	"tinvest/internal/converter"
	"tinvest/internal/model"
	investapi "tinvest/internal/pb/v1"
)

type MarketDataServiceClient interface {
	GetTechAnalyseMacD(context context.Context, instrumentUid string, interval int, from *timestamppb.Timestamp, to *timestamppb.Timestamp, fastLength int32) ([]*model.MacDItemTechAnalyse, error)
	GetTechAnalyseRsi(context context.Context, instrumentUid string, interval int, from *timestamppb.Timestamp, to *timestamppb.Timestamp) ([]*model.RsiItemTechAnalyse, error)
	GetTechAnalyseEma(context context.Context, instrumentUid string, interval int, from *timestamppb.Timestamp, to *timestamppb.Timestamp, length int32) ([]*model.EmaItemTechAnalyse, error)
	GetTechAnalyseBB(context context.Context, instrumentUid string, interval int, from *timestamppb.Timestamp, to *timestamppb.Timestamp) ([]*model.BbItemTechAnalyse, error)
	GetCandles(context context.Context, instrumentUid *string, interval int32, from *timestamp.Timestamp, to *timestamp.Timestamp, limit *int32, withWeekend bool) ([]*model.CandleItemTechAnalyse, error)
}

type marketDataService struct {
	marketDataApi investapi.MarketDataServiceClient
	auth          *Auth
}

func (m *marketDataService) GetTechAnalyseBB(ctx context.Context, instrumentUid string, interval int, from *timestamppb.Timestamp, to *timestamppb.Timestamp) ([]*model.BbItemTechAnalyse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := m.marketDataApi.GetTechAnalysis(ctx, &investapi.GetTechAnalysisRequest{
		Length:        20,
		IndicatorType: investapi.GetTechAnalysisRequest_INDICATOR_TYPE_BB,
		InstrumentUid: instrumentUid,
		From:          from,
		To:            to,
		Interval:      investapi.GetTechAnalysisRequest_IndicatorInterval(interval),
		TypeOfPrice:   investapi.GetTechAnalysisRequest_TYPE_OF_PRICE_CLOSE,
		Deviation: &investapi.GetTechAnalysisRequest_Deviation{
			DeviationMultiplier: &investapi.Quotation{Units: 2},
		},
	}, NewRPCCredential(m.auth))
	fmt.Println(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to request TechAnalysis: %w", err)
	}

	return converter.ConvertBbTechAnalysisFromPb(resp.GetTechnicalIndicators()), nil
}

func (m *marketDataService) GetTechAnalyseEma(ctx context.Context, instrumentUid string, interval int, from *timestamppb.Timestamp, to *timestamppb.Timestamp, length int32) ([]*model.EmaItemTechAnalyse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := m.marketDataApi.GetTechAnalysis(ctx, &investapi.GetTechAnalysisRequest{
		Length:        length,
		IndicatorType: investapi.GetTechAnalysisRequest_INDICATOR_TYPE_EMA,
		InstrumentUid: instrumentUid,
		From:          from,
		To:            to,
		Interval:      investapi.GetTechAnalysisRequest_IndicatorInterval(interval),
		TypeOfPrice:   investapi.GetTechAnalysisRequest_TYPE_OF_PRICE_CLOSE,
	}, NewRPCCredential(m.auth))

	if err != nil {
		return nil, fmt.Errorf("failed to request TechAnalysis: %w", err)
	}

	return converter.ConvertEmaTechAnalysisFromPb(resp.GetTechnicalIndicators()), nil
}

func (m *marketDataService) GetTechAnalyseRsi(ctx context.Context, instrumentUid string, interval int, from *timestamppb.Timestamp, to *timestamppb.Timestamp) ([]*model.RsiItemTechAnalyse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := m.marketDataApi.GetTechAnalysis(ctx, &investapi.GetTechAnalysisRequest{
		Length:        14,
		IndicatorType: investapi.GetTechAnalysisRequest_INDICATOR_TYPE_RSI,
		InstrumentUid: instrumentUid,
		From:          from,
		To:            to,
		Interval:      investapi.GetTechAnalysisRequest_IndicatorInterval(interval),
		TypeOfPrice:   investapi.GetTechAnalysisRequest_TYPE_OF_PRICE_CLOSE,
	}, NewRPCCredential(m.auth))

	if err != nil {
		return nil, fmt.Errorf("failed to request TechAnalysis: %w", err)
	}

	return converter.ConvertRsiTechAnalysisFromPb(resp.GetTechnicalIndicators()), nil
}

func (m *marketDataService) GetTechAnalyseMacD(ctx context.Context, instrumentUid string, interval int, from *timestamppb.Timestamp, to *timestamppb.Timestamp, fastLength int32) ([]*model.MacDItemTechAnalyse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := m.marketDataApi.GetTechAnalysis(ctx, &investapi.GetTechAnalysisRequest{
		IndicatorType: investapi.GetTechAnalysisRequest_INDICATOR_TYPE_MACD,
		InstrumentUid: instrumentUid,
		From:          from,
		To:            to,
		Interval:      investapi.GetTechAnalysisRequest_IndicatorInterval(interval),
		TypeOfPrice:   investapi.GetTechAnalysisRequest_TYPE_OF_PRICE_CLOSE,
		Smoothing: &investapi.GetTechAnalysisRequest_Smoothing{
			FastLength:      fastLength,
			SlowLength:      26,
			SignalSmoothing: 9,
		},
	}, NewRPCCredential(m.auth))

	if err != nil {
		return nil, fmt.Errorf("failed to request TechAnalysis: %w", err)
	}

	return converter.ConvertMacDTechAnalysisFromPb(resp.GetTechnicalIndicators()), nil
}

func (m *marketDataService) GetCandles(ctx context.Context, instrumentUid *string, interval int32, from *timestamp.Timestamp, to *timestamp.Timestamp, limit *int32, withWeekend bool) ([]*model.CandleItemTechAnalyse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	///	candleSourceType := investapi.GetCandlesRequest_CANDLE_SOURCE_UNSPECIFIED

	if withWeekend == true {
		//		candleSourceType = investapi.GetCandlesRequest_CANDLE_SOURCE_INCLUDE_WEEKEND
	}
	resp, err := m.marketDataApi.GetCandles(ctx, &investapi.GetCandlesRequest{
		From:         from,
		InstrumentId: instrumentUid,
		To:           to,
		Interval:     investapi.CandleInterval_CANDLE_INTERVAL_DAY,
		//CandleSourceType: &candleSourceType,
		Limit: limit,
	}, NewRPCCredential(m.auth))
	fmt.Println("4342342", resp.GetCandles())
	if err != nil {
		return nil, fmt.Errorf("failed to request TechAnalysis: %w", err)
	}

	return converter.ConvertCandlesTechAnalysisFromPb(resp.GetCandles()), nil
}

func NewMarketDataService(conn grpc.ClientConnInterface, token string) MarketDataServiceClient {
	return &marketDataService{
		marketDataApi: investapi.NewMarketDataServiceClient(conn),
		auth:          NewAuth(token),
	}
}
