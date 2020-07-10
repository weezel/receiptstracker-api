package utils

import "testing"

func Test_isAllowedFileExt(t *testing.T) {
	type args struct {
		fname string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"Plain extension",
			args{"jpg"},
			false,
		},
		{
			"Allowed jpg",
			args{"t.jpg"},
			true,
		},
		{
			"Not allowed csv",
			args{"t.csv"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAllowedFileExt(tt.args.fname)
			if got != tt.want {
				t.Errorf("%s: IsAllowerFileExt() = %v, want %v",
					tt.name,
					got,
					tt.want)
			}
		})
	}
}
