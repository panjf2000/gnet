package math

import "testing"

func TestCeilToPowerOfTwo(t *testing.T) {
	type args struct {
		n int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{name: "test1", args: args{n: 1<<32 + 1}, want: 1 << 33},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CeilToPowerOfTwo(tt.args.n); got != tt.want {
				t.Errorf("CeilToPowerOfTwo() = %v, want %v", got, tt.want)
			}
		})
	}
}
