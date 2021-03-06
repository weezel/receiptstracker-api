package httpserver

import (
	"mime/multipart"
	"reflect"
	"testing"
	"time"
)

func Test_parseExpiryDate(t *testing.T) {
	t.Parallel()
	type args struct {
		tags      *[]string
		startDate time.Time
	}

	tests := []struct {
		name    string
		args    args
		want    time.Time
		wantErr bool
	}{
		{
			"No expiry date",
			args{&[]string{"4_habla", "f", "asdf"},
				time.Date(2019, 8, 6, 0, 0, 0, 0, time.UTC)},
			time.Time{},
			true,
		},
		{
			"Malformed expire date",
			args{&[]string{"4_habla", "d_days", "asdf"},
				time.Date(2019, 8, 6, 0, 0, 0, 0, time.UTC)},
			time.Time{},
			true,
		},
		{
			"Expiry date 1 day",
			args{&[]string{"4_habla", "not_a_date", "1_day"},
				time.Date(2019, 8, 6, 0, 0, 0, 0, time.UTC)},
			time.Date(2019, 8, 7, 0, 0, 0, 0, time.UTC),
			false,
		},
		{
			"Expiry date 60 days",
			args{&[]string{"4_habla", "not_a_date", "6_days"},
				time.Date(2019, 8, 6, 0, 0, 0, 0, time.UTC)},
			time.Date(2019, 8, 12, 0, 0, 0, 0, time.UTC),
			false,
		},
		{
			"Expiry date 1 month",
			args{&[]string{"4_habla", "not_a_date", "1_month"},
				time.Date(2019, 8, 6, 0, 0, 0, 0, time.UTC)},
			time.Date(2019, 9, 6, 0, 0, 0, 0, time.UTC),
			false,
		},
		{
			"Expiry date 6 months",
			args{&[]string{"4_habla", "not_a_date", "6_months"},
				time.Date(2019, 8, 6, 0, 0, 0, 0, time.UTC)},
			time.Date(2020, 2, 6, 0, 0, 0, 0, time.UTC),
			false,
		},
		{
			"Expiry date 1 year",
			args{&[]string{"4_habla", "not_a_date", "1_year"},
				time.Date(2019, 8, 6, 0, 0, 0, 0, time.UTC)},
			time.Date(2020, 8, 6, 0, 0, 0, 0, time.UTC),
			false,
		},
		{
			"Expiry date 6 years",
			args{&[]string{"4_habla", "not_a_date", "6_years"},
				time.Date(2019, 8, 6, 0, 0, 0, 0, time.UTC)},
			time.Date(2025, 8, 6, 0, 0, 0, 0, time.UTC),
			false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseExpiryDate(tt.args.tags, tt.args.startDate)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: ParseExpiryDate() error = %v, wantErr %v",
					tt.name,
					err,
					tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%s: ParseExpiryDate() = %v, want %v",
					tt.name,
					got,
					tt.want)
			}

		})
	}
}

func Test_parsePurchaseDate(t *testing.T) {
	t.Parallel()
	type args struct {
		tags *[]string
	}
	tests := []struct {
		name    string
		args    args
		want    time.Time
		wantErr bool
	}{
		{
			"Purchase date in correct format",
			args{&[]string{"2019-02-06", "testing", "not_a_date"}},
			time.Date(2019, 2, 6, 0, 0, 0, 0, time.UTC),
			false,
		},
		{
			"Purchase date in wrong format",
			args{&[]string{"02-06-2019", "testing", "not_a_date"}},
			time.Time{},
			true,
		},
		{
			"Noe date",
			args{&[]string{"testing", "not_a_date"}},
			time.Time{},
			true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParsePurchaseDate(tt.args.tags)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: ParsePurchaseDate() error = %v, wantErr %v",
					tt.name,
					err,
					tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%s: ParsePurchaseDate() = %v, want %v",
					tt.name,
					got,
					tt.want)
			}
		})
	}
}

func Test_normaliseTags(t *testing.T) {
	t.Parallel()
	type args struct {
		tags string
	}
	tests := []struct {
		name string
		args args
		want *[]string
	}{
		{
			"No tags",
			args{""},
			&[]string{},
		},
		{
			"One tag",
			args{"testtag"},
			&[]string{"testtag"},
		},
		{
			"Two tags",
			args{"test tag"},
			&[]string{"test", "tag"},
		},
		{
			"Much space wow",
			args{"te  st     tag"},
			&[]string{"te", "st", "tag"},
		},
		{
			"Leading space",
			args{" te   tag"},
			&[]string{"te", "tag"},
		},
		{
			"Trailing space",
			args{"te   tag "},
			&[]string{"te", "tag"},
		},
		{
			"Leading and trailing space",
			args{" te   tag "},
			&[]string{"te", "tag"},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NormaliseTags(tt.args.tags)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%s: NormaliseTags() = %v, want %v",
					tt.name,
					got,
					tt.want)
			}
		})
	}
}

func Test_calculateFileHash(t *testing.T) {
	type args struct {
		binFile         []byte
		formFileHeaders *multipart.FileHeader
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"Null byte",
			args{[]byte{}, &multipart.FileHeader{}},
			"",
			true,
		},
		{
			"Real content",
			args{[]byte{0, 1, 0, 1, 0},
				&multipart.FileHeader{Filename: "test-file.JPEG"},
			},
			"01e246b58d8e782fc96881c090d833eefa37e804cb308aeae0f7471c9ef1ea1a.jpeg",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculateFileHash(tt.args.binFile, tt.args.formFileHeaders)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: CalculateFileHash() error = %v, wantErr %v",
					tt.name,
					err,
					tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("%s: CalculateFileHash() = %v, want %v",
					tt.name,
					got,
					tt.want)
			}
		})
	}
}
