package hd

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

// slip10SeedKey 是 SLIP-0010 规定的 ed25519 主密钥 HMAC key。
var slip10SeedKey = []byte("ed25519 seed")

// ExtendedKeyEd25519 是 SLIP-0010（ed25519）派生出的扩展密钥。
type ExtendedKeyEd25519 struct {
	Key       []byte // 32 字节私钥种子
	ChainCode []byte // 32 字节链码
	Depth     byte
	Index     uint32
}

// MasterFromSeedEd25519 由种子生成 ed25519 主扩展密钥（SLIP-0010）。
func MasterFromSeedEd25519(seed []byte) (*ExtendedKeyEd25519, error) {
	if len(seed) == 0 {
		return nil, errors.New("hd: 空种子")
	}
	mac := hmac.New(sha512.New, slip10SeedKey)
	mac.Write(seed)
	sum := mac.Sum(nil)
	return &ExtendedKeyEd25519{Key: sum[:32], ChainCode: sum[32:]}, nil
}

// DeriveChild 派生一层子密钥。
//
// SLIP-0010 的 ed25519 仅支持硬化派生，hardened=false 时返回错误。
func (e *ExtendedKeyEd25519) DeriveChild(index uint32, hardened bool) (*ExtendedKeyEd25519, error) {
	if e == nil || len(e.Key) != 32 || len(e.ChainCode) != 32 {
		return nil, errors.New("hd: 非法 ed25519 扩展密钥")
	}
	if !hardened {
		return nil, errors.New("hd: SLIP-0010 ed25519 仅支持硬化派生")
	}
	if index >= 0x80000000 {
		return nil, fmt.Errorf("hd: 索引越界 %d", index)
	}

	data := make([]byte, 0, 1+32+4)
	data = append(data, 0x00)
	data = append(data, e.Key...)
	var ib [4]byte
	binary.BigEndian.PutUint32(ib[:], index+0x80000000)
	data = append(data, ib[:]...)

	mac := hmac.New(sha512.New, e.ChainCode)
	mac.Write(data)
	sum := mac.Sum(nil)

	return &ExtendedKeyEd25519{
		Key:       sum[:32],
		ChainCode: sum[32:],
		Depth:     e.Depth + 1,
		Index:     index + 0x80000000,
	}, nil
}

// DerivePathEd25519 按路径派生（所有段必须是硬化段）。
func DerivePathEd25519(seed []byte, path string) (*ExtendedKeyEd25519, error) {
	k, err := MasterFromSeedEd25519(seed)
	if err != nil {
		return nil, err
	}
	segs, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	for _, s := range segs {
		if !s.hardened {
			return nil, fmt.Errorf("hd: ed25519 路径 %q 含非硬化段，非法", path)
		}
		k, err = k.DeriveChild(s.index, true)
		if err != nil {
			return nil, err
		}
	}
	return k, nil
}

// PublicKey 返回 32 字节 ed25519 公钥。
func (e *ExtendedKeyEd25519) PublicKey() []byte {
	priv := ed25519.NewKeyFromSeed(e.Key)
	return priv.Public().(ed25519.PublicKey)
}

// PrivateKey 返回 64 字节 ed25519 私钥（seed || pubkey）。
func (e *ExtendedKeyEd25519) PrivateKey() []byte {
	return ed25519.NewKeyFromSeed(e.Key)
}

// DefaultSolanaPath 生成 Solana 账户的默认派生路径。
func DefaultSolanaPath(account uint32) string {
	return fmt.Sprintf("m/44'/501'/%d'/0'", account)
}

// DefaultTronPath 生成 TRON 账户的默认派生路径（index 在末位、非硬化，
// 与 TronLink / gotron-sdk 约定一致）。
func DefaultTronPath(index uint32) string {
	return fmt.Sprintf("m/44'/195'/0'/0/%d", index)
}

// TrimHardened 便于测试：把 ' 归一化为 h。
func TrimHardened(p string) string { return strings.ReplaceAll(p, "'", "h") }
