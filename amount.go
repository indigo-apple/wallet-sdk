package walletsdk

import (
	"fmt"
	"math/big"
	"strings"
)

// ParseAmount 把十进制金额字符串（如 "1.5"）按精度解析为最小单位整数。
//
// 不接受负数、空串、超过 decimals 的小数位；decimals 范围 0..38。
func ParseAmount(s string, decimals int) (*big.Int, error) {
	if decimals < 0 || decimals > 38 {
		return nil, fmt.Errorf("%w: 精度 %d", ErrInvalidAmount, decimals)
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("%w: 空", ErrInvalidAmount)
	}
	if strings.HasPrefix(s, "-") || strings.HasPrefix(s, "+") {
		return nil, fmt.Errorf("%w: 不支持符号 %q", ErrInvalidAmount, s)
	}
	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return nil, fmt.Errorf("%w: %q", ErrInvalidAmount, s)
	}
	intPart := parts[0]
	fracPart := ""
	if len(parts) == 2 {
		fracPart = parts[1]
	}
	if intPart == "" && fracPart == "" {
		return nil, fmt.Errorf("%w: %q", ErrInvalidAmount, s)
	}
	if intPart == "" {
		intPart = "0"
	}
	if !allDigits(intPart) || !allDigits(fracPart) {
		return nil, fmt.Errorf("%w: 非数字 %q", ErrInvalidAmount, s)
	}
	if len(fracPart) > decimals {
		return nil, fmt.Errorf("%w: 小数位超过 %d", ErrInvalidAmount, decimals)
	}
	fracPart += strings.Repeat("0", decimals-len(fracPart))
	v, ok := new(big.Int).SetString(intPart+fracPart, 10)
	if !ok {
		return nil, fmt.Errorf("%w: 解析失败 %q", ErrInvalidAmount, s)
	}
	return v, nil
}

// FormatAmount 把最小单位整数格式化为十进制字符串，去掉尾随零。
func FormatAmount(v *big.Int, decimals int) string {
	if v == nil {
		return "0"
	}
	neg := v.Sign() < 0
	av := new(big.Int).Abs(v)
	s := av.String()
	if decimals <= 0 {
		if neg {
			return "-" + s
		}
		return s
	}
	for len(s) <= decimals {
		s = "0" + s
	}
	intPart := s[:len(s)-decimals]
	fracPart := strings.TrimRight(s[len(s)-decimals:], "0")
	out := intPart
	if fracPart != "" {
		out += "." + fracPart
	}
	if neg {
		out = "-" + out
	}
	return out
}

func allDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// IsEmptyAmount 判断金额是否为零。
func IsEmptyAmount(v *big.Int) bool { return v == nil || v.Sign() == 0 }
