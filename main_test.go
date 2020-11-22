package main

import (
	"testing"
)

func Test_reversePIN(t *testing.T) {
	type args struct {
		pinblock string
		pan      string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"testing reverse pin", args{pinblock: "d122f06d07b3ef95", pan: "9222081700176714465"}, "0000", false},
		{"testing reverse pin", args{pinblock: "d122f06d07b3ef95", pan: "9222081700176714465"}, "1234", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := reversePIN(tt.args.pinblock, tt.args.pan)

			if got != tt.want {
				t.Errorf("reversePIN() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseStripe(t *testing.T) {
	type args struct {
		res string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"parse successfull", args{res: "Request req_sj1ekhBbk31Aq2: This value must be greater than or equal to 1."}, "This value must be greater than or equal to 1."},
		{"parse successfull", args{res: "Request req_sj1ekhBbk31Aq2-- This value must be greater than or equal to 1."}, "Request req_sj1ekhBbk31Aq2-- This value must be greater than or equal to 1."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseStripe(tt.args.res); got != tt.want {
				t.Errorf("parseStripe() = %v, want %v", got, tt.want)
			}
		})
	}
}
