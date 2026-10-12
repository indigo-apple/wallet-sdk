package base58x

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestChecksumAndRoundTrip(t *testing.T) {
	payload := []byte("hello-tron")
	enc := CheckEncode(payload)
	got, err := CheckDecode(enc)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("round-trip = %x, want %x", got, payload)
	}
}

// TRON 官方已知地址：41(hex) 前缀 + 20 字节，Base58Check 后以 T 开头
func TestCheckEncode_TronAddress(t *testing.T) {
	// 一个公开已知的 TRON 地址字节（0x41 + 20B）
	addrBytes, err := hex.DecodeString("41e552f6487585c2b58bc2c9bb4492bc1f17132cd0")
	if err != nil {
		t.Fatal(err)
	}
	enc := CheckEncode(addrBytes)
	if enc[0] != 'T' {
		t.Errorf("TRON 地址应以 T 开头, got %q", enc)
	}
	// 解码回原字节
	back, err := CheckDecode(enc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(back, addrBytes) {
		t.Errorf("decode = %x, want %x", back, addrBytes)
	}
}

func TestCheckDecode_BadChecksum(t *testing.T) {
	enc := CheckEncode([]byte("data"))
	// 篡改最后一个字符
	mutated := []byte(enc)
	if mutated[len(mutated)-1] == '1' {
		mutated[len(mutated)-1] = '2'
	} else {
		mutated[len(mutated)-1] = '1'
	}
	if _, err := CheckDecode(string(mutated)); err == nil {
		t.Error("篡改后的校验和应报错")
	}
}

func TestCheckDecode_TooShort(t *testing.T) {
	if _, err := CheckDecode("1"); err == nil {
		t.Error("过短输入应报错")
	}
	if _, err := CheckDecode(""); err == nil {
		t.Error("空输入应报错")
	}
}

func TestRawRoundTrip(t *testing.T) {
	b := []byte{0, 1, 2, 3, 250, 255}
	s := RawEncode(b)
	got, err := RawDecode(s)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, b) {
		t.Errorf("raw round-trip = %x, want %x", got, b)
	}
}
