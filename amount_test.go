package walletsdk

import (
	"math/big"
	"testing"
)

func TestParseAmount(t *testing.T) {
	cases := []struct {
		in       string
		decimals int
		want     string // 期望的最小单位整数字符串
		wantErr  bool
	}{
		{"1", 6, "1000000", false},
		{"1.5", 6, "1500000", false},
		{"0.000001", 6, "1", false},
		{".5", 6, "500000", false},
		{"1.", 6, "1000000", false},
		{"  2  ", 6, "2000000", false},
		{"1", 9, "1000000000", false},
		{"0", 6, "0", false},
		{"1.1234567", 6, "", true}, // 小数位超精度
		{"", 6, "", true},
		{"-1", 6, "", true},
		{"+1", 6, "", true},
		{"abc", 6, "", true},
		{"1.2.3", 6, "", true},
		{"1", -1, "", true},
		{"1", 39, "", true},
	}
	for _, tc := range cases {
		got, err := ParseAmount(tc.in, tc.decimals)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseAmount(%q,%d): 期望报错，实际 %s", tc.in, tc.decimals, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseAmount(%q,%d): %v", tc.in, tc.decimals, err)
			continue
		}
		if got.String() != tc.want {
			t.Errorf("ParseAmount(%q,%d) = %s, want %s", tc.in, tc.decimals, got, tc.want)
		}
	}
}

func TestFormatAmount(t *testing.T) {
	cases := []struct {
		in       string
		decimals int
		want     string
	}{
		{"1000000", 6, "1"},
		{"1500000", 6, "1.5"},
		{"1", 6, "0.000001"},
		{"0", 6, "0"},
		{"123456789", 6, "123.456789"},
		{"1000000000", 9, "1"},
	}
	for _, tc := range cases {
		v, _ := new(big.Int).SetString(tc.in, 10)
		if got := FormatAmount(v, tc.decimals); got != tc.want {
			t.Errorf("FormatAmount(%s,%d) = %s, want %s", tc.in, tc.decimals, got, tc.want)
		}
	}
	if FormatAmount(nil, 6) != "0" {
		t.Error("FormatAmount(nil) 应为 0")
	}
}

func TestAmountRoundTrip(t *testing.T) {
	for _, s := range []string{"1", "1.5", "0.000001", "123.456789", "0.1"} {
		v, err := ParseAmount(s, 6)
		if err != nil {
			t.Fatal(err)
		}
		if got := FormatAmount(v, 6); got != s {
			t.Errorf("round trip %q -> %q", s, got)
		}
	}
}

func TestIsEmptyAmount(t *testing.T) {
	if !IsEmptyAmount(nil) {
		t.Error("nil 应为空")
	}
	if !IsEmptyAmount(big.NewInt(0)) {
		t.Error("0 应为空")
	}
	if IsEmptyAmount(big.NewInt(1)) {
		t.Error("1 不应为空")
	}
}
