// Package hd 提供助记词与分层确定性钱包（HD Wallet）的派生能力。
//
// 支持两条派生标准：
//   - BIP32（secp256k1），用于 TRON（路径 m/44'/195'/...）
//   - SLIP-0010（ed25519），用于 Solana（路径 m/44'/501'/...）
//
// 并发安全。
package hd

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

// secp256k1 曲线阶 n。
var curveN = mustBigHex("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141")

func mustBigHex(s string) *big.Int {
	n, ok := new(big.Int).SetString(s, 16)
	if !ok {
		panic("hd: invalid curve order constant")
	}
	return n
}

// ExtendedKey 是一层派生后的扩展私钥（BIP32）。
type ExtendedKey struct {
	Key       []byte // 32 字节私钥
	ChainCode []byte // 32 字节链码
	Depth     byte   // 派生深度
	Index     uint32 // 本层索引（含硬化位）
}

// masterSeedKey 是 BIP32 规定的 HMAC key。
var masterSeedKey = []byte("Bitcoin seed")

// MasterFromSeed 由种子生成主扩展私钥（BIP32）。
//
// seed 通常为 64 字节（BIP39）。
func MasterFromSeed(seed []byte) (*ExtendedKey, error) {
	if len(seed) == 0 {
		return nil, errors.New("hd: 空种子")
	}
	mac := hmac.New(sha512.New, masterSeedKey)
	mac.Write(seed)
	sum := mac.Sum(nil)
	il, ir := sum[:32], sum[32:]

	k := new(big.Int).SetBytes(il)
	if k.Sign() == 0 || k.Cmp(curveN) >= 0 {
		return nil, errors.New("hd: 非法主密钥（IL 越界）")
	}
	return &ExtendedKey{Key: il, ChainCode: ir}, nil
}

// DeriveChild 派生一层子密钥。
//
// hardened 为真时使用硬化派生。index 需 < 2^31。
func (e *ExtendedKey) DeriveChild(index uint32, hardened bool) (*ExtendedKey, error) {
	if e == nil || len(e.Key) != 32 || len(e.ChainCode) != 32 {
		return nil, errors.New("hd: 非法扩展密钥")
	}
	if index >= 0x80000000 {
		return nil, fmt.Errorf("hd: 索引越界 %d", index)
	}
	idx := index + 0x80000000
	if !hardened {
		idx = index
	}

	var data []byte
	if hardened {
		data = make([]byte, 0, 1+32+4)
		data = append(data, 0x00)
		data = append(data, e.Key...)
	} else {
		pub := secp256k1.PrivKeyFromBytes(e.Key).PubKey().SerializeCompressed()
		data = make([]byte, 0, 33+4)
		data = append(data, pub...)
	}
	var ib [4]byte
	binary.BigEndian.PutUint32(ib[:], idx)
	data = append(data, ib[:]...)

	mac := hmac.New(sha512.New, e.ChainCode)
	mac.Write(data)
	sum := mac.Sum(nil)
	il, ir := sum[:32], sum[32:]

	ilInt := new(big.Int).SetBytes(il)
	if ilInt.Cmp(curveN) >= 0 {
		return nil, fmt.Errorf("hd: IL 越界，索引 %d", idx)
	}
	child := new(big.Int).Add(ilInt, new(big.Int).SetBytes(e.Key))
	child.Mod(child, curveN)
	if child.Sign() == 0 {
		return nil, fmt.Errorf("hd: 子密钥为零，索引 %d", idx)
	}
	kb := make([]byte, 32)
	child.FillBytes(kb)

	return &ExtendedKey{Key: kb, ChainCode: ir, Depth: e.Depth + 1, Index: idx}, nil
}

// DerivePath 按 BIP32 路径一次性派生，如 "m/44'/195'/0'/0/0"。
//
// 硬化标记支持 ' 或 h/H 后缀。
func DerivePath(seed []byte, path string) (*ExtendedKey, error) {
	if len(seed) == 0 {
		return nil, errors.New("hd: 空种子")
	}
	k, err := MasterFromSeed(seed)
	if err != nil {
		return nil, err
	}
	segments, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	for _, s := range segments {
		k, err = k.DeriveChild(s.index, s.hardened)
		if err != nil {
			return nil, err
		}
	}
	return k, nil
}

type pathSegment struct {
	index    uint32
	hardened bool
}

func parsePath(path string) ([]pathSegment, error) {
	p := strings.TrimSpace(path)
	if p == "" {
		return nil, errors.New("hd: 空路径")
	}
	parts := strings.Split(p, "/")
	if parts[0] != "m" && parts[0] != "M" {
		return nil, fmt.Errorf("hd: 路径必须以 m 开头: %q", path)
	}
	segs := make([]pathSegment, 0, len(parts)-1)
	for _, raw := range parts[1:] {
		if raw == "" {
			return nil, fmt.Errorf("hd: 非法路径 %q", path)
		}
		hardened := false
		if last := raw[len(raw)-1]; last == '\'' || last == 'h' || last == 'H' {
			hardened = true
			raw = raw[:len(raw)-1]
		}
		n, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("hd: 非法路径段 %q", raw)
		}
		if n >= 0x80000000 {
			return nil, fmt.Errorf("hd: 路径段越界 %q", raw)
		}
		segs = append(segs, pathSegment{index: uint32(n), hardened: hardened})
	}
	return segs, nil
}

// PublicKeyCompressed 返回压缩公钥（33 字节）。
func (e *ExtendedKey) PublicKeyCompressed() []byte {
	return secp256k1.PrivKeyFromBytes(e.Key).PubKey().SerializeCompressed()
}

// PublicKeyUncompressed 返回未压缩公钥（65 字节，含 0x04 前缀）。
func (e *ExtendedKey) PublicKeyUncompressed() []byte {
	return secp256k1.PrivKeyFromBytes(e.Key).PubKey().SerializeUncompressed()
}
