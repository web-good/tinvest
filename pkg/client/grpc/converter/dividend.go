package converter

import (
	"time"

	investapi "tinvest/internal/pb/v1"
	"tinvest/pkg/client/grpc/model"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// ConvertDividendsFromPb converts dividend events, skipping nil entries. A missing timestamp
// becomes the zero time (timestamppb's AsTime on nil would yield 1970-01-01 instead).
func ConvertDividendsFromPb(in []*investapi.Dividend) []*model.Dividend {
	res := make([]*model.Dividend, 0, len(in))
	for _, d := range in {
		if d == nil {
			continue
		}
		res = append(res, &model.Dividend{
			LastBuyDate:  timeOrZero(d.GetLastBuyDate()),
			RecordDate:   timeOrZero(d.GetRecordDate()),
			DividendType: d.GetDividendType(),
		})
	}
	return res
}

func timeOrZero(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}
