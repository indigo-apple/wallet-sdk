package solana

import (
	"encoding/hex"
	"fmt"
	"testing"

	solana "github.com/gagliardetto/solana-go"
)

const testMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

// 与本仓库自研 SLIP-0010 实现交叉验证：结果必须与 solana-go 官方派生一致。
func TestDeriveKeypair_MatchesSolanaGo(t *testing.T) {
	for i := uint32(0); i < 3; i++ {
		kp, err := DeriveKeypair(testMnemonic, i)
		if err != nil {
			t.Fatalf("index %d: %v", i, err)
		}
		path := fmt.Sprintf("m/44'/501'/%d'/0'", i)
		want, err := solana.PrivateKeyFromMnemonicAtPath(testMnemonic, "", path)
		if err != nil {
			t.Fatalf("solana-go 派生失败: %v", err)
		}
		if kp.Address != want.PublicKey().String() {
			t.Errorf("index %d: 地址 = %s, want %s", i, kp.Address, want.PublicKey().String())
		}
		if kp.PrivateKey != hex.EncodeToString(want) {
			t.Errorf("index %d: 私钥不匹配", i)
		}
		if kp.PublicKey != hex.EncodeToString(want.PublicKey().Bytes()) {
			t.Errorf("index %d: 公钥不匹配", i)
		}
	}
}

func TestKeypairFromPrivateKey_RoundTrip(t *testing.T) {
	kp, err := DeriveKeypair(testMnemonic, 0)
	if err != nil {
		t.Fatal(err)
	}
	got, err := KeypairFromPrivateKey(kp.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	if got.Address != kp.Address || got.PublicKey != kp.PublicKey {
		t.Error("私钥往返不一致")
	}
	// 32 字节种子也应可导入
	seed := kp.PrivateKey[:64] // 64 hex 字符 = 32 字节
	got2, err := KeypairFromPrivateKey(seed)
	if err != nil {
		t.Fatal(err)
	}
	if got2.Address != kp.Address {
		t.Error("32 字节种子导入地址不一致")
	}
}

func TestKeypairFromPrivateKey_Invalid(t *testing.T) {
	for _, bad := range []string{"", "zz", "aabb", "0x1122"} {
		if _, err := KeypairFromPrivateKey(bad); err == nil {
			t.Errorf("%q 应报错", bad)
		}
	}
}

func TestValidateAddress(t *testing.T) {
	good := []string{
		"9xQeWvG816bUx9EPjHmaT23yvVM2ZWbrrpZb9PusVFin",
	}
	for _, a := range good {
		if err := ValidateAddress(a); err != nil {
			t.Errorf("合法地址 %q 被拒: %v", a, err)
		}
	}
	bad := []string{"", "0OIl", "too-short", "11111111111111111111111111111111"} // 最后一个是全零
	for _, a := range bad {
		if err := ValidateAddress(a); err == nil {
			t.Errorf("非法地址 %q 未报错", a)
		}
	}
}
