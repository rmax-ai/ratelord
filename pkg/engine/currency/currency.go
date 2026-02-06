package currency

import "fmt"

// MicroUSD represents currency in millionths of a US Dollar.
// This allows for precise integer arithmetic without floating point errors.
type MicroUSD int64

const (
	// USD represents one US Dollar in MicroUSD.
	USD MicroUSD = 1_000_000
)

// String implements the fmt.Stringer interface.
// It formats the MicroUSD value as $X.XXXXXX.
func (m MicroUSD) String() string {
	dollars := int64(m) / int64(USD)
	micros := int64(m) % int64(USD)
	if micros < 0 {
		micros = -micros
	}
	return fmt.Sprintf("$%d.%06d", dollars, micros)
}
