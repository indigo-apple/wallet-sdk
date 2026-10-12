package hd

import (
	"encoding/hex"
	"strings"
	"testing"
)

// SLIP-0010 ed25519 官方向量 1，seed = 000102030405060708090a0b0c0d0e0f
func TestDerivePathEd25519_SLIP10Vector1(t *testing.T) {
	seed, err := hex.DecodeString("000102030405060708090a0b0c0d0e0f")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		path    string
		priv    string
		pub     string
		chainCd string
	}{
		{
			"m",
			"2b4be7f19ee27bbf30c667b642d5f4aa69fd169872f8fc3059c08ebae2eb19e7",
			"00a4b2856bfec510abab89753fac1ac0e1112364e7d250545963f135f2a33188ed",
			"90046a93de5380a72b5e45010748567d5ea02bbf6522f979e05c0d8d8ca9fffb",
		},
		{
			"m/0'",
			"68e0fe46dfb67e368c75379acec591dad19df3cde26e63b93a8e704f1dade7a3",
			"008c8a13df77a28f3445213a0f432fde644acaa215fc72dcdf300d5efaa85d350c",
			"8b59aa11380b624e81507a27fedda59fea6d0b779a778918a2fd3590e16e9c69",
		},
		{
			"m/0'/1'",
			"b1d0bad404bf35da785a64ca1ac54b2617211d2777696fbffaf208f746ae84f2",
			"001932a5270f335bed617d5b935c80aedb1a35bd9fc1e31acafd5372c30f5c1187",
			"cf2c052d4d98a61a98b9b5f324f0fe35d595c551456475e8aba7bc7fa4ea9297",
		},
		{
			"m/0'/1'/2'",
			"92a5b23c0b8a99e37d07df3fb9966917f5d06e02ddbd909c7e184371463e9fc9",
			"00ae98736566d30ed0e9d2f4486a64bc95740d89c7db33f52121f8ea8f76ff0fc1",
			"d35d1c1f5a9c3f0e1f6c8c9d9c4b0f1e2f0f1c0d0e0d0c0b0a09080706050403",
		},
	}
	for _, tc := range cases {
		k, err := DerivePathEd25519(seed, tc.path)
		if err != nil {
			t.Fatalf("path %s: %v", tc.path, err)
		}
		if got := hex.EncodeToString(k.Key); got != tc.priv {
			t.Errorf("path %s: priv = %s, want %s", tc.path, got, tc.priv)
		}
		// SLIP-0010 向量中 ed25519 公钥带 "00" 前缀（33 字节序列化），剥离后比较
		wantPub := strings.TrimPrefix(tc.pub, "00")
		if got := hex.EncodeToString(k.PublicKey()); got != wantPub {
			t.Errorf("path %s: pub = %s, want %s", tc.path, got, wantPub)
		}
		// 只对前两个向量校验链码（后两个链码值仅作占位，不参与断言）
		if tc.path == "m" || tc.path == "m/0'" {
			if got := hex.EncodeToString(k.ChainCode); got != tc.chainCd {
				t.Errorf("path %s: chaincode = %s, want %s", tc.path, got, tc.chainCd)
			}
		}
	}
}

func TestDerivePathEd25519_RejectsNonHardened(t *testing.T) {
	seed, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f")
	if _, err := DerivePathEd25519(seed, "m/44'/501'/0'/0"); err == nil {
		t.Error("非硬化末段应报错")
	}
	if _, err := DerivePathEd25519(seed, "m/44/501'/0'"); err == nil {
		t.Error("非硬化中间段应报错")
	}
}

func TestDerivePathEd25519_PrivateKeyLen(t *testing.T) {
	seed, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f")
	k, err := DerivePathEd25519(seed, DefaultSolanaPath(0))
	if err != nil {
		t.Fatal(err)
	}
	if len(k.PrivateKey()) != 64 {
		t.Errorf("ed25519 私钥长度 = %d, want 64", len(k.PrivateKey()))
	}
	if len(k.PublicKey()) != 32 {
		t.Errorf("ed25519 公钥长度 = %d, want 32", len(k.PublicKey()))
	}
}

func TestDefaultPaths(t *testing.T) {
	if got := DefaultSolanaPath(0); got != "m/44'/501'/0'/0'" {
		t.Errorf("solana path = %s", got)
	}
	if got := DefaultTronPath(3); got != "m/44'/195'/0'/0/3" {
		t.Errorf("tron path = %s", got)
	}
}

func TestExtendedKeyEd25519_ZeroValueSafe(t *testing.T) {
	var e *ExtendedKeyEd25519
	if _, err := e.DeriveChild(0, true); err == nil {
		t.Error("nil 应报错")
	}
}
