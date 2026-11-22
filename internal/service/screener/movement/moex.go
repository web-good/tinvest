package movement

import (
	"context"
	"time"
	pb "tinvest/internal/pb/v1"
	client "tinvest/pkg/client/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type moex struct {
	shares pb.InstrumentsServiceClient
	data   pb.MarketDataServiceClient
	auth   grpc.CallOption
}

func NewMOEX(conn grpc.ClientConnInterface, token string, minPct, minRVOL float64, send func(string) error) *Worker {
	return &Worker{market: "Мосбиржа · акции", feed: moex{pb.NewInstrumentsServiceClient(conn), pb.NewMarketDataServiceClient(conn), client.NewRPCCredential(client.NewAuth(token))}, minPct: minPct, minRVOL: minRVOL, delay: 450 * time.Millisecond, send: send}
}

func (m moex) instruments(ctx context.Context) ([]instrument, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	resp, err := m.shares.Shares(ctx, &pb.InstrumentsRequest{InstrumentStatus: pb.InstrumentStatus_INSTRUMENT_STATUS_BASE.Enum()}, m.auth)
	if err != nil {
		return nil, err
	}
	var out []instrument
	for _, s := range resp.GetInstruments() {
		if s.GetClassCode() == "TQBR" && s.GetRealExchange() == pb.RealExchange_REAL_EXCHANGE_MOEX {
			out = append(out, instrument{s.GetTicker(), s.GetUid()})
		}
	}
	return out, nil
}

func price(q *pb.Quotation) float64 { return float64(q.GetUnits()) + float64(q.GetNano())/1e9 }

func (m moex) candles(ctx context.Context, id string, start, end time.Time) ([]bar, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	resp, err := m.data.GetCandles(ctx, &pb.GetCandlesRequest{InstrumentId: &id, From: timestamppb.New(start), To: timestamppb.New(end), Interval: pb.CandleInterval_CANDLE_INTERVAL_5_MIN}, m.auth)
	if status.Code(err) == codes.ResourceExhausted {
		return nil, errRateLimited
	}
	if err != nil {
		return nil, err
	}
	var out []bar
	for _, c := range resp.GetCandles() {
		if c.GetIsComplete() {
			out = append(out, bar{c.GetTime().AsTime(), price(c.GetOpen()), price(c.GetHigh()), price(c.GetLow()), float64(c.GetVolume())})
		}
	}
	return out, nil
}
