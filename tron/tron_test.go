package tron

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/keys"
)

const testMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

// 与本仓库自研 BIP32 实现交叉验证：结果必须与 gotron-sdk 一致。
func TestDeriveKeypair_MatchesGotron(t *testing.T) {
	for i := uint32(0); i < 3; i++ {
		kp, err := DeriveKeypair(testMnemonic, i)
		if err != nil {
			t.Fatalf("index %d: %v", i, err)
		}
		priv, _ := keys.FromMnemonicSeedAndPassphrase(testMnemonic, "", int(i))
		want := address.BTCECPrivkeyToAddress(priv).String()
		if kp.Address != want {
			t.Errorf("index %d: 地址 = %s, want %s", i, kp.Address, want)
		}
	}
}

func TestKeypairFromPrivateKey_RoundTrip(t *testing.T) {
	kp, err := DeriveKeypair(testMnemonic, 5)
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
}

func TestKeypairFromPrivateKey_Invalid(t *testing.T) {
	for _, bad := range []string{"", "zz", "aabb", "0x11223344556677889900112233445566778899001122334455667788990011223"} {
		if _, err := KeypairFromPrivateKey(bad); err == nil {
			t.Errorf("%q 应报错", bad)
		}
	}
}

func TestValidateAddress(t *testing.T) {
	// 从派生得到的合法地址
	kp, _ := DeriveKeypair(testMnemonic, 0)
	if err := ValidateAddress(kp.Address); err != nil {
		t.Errorf("合法地址被拒: %v", err)
	}
	bad := []string{"", "0x1234", "TX", "1111111111111111111111111111111111"}
	for _, a := range bad {
		if err := ValidateAddress(a); err == nil {
			t.Errorf("非法地址 %q 未报错", a)
		}
	}
}
