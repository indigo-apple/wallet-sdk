// Package base58x 提供带校验和的 Base58 编解码（Base58Check）。
//
// 校验和取 payload 的双 SHA-256 前 4 字节，用于 TRON 地址等场景。
// 并发安全。
package base58x

import (
	"bytes"
	"crypto/sha256"
	"errors"

	"github.com/mr-tron/base58"
)

// checksumLen 是校验和字节数。
const checksumLen = 4

// Checksum 返回 payload 的双 SHA-256 前 4 字节。
func Checksum(payload []byte) []byte {
	first := sha256.Sum256(payload)
	second := sha256.Sum256(first[:])
	return second[:checksumLen]
}

// CheckEncode 先追加 4 字节校验和，再 Base58 编码。
func CheckEncode(payload []byte) string {
	buf := make([]byte, 0, len(payload)+checksumLen)
	buf = append(buf, payload...)
	buf = append(buf, Checksum(payload)...)
	return base58.Encode(buf)
}

// CheckDecode 解码 Base58 并校验尾部校验和，返回去掉校验和的原始 payload。
func CheckDecode(s string) ([]byte, error) {
	raw, err := base58.Decode(s)
	if err != nil {
		return nil, err
	}
	if len(raw) < checksumLen {
		return nil, errors.New("base58x: 数据过短")
	}
	payload := raw[:len(raw)-checksumLen]
	want := raw[len(raw)-checksumLen:]
	if !bytes.Equal(Checksum(payload), want) {
		return nil, errors.New("base58x: 校验和不匹配")
	}
	return payload, nil
}

// RawEncode 不带校验和的 Base58 编码（Solana 地址用）。
func RawEncode(b []byte) string { return base58.Encode(b) }

// RawDecode 不带校验和的 Base58 解码。
func RawDecode(s string) ([]byte, error) { return base58.Decode(s) }
