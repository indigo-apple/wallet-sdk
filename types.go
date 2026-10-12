// Package walletsdk 是多链钱包 SDK 的统一入口，支持 Solana（SOL/SPL-USDT）
// 与 TRON（TRX/TRC20-USDT）的钱包创建、导入、地址校验、余额查询与转账。
//
// 本包是门面层；链相关实现分别在 solana / tron 子包。
package walletsdk

import "fmt"

// Chain 是链标识。
type Chain string

const (
	ChainSolana Chain = "solana"
	ChainTron   Chain = "tron"
)

// Valid 判断是否为受支持的链。
func (c Chain) Valid() bool { return c == ChainSolana || c == ChainTron }

// String 返回链名。
func (c Chain) String() string { return string(c) }

// Asset 描述一种可操作资产。
type Asset struct {
	Symbol        string `json:"symbol"`
	Chain         Chain  `json:"chain"`
	Decimals      int    `json:"decimals"`
	Token         bool   `json:"token"`         // 是否合约代币
	TokenContract string `json:"tokenContract"` // SPL mint / TRC20 合约；原生币为空
}

// IsNative 是否为该链原生币。
func (a Asset) IsNative() bool { return !a.Token }

// 预置资产。
var (
	// SOL 是 Solana 原生币，9 位小数。
	SOL = Asset{Symbol: "SOL", Chain: ChainSolana, Decimals: 9}
	// USDTSolana 是 Solana 上的 SPL-USDT，6 位小数。
	USDTSolana = Asset{
		Symbol: "USDT", Chain: ChainSolana, Decimals: 6, Token: true,
		TokenContract: "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB",
	}
	// TRX 是 TRON 原生币，6 位小数。
	TRX = Asset{Symbol: "TRX", Chain: ChainTron, Decimals: 6}
	// USDTTRC20 是 TRON 上的 TRC20-USDT，6 位小数。
	USDTTRC20 = Asset{
		Symbol: "USDT", Chain: ChainTron, Decimals: 6, Token: true,
		TokenContract: "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
	}
)

// Wallet 是一个钱包账户的完整信息。
//
// PrivateKey / PublicKey 均为十六进制字符串；Mnemonic 仅在由助记词派生时填充。
type Wallet struct {
	Chain          Chain  `json:"chain"`
	Address        string `json:"address"`
	PublicKey      string `json:"publicKey"`
	PrivateKey     string `json:"privateKey"`
	Mnemonic       string `json:"mnemonic,omitempty"`
	DerivationPath string `json:"derivationPath,omitempty"`
	Index          uint32 `json:"index"`
}

// Public 返回只含公开信息的钱包副本（不含私钥/助记词），可安全下发给客户端。
func (w *Wallet) Public() *Wallet {
	c := *w
	c.PrivateKey = ""
	c.Mnemonic = ""
	return &c
}

func assetChainMismatch(asset Asset, chain Chain) error {
	return fmt.Errorf("%w: 资产 %s(%s) 与链 %s 不匹配", ErrAssetChainMismatch, asset.Symbol, asset.Chain, chain)
}
