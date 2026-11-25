package pipeline

import (
	"testing"
	"time"

	investapi "tinvest/internal/pb/v1"
	"tinvest/pkg/client/grpc/converter"
	"tinvest/pkg/client/grpc/model"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestFinderRequiresCorporateAAA(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		name    string
		ratings string
		ofz     bool
		want    bool
	}{
		{"AAA", `[{"ratingLevel":"AAA"}]`, false, true},
		{"national scales", `[{"ratingLevel":"AAA(RU)"},{"ratingLevel":"ruAAA"},{"ratingLevel":"AAA.ru"},{"ratingLevel":"AAA|ru|"}]`, false, true},
		{"A", `[{"ratingLevel":"A"}]`, false, false},
		{"A minus", `[{"ratingLevel":"A-"}]`, false, false},
		{"split", `[{"ratingLevel":"ruAAA"},{"ratingLevel":"AA+"}]`, false, false},
		{"withdrawn", `[{"ratingLevel":"отозван"}]`, false, false},
		{"missing", `[]`, false, false},
		{"OFZ", `[]`, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var pb investapi.Bond
			if err := protojson.Unmarshal([]byte(`{"ratings":`+tc.ratings+`}`), &pb); err != nil {
				t.Fatal(err)
			}
			pb.Name = "Корпоративный выпуск"
			if tc.ofz {
				pb.Name = "ОФЗ 26238"
			}
			pb.RiskLevel = investapi.RiskLevel_RISK_LEVEL_LOW
			pb.LiquidityFlag = true
			pb.MaturityDate = timestamppb.New(now)
			pb.Nominal = &investapi.MoneyValue{Units: 1000}
			pb.AciValue = &investapi.MoneyValue{Units: 1}
			bond := converter.ConvertBondModelFromBondPb(&pb)
			_, ok := <-Finder(make(chan struct{}), []*model.Bond{bond}, tc.ofz, now.Add(-time.Hour), now.Add(time.Hour))
			if ok != tc.want {
				t.Fatalf("included = %v, want %v", ok, tc.want)
			}
		})
	}
}
