package walletsdk

import (
	"errors"
	"strings"
	"testing"
)

func TestGenerateMnemonic(t *testing.T) {
	for _, wc := range []int{12, 15, 18, 21, 24} {
		m, err := GenerateMnemonic(wc)
		if err != nil {
			t.Fatalf("wordCount %d: %v", wc, err)
		}
		if got := len(strings.Fields(m)); got != wc {
			t.Errorf("wordCount %d: 实际 %d 个词", wc, got)
		}
	}
	// 0 走默认
	m, err := GenerateMnemonic(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(strings.Fields(m)) != DefaultWordCount {
		t.Errorf("默认词数错误: %q", m)
	}
	// 非法词数
	for _, wc := range []int{11, 13, 25, -1} {
		if _, err := GenerateMnemonic(wc); !errors.Is(err, ErrInvalidWordCount) {
			t.Errorf("wordCount %d 应返回 ErrInvalidWordCount, got %v", wc, err)
		}
	}
}

func TestFromMnemonic_BothChains(t *testing.T) {
	mnemonic := testMnemonic
	for _, chain := range []Chain{ChainSolana, ChainTron} {
		w, err := FromMnemonic(chain, mnemonic, 0)
		if err != nil {
			t.Fatalf("%s: %v", chain, err)
		}
		if w.Address == "" || w.PublicKey == "" || w.PrivateKey == "" {
			t.Errorf("%s: 字段为空 %+v", chain, w)
		}
		if err := ValidateAddress(chain, w.Address); err != nil {
			t.Errorf("%s: 派生出的地址未通过校验: %v", chain, err)
		}
		// 确定性：同助记词同 index 结果一致
		w2, _ := FromMnemonic(chain, mnemonic, 0)
		if w2.Address != w.Address || w2.PrivateKey != w.PrivateKey {
			t.Errorf("%s: 派生不确定", chain)
		}
		// 不同 index 应不同
		w3, _ := FromMnemonic(chain, mnemonic, 1)
		if w3.Address == w.Address {
			t.Errorf("%s: index 0 与 1 地址相同（异常）", chain)
		}
	}
}

func TestFromMnemonic_Invalid(t *testing.T) {
	if _, err := FromMnemonic(ChainSolana, "not a mnemonic at all", 0); !errors.Is(err, ErrInvalidMnemonic) {
		t.Errorf("非法助记词应返回 ErrInvalidMnemonic, got %v", err)
	}
	if _, err := FromMnemonic(Chain("btc"), testMnemonic, 0); !errors.Is(err, ErrUnsupportedChain) {
		t.Errorf("不支持的链应返回 ErrUnsupportedChain, got %v", err)
	}
}

func TestFromMnemonic_WhitespaceTolerant(t *testing.T) {
	w, err := FromMnemonic(ChainSolana, "  "+testMnemonic+"\n", 0)
	if err != nil {
		t.Fatalf("多余空白应被容忍: %v", err)
	}
	base, _ := FromMnemonic(ChainSolana, testMnemonic, 0)
	if w.Address != base.Address {
		t.Error("空白归一化后地址应一致")
	}
}

func TestFromPrivateKey_RoundTrip(t *testing.T) {
	for _, chain := range []Chain{ChainSolana, ChainTron} {
		w, err := FromMnemonic(chain, testMnemonic, 3)
		if err != nil {
			t.Fatal(err)
		}
		w2, err := FromPrivateKey(chain, w.PrivateKey)
		if err != nil {
			t.Fatalf("%s: 私钥导入失败: %v", chain, err)
		}
		if w2.Address != w.Address || w2.PublicKey != w.PublicKey {
			t.Errorf("%s: 私钥导入结果不一致", chain)
		}
	}
}

func TestFromPrivateKey_Invalid(t *testing.T) {
	if _, err := FromPrivateKey(ChainSolana, "zzzz"); err == nil {
		t.Error("非法 hex 应报错")
	}
	if _, err := FromPrivateKey(ChainTron, "aabb"); err == nil {
		t.Error("长度不足应报错")
	}
	if _, err := FromPrivateKey(Chain("x"), testMnemonic); !errors.Is(err, ErrUnsupportedChain) {
		t.Error("不支持的链应报错")
	}
}

func TestValidateAddress_Invalid(t *testing.T) {
	if err := ValidateAddress(ChainSolana, ""); err == nil {
		t.Error("空地址应报错")
	}
	if err := ValidateAddress(ChainSolana, "0OIl"); err == nil {
		t.Error("非法 base58 应报错")
	}
	if err := ValidateAddress(ChainTron, "0x1234"); err == nil {
		t.Error("非 TRON 地址应报错")
	}
	if err := ValidateAddress(ChainTron, "TX"); err == nil {
		t.Error("过短地址应报错")
	}
	if err := ValidateAddress(Chain("x"), "whatever"); !errors.Is(err, ErrUnsupportedChain) {
		t.Error("不支持的链应报错")
	}
}

func TestPublicStripsSecrets(t *testing.T) {
	w, err := FromMnemonic(ChainSolana, testMnemonic, 0)
	if err != nil {
		t.Fatal(err)
	}
	p := w.Public()
	if p.PrivateKey != "" || p.Mnemonic != "" {
		t.Error("Public() 不应包含私钥/助记词")
	}
	if p.Address != w.Address {
		t.Error("Public() 应保留地址")
	}
	// 原对象不受影响
	if w.PrivateKey == "" {
		t.Error("Public() 不应修改原对象")
	}
}

func TestAssetChainMismatch(t *testing.T) {
	w, _ := FromMnemonic(ChainSolana, testMnemonic, 0)
	// 用 SOL 资产去查 TRON 地址 → 链不匹配
	_, err := Balance(nil, ChainTron, w.Address, SOL, "http://x")
	if !errors.Is(err, ErrAssetChainMismatch) {
		t.Errorf("应返回 ErrAssetChainMismatch, got %v", err)
	}
}
