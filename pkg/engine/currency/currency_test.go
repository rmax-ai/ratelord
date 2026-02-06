package currency

import "testing"

func TestMicroUSD_String(t *testing.T) {
	tests := []struct {
		name string
		val  MicroUSD
		want string
	}{
		{
			name: "Zero",
			val:  0,
			want: "$0.000000",
		},
		{
			name: "One Dollar",
			val:  USD,
			want: "$1.000000",
		},
		{
			name: "One Dollar and a half",
			val:  USD + 500_000,
			want: "$1.500000",
		},
		{
			name: "Small amount",
			val:  1,
			want: "$0.000001",
		},
		{
			name: "Large amount",
			val:  1234567890,
			want: "$1234.567890",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.val.String(); got != tt.want {
				t.Errorf("MicroUSD.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
