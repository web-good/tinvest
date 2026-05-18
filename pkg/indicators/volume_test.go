package indicators

import "testing"

func TestVolumeConfirmed(t *testing.T) {
	tests := []struct {
		name       string
		volumes    []int64
		lookback   int
		multiplier float64
		want       bool
	}{
		{
			name:       "confirmed: last is 2x SMA",
			volumes:    append(repeatInt64(10, 20), 20),
			lookback:   20,
			multiplier: 1.5,
			want:       true,
		},
		{
			name:       "not confirmed: last equals SMA",
			volumes:    repeatInt64(10, 21),
			lookback:   20,
			multiplier: 1.5,
			want:       false,
		},
		{
			name:       "not confirmed: last below 1.5x SMA",
			volumes:    append(repeatInt64(10, 20), 14),
			lookback:   20,
			multiplier: 1.5,
			want:       false,
		},
		{
			name:       "boundary: last equals 1.5x SMA (strict >)",
			volumes:    append(repeatInt64(10, 20), 15),
			lookback:   20,
			multiplier: 1.5,
			want:       false,
		},
		{
			name:       "insufficient history",
			volumes:    repeatInt64(100, 5),
			lookback:   20,
			multiplier: 1.5,
			want:       false,
		},
		{
			name:       "empty input",
			volumes:    nil,
			lookback:   20,
			multiplier: 1.5,
			want:       false,
		},
		{
			name:       "different multiplier 2.0x boundary",
			volumes:    append(repeatInt64(10, 20), 20),
			lookback:   20,
			multiplier: 2.0,
			want:       false,
		},
		{
			name:       "different multiplier 2.0x pass",
			volumes:    append(repeatInt64(10, 20), 21),
			lookback:   20,
			multiplier: 2.0,
			want:       true,
		},
		{
			name:       "lookback zero is silent false",
			volumes:    repeatInt64(10, 5),
			lookback:   0,
			multiplier: 1.5,
			want:       false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := VolumeConfirmed(tc.volumes, tc.lookback, tc.multiplier)
			if got != tc.want {
				t.Fatalf("VolumeConfirmed = %v, want %v", got, tc.want)
			}
		})
	}
}

// repeatInt64 returns a slice of n copies of v.
func repeatInt64(v int64, n int) []int64 {
	out := make([]int64, n)
	for i := range out {
		out[i] = v
	}
	return out
}
