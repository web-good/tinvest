package feed

import (
	"fmt"
	"math"
	"math/bits"
	"strings"
)

// decimal keeps exchange price calculations in base ten. Trailing fractional zeros
// are removed before converting the mantissa to an integer.
type decimal struct {
	mantissa uint64
	scale    int
}

func parseDecimal(text string) (decimal, error) {
	whole, fraction, hasPoint := strings.Cut(text, ".")
	if whole == "" && fraction == "" || hasPoint && fraction == "" {
		return decimal{}, fmt.Errorf("invalid decimal %q", text)
	}
	fraction = strings.TrimRight(fraction, "0")
	var result decimal
	for _, digits := range [...]string{whole, fraction} {
		for i := range len(digits) {
			digit := uint64(digits[i] - '0')
			if digit > 9 || result.mantissa > (math.MaxUint64-digit)/10 {
				return decimal{}, fmt.Errorf("invalid decimal %q", text)
			}
			result.mantissa = result.mantissa*10 + digit
		}
	}
	result.scale = len(fraction)

	return result, nil
}

func (d decimal) ticks(price string) (int64, error) {
	p, err := parseDecimal(price)
	if err != nil || p.mantissa == 0 {
		return 0, fmt.Errorf("invalid price %q", price)
	}
	if p.scale > d.scale {
		return 0, fmt.Errorf("price %q is not aligned to the exchange tick", price)
	}

	var high uint64
	low := p.mantissa
	for range d.scale - p.scale {
		carry, next := bits.Mul64(low, 10)
		if high > (math.MaxUint64-carry)/10 {
			return 0, fmt.Errorf("price %q exceeds tick range", price)
		}
		high, low = high*10+carry, next
	}
	if high >= d.mantissa {
		return 0, fmt.Errorf("price %q exceeds tick range", price)
	}
	ticks, remainder := bits.Div64(high, low, d.mantissa)
	if remainder != 0 {
		return 0, fmt.Errorf("price %q is not aligned to the exchange tick", price)
	}
	if ticks > math.MaxInt64 {
		return 0, fmt.Errorf("price %q exceeds tick range", price)
	}

	return int64(ticks), nil
}
