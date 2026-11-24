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
	ticks, aligned, err := d.splitTicks(price)
	if err != nil {
		return 0, err
	}
	if !aligned {
		return 0, fmt.Errorf("price %q is not aligned to the exchange tick", price)
	}
	return ticks, nil
}

// tradeTicks preserves the direction of strict price comparisons against
// aligned limits when the archive prints a finer price than the book tick.
func (d decimal) tradeTicks(price string, sell bool) (int64, error) {
	ticks, aligned, err := d.splitTicks(price)
	if err != nil {
		return 0, err
	}
	if !aligned && !sell {
		if ticks == math.MaxInt64 {
			return 0, fmt.Errorf("price %q exceeds tick range", price)
		}
		ticks++
	}
	return ticks, nil
}

func (d decimal) splitTicks(price string) (ticks int64, aligned bool, err error) {
	p, err := parseDecimal(price)
	if err != nil || p.mantissa == 0 {
		return 0, false, fmt.Errorf("invalid price %q", price)
	}
	aligned = p.scale <= d.scale
	for ; p.scale > d.scale && p.mantissa > 0; p.scale-- {
		p.mantissa /= 10
	}

	var high uint64
	low := p.mantissa
	if low == 0 {
		return 0, false, nil
	}
	for range d.scale - p.scale {
		carry, next := bits.Mul64(low, 10)
		if high > (math.MaxUint64-carry)/10 {
			return 0, false, fmt.Errorf("price %q exceeds tick range", price)
		}
		high, low = high*10+carry, next
	}
	if high >= d.mantissa {
		return 0, false, fmt.Errorf("price %q exceeds tick range", price)
	}
	units, remainder := bits.Div64(high, low, d.mantissa)
	if units > math.MaxInt64 {
		return 0, false, fmt.Errorf("price %q exceeds tick range", price)
	}
	return int64(units), aligned && remainder == 0, nil
}
