package live

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// A timeout, pending task or duplicate report never proves non-acceptance.
func definitelyRejectedOrder(err error) bool {
	if status.Code(err) == codes.PermissionDenied || status.Code(err) == codes.Unauthenticated {
		return true
	}
	if status.Code(err) != codes.InvalidArgument {
		return false
	}
	var rpc interface{ GRPCStatus() *status.Status }
	if !errors.As(err, &rpc) || rpc.GRPCStatus() == nil {
		return false
	}
	text := rpc.GRPCStatus().Message()
	if len(text) < 5 {
		return false
	}
	code := text[:5]
	for _, c := range code {
		if c < '0' || c > '9' {
			return false
		}
	}
	if len(text) > 5 && text[5] != ':' && text[5] != ' ' {
		return false
	}
	return code != "30057" && code != "30058" && code != "30049" && code != "30053"
}
