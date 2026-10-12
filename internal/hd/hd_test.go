package hd

import (
	"encoding/hex"
	"strings"
	"testing"
)

// BIP32 官方向量 1（Test vector 1），seed = 000102030405060708090a0b0c0d0e0f
func TestDerivePath_BIP32Vector1(t *testing.T) {
	seed, err := hex.DecodeString("000102030405060708090a0b0c0d0e0f")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		path string
		xprv string
	}{
		{"m", "e8f32e723decf4051aefac8e2c93c9c5b214313817cdb01a1494b917c8436b35"},
		{"m/0'", "edb2e14f9ee77d26dd93b4ecede8d16ed408ce149b6cd80b0715a2d911a0afea"},
		{"m/0'/1", "3c6cb8d0f6a264c91ea8b5030fadaa8e538b020f0a387421a12de9319dc93368"},
		{"m/0'/1/2'", "cbce0d719ecf7431d88e6a89fa1483e02e35092af60c042b1df2ff59fa424dca"},
		{"m/0'/1/2'/2", "0f479245fb19a38a1954c5c7c0ebab2f9bdfd96a17563ef28a6a4b1a2a764ef4"},
		{"m/0'/1/2'/2/1000000000", "471b76e389e528d6de6d816857e012c5455051cad6660850e58372a6c3e6e7c8"},
		// 硬化标记 h / H 等价于 '
		{"m/0h/1", "3c6cb8d0f6a264c91ea8b5030fadaa8e538b020f0a387421a12de9319dc93368"},
	}
	for _, tc := range cases {
		k, err := DerivePath(seed, tc.path)
		if err != nil {
			t.Fatalf("path %s: %v", tc.path, err)
		}
		if got := hex.EncodeToString(k.Key); got != tc.xprv {
			t.Errorf("path %s: key = %s, want %s", tc.path, got, tc.xprv)
		}
	}
}

func TestMasterFromSeed_ChainCodeAndPubkey(t *testing.T) {
	seed, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f")
	k, err := MasterFromSeed(seed)
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(k.ChainCode); got != "873dff81c02f525623fd1fe5167eac3a55a049de3d314bb42ee227ffed37d508" {
		t.Errorf("chaincode = %s", got)
	}
	if got := hex.EncodeToString(k.PublicKeyCompressed()); got != "0339a36013301597daef41fbe593a02cc513d0b55527ec2df1050e2e8ff49c85c2" {
		t.Errorf("pubkey = %s", got)
	}
	// 未压缩公钥应为 65 字节且以 0x04 开头
	un := k.PublicKeyUncompressed()
	if len(un) != 65 || un[0] != 0x04 {
		t.Errorf("uncompressed pubkey 格式错误: %x", un)
	}
}

// 派生路径 m/0'/1 的链码与公钥（向量 1）
func TestDeriveChild_DepthAndIndex(t *testing.T) {
	seed, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f")
	k, err := DerivePath(seed, "m/0'/1")
	if err != nil {
		t.Fatal(err)
	}
	if k.Depth != 2 {
		t.Errorf("depth = %d, want 2", k.Depth)
	}
	if k.Index != 1 {
		t.Errorf("index = %d, want 1", k.Index)
	}
}

func TestDerivePath_Errors(t *testing.T) {
	seed, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f")
	bad := []string{
		"",             // 空
		"44'/195'/0",   // 不以 m 开头
		"m/",           // 空段
		"m//1",         // 空段
		"m/abc",        // 非数字
		"m/-1",         // 负数
		"m/2147483648", // 越界 (2^31)
	}
	for _, p := range bad {
		if _, err := DerivePath(seed, p); err == nil {
			t.Errorf("path %q: 期望报错，实际通过", p)
		}
	}
}

func TestMasterFromSeed_Empty(t *testing.T) {
	if _, err := MasterFromSeed(nil); err == nil {
		t.Error("空种子应报错")
	}
	if _, err := DerivePath(nil, "m/0"); err == nil {
		t.Error("空种子应报错")
	}
}

// 零值/非法扩展密钥调用不应 panic
func TestExtendedKey_ZeroValueSafe(t *testing.T) {
	var e *ExtendedKey
	if _, err := e.DeriveChild(0, false); err == nil {
		t.Error("nil 扩展密钥应报错")
	}
	bad := &ExtendedKey{Key: []byte{1, 2}, ChainCode: []byte{3}}
	if _, err := bad.DeriveChild(0, false); err == nil {
		t.Error("非法长度应报错")
	}
}

func TestParsePath_TrailingH(t *testing.T) {
	segs, err := parsePath("m/44'/195H/0'")
	if err != nil {
		t.Fatal(err)
	}
	want := []pathSegment{{44, true}, {195, true}, {0, true}}
	if len(segs) != len(want) {
		t.Fatalf("段数 = %d", len(segs))
	}
	for i := range want {
		if segs[i] != want[i] {
			t.Errorf("段 %d = %+v, want %+v", i, segs[i], want[i])
		}
	}
	if !strings.Contains("m/44'/195'/0'/0/0", "'") {
		t.Fatal("sanity")
	}
}
