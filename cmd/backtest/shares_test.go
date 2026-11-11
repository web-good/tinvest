package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"tinvest/internal/model"
	grpcclient "tinvest/pkg/client/grpc"
)

type shareClientStub struct {
	grpcclient.GrpcClient
	instruments *shareListStub
}

func (c shareClientStub) InstrumentsServiceClient() grpcclient.InstrumentsServiceClient {
	return c.instruments
}

type shareListStub struct {
	grpcclient.InstrumentsServiceClient
	shares []*model.Share
	err    error
}

func (s *shareListStub) Shares(context.Context) ([]*model.Share, error) {
	return s.shares, s.err
}

func TestResolveShareReusesDiskCacheAcrossTickers(t *testing.T) {
	t.Chdir(t.TempDir())
	client := shareClientStub{instruments: &shareListStub{shares: []*model.Share{
		{Ticker: "LKOH", ID: "lkoh-uid", Lot: 1},
		{Ticker: "GAZP", ID: "gazp-uid", Lot: 10},
	}}}
	got, err := resolveShare(context.Background(), client, "LKOH")
	if err != nil || got != (shareInfo{ID: "lkoh-uid", Lot: 1}) {
		t.Fatalf("first lookup: got %+v, err %v", got, err)
	}
	// A fresh client with an unavailable API simulates the next CLI process.
	offline := shareClientStub{instruments: &shareListStub{err: errors.New("unexpected EOF")}}
	got, err = resolveShare(context.Background(), offline, "GAZP")
	if err != nil || got != (shareInfo{ID: "gazp-uid", Lot: 10}) {
		t.Fatalf("cached lookup must work offline: got %+v, err %v", got, err)
	}
}

func TestResolveShareRefreshesUnusableCache(t *testing.T) {
	for _, tc := range []struct {
		name, data string
		expired    bool
	}{
		{"corrupt", "{", false},
		{"missing ticker", `{ "GAZP": {"ID":"gazp-uid","Lot":10} }`, false},
		{"invalid lot", `{ "LKOH": {"ID":"old-uid","Lot":0} }`, false},
		{"missing UID", `{ "LKOH": {"Lot":1} }`, false},
		{"expired", `{ "LKOH": {"ID":"old-uid","Lot":1} }`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			if err := os.MkdirAll(cacheDir, 0o755); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(cacheDir, "shares.json")
			if err := os.WriteFile(path, []byte(tc.data), 0o600); err != nil {
				t.Fatal(err)
			}
			if tc.expired {
				old := time.Now().Add(-25 * time.Hour)
				if err := os.Chtimes(path, old, old); err != nil {
					t.Fatal(err)
				}
			}
			client := shareClientStub{instruments: &shareListStub{shares: []*model.Share{{Ticker: "LKOH", ID: "new-uid", Lot: 10}}}}
			got, err := resolveShare(context.Background(), client, "LKOH")
			if err != nil || got != (shareInfo{ID: "new-uid", Lot: 10}) {
				t.Fatalf("got %+v, err %v", got, err)
			}
		})
	}
}

func TestResolveSharePreservesAPIErrors(t *testing.T) {
	t.Chdir(t.TempDir())
	want := errors.New("unexpected EOF")
	client := shareClientStub{instruments: &shareListStub{err: want}}
	if _, err := resolveShare(context.Background(), client, "LKOH"); !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "shares.json")); !os.IsNotExist(err) {
		t.Fatalf("failed lookup created cache: %v", err)
	}
}
