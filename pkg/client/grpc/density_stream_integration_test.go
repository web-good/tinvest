package grpc

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"
	"time"
	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/service/screener/density"

	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

type localDensityServer struct {
	investapi.UnimplementedMarketDataStreamServiceServer
	received chan *investapi.MarketDataRequest
	auth     chan string
}

func (s *localDensityServer) MarketDataStream(stream googlegrpc.BidiStreamingServer[investapi.MarketDataRequest, investapi.MarketDataResponse]) error {
	md, _ := metadata.FromIncomingContext(stream.Context())
	values := md.Get("authorization")
	if len(values) > 0 {
		s.auth <- values[0]
	} else {
		s.auth <- ""
	}

	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}
		s.received <- req
		var response *investapi.MarketDataResponse
		switch {
		case req.GetSubscribeOrderBookRequest() != nil:
			response = &investapi.MarketDataResponse{Payload: &investapi.MarketDataResponse_SubscribeOrderBookResponse{SubscribeOrderBookResponse: &investapi.SubscribeOrderBookResponse{OrderBookSubscriptions: []*investapi.OrderBookSubscription{{InstrumentUid: "uid", SubscriptionStatus: investapi.SubscriptionStatus_SUBSCRIPTION_STATUS_SUCCESS, OrderBookType: investapi.OrderBookType_ORDERBOOK_TYPE_EXCHANGE}}}}}
		case req.GetSubscribeInfoRequest() != nil:
			response = &investapi.MarketDataResponse{Payload: &investapi.MarketDataResponse_SubscribeInfoResponse{SubscribeInfoResponse: &investapi.SubscribeInfoResponse{InfoSubscriptions: []*investapi.InfoSubscription{{InstrumentUid: "uid", SubscriptionStatus: investapi.SubscriptionStatus_SUBSCRIPTION_STATUS_SUCCESS}}}}}
		case req.GetSubscribeTradesRequest() != nil:
			response = &investapi.MarketDataResponse{Payload: &investapi.MarketDataResponse_SubscribeTradesResponse{SubscribeTradesResponse: &investapi.SubscribeTradesResponse{TradeSource: investapi.TradeSourceType_TRADE_SOURCE_EXCHANGE, TradeSubscriptions: []*investapi.TradeSubscription{{InstrumentUid: "uid", SubscriptionStatus: investapi.SubscriptionStatus_SUBSCRIPTION_STATUS_INSTRUMENT_NOT_FOUND}}}}}
		}
		if response != nil {
			if err := stream.Send(response); err != nil {
				return err
			}
		}
	}

}
func TestDensityStreamLocalGRPC(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "localhost"}, DNSNames: []string{"localhost"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, BasicConstraintsValid: true, IsCA: true}
	der, err := x509.CreateCertificate(rand.Reader, template, template, pub, priv)
	if err != nil {
		t.Fatal(err)
	}
	cert := tls.Certificate{Certificate: [][]byte{der}, PrivateKey: priv}
	roots := x509.NewCertPool()
	parsed, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	roots.AddCert(parsed)
	listener := bufconn.Listen(1 << 20)
	server := googlegrpc.NewServer(googlegrpc.Creds(credentials.NewTLS(&tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12})))
	fake := &localDensityServer{received: make(chan *investapi.MarketDataRequest, 3), auth: make(chan string, 1)}
	investapi.RegisterMarketDataStreamServiceServer(server, fake)
	go func() { _ = server.Serve(listener) }()
	defer server.Stop()
	conn, err := googlegrpc.NewClient("passthrough:///localhost", googlegrpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), googlegrpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{RootCAs: roots, ServerName: "localhost", MinVersion: tls.VersionTLS12})))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	market := NewDensityMarketData(conn, "test-token")
	stream, err := market.OpenStream(ctx, map[string]int64{"uid": 10})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	if err = stream.SubscribeBooks([]string{"uid"}, 50, true); err != nil {
		t.Fatal(err)
	}
	events, err := stream.Receive()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Kind != density.EventBookAck || !events[0].Success {
		t.Fatalf("ack: %+v", events)
	}
	if got := <-fake.auth; got != "Bearer test-token" {
		t.Fatal("authorization metadata missing")
	}
	request := <-fake.received
	if request.GetSubscribeOrderBookRequest().GetInstruments()[0].GetOrderBookType() != investapi.OrderBookType_ORDERBOOK_TYPE_EXCHANGE {
		t.Fatal("non-exchange subscription")
	}

	if err = stream.SubscribeInfo([]string{"uid"}, true); err != nil {
		t.Fatal(err)
	}
	events, err = stream.Receive()
	if err != nil || len(events) != 1 || events[0].Kind != density.EventInfoAck || !events[0].Success {
		t.Fatalf("info ack: %+v %v", events, err)
	}
	if req := <-fake.received; req.GetSubscribeInfoRequest() == nil {
		t.Fatal("missing info request")
	}
	if err = stream.SubscribeTrades([]string{"uid"}, true); err != nil {
		t.Fatal(err)
	}
	events, err = stream.Receive()
	if err != nil || len(events) != 1 || events[0].Kind != density.EventTradeAck || events[0].Success {
		t.Fatalf("trade rejection: %+v %v", events, err)
	}
	if req := <-fake.received; req.GetSubscribeTradesRequest().GetTradeSource() != investapi.TradeSourceType_TRADE_SOURCE_EXCHANGE {
		t.Fatal("non-exchange trades")
	}
	cancel()
	if _, err = stream.Receive(); err == nil {
		t.Fatal("canceled stream remained open")
	}
}
