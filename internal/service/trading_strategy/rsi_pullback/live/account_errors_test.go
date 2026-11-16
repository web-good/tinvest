package live

import (
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestDefiniteRejectionRecognizesWrappedBrokerDetails(t *testing.T) {
	err := fmt.Errorf("%w (message: invalid price)", status.Error(codes.InvalidArgument, "30078"))
	if !definitelyRejectedOrder(err) {
		t.Fatal("confirmed rejection treated as unknown acceptance")
	}
}
