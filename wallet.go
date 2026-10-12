package walletsdk

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	bip39 "github.com/tyler-smith/go-bip39"

	"github.com/indigo-apple/wallet-sdk/internal/hd"
	"github.com/indigo-apple/wallet-sdk/solana"
	"github.com/indigo-apple/wallet-sdk/tron"
)

// DefaultWordCount 是默认助记词词数。
const DefaultWordCount = 12

// GenerateMnemonic 生成助记词，wordCount 支持 12/15/18/21/24，传 0 用默认 12。
func GenerateMnemonic(wordCount int) (string, error) {
	if wordCount == 0 {
		wordCount = DefaultWordCount
	}
	switch wordCount {
	case 12, 15, 18, 21, 24:
	default:
		return "", fmt.Errorf("%w: %d（支持 12/15/18/21/24）", ErrInvalidWordCount, wordCount)
	}
	entropy, err := bip39.NewEntropy(wordCount / 3 * 32)
	if err != nil {
		return "", err
	}
	return bip39.NewMnemonic(entropy)
}

// NewWallet 生成新助记词并派生钱包（index=0）。wordCount 传 0 用默认。
func NewWallet(chain Chain, wordCount int) (*Wallet, error) {
	m, err := GenerateMnemonic(wordCount)
	if err != nil {
		return nil, err
	}
	return FromMnemonic(chain, m, 0)
}

// FromMnemonic 由助记词派生第 index 个账户的钱包。
func FromMnemonic(chain Chain, mnemonic string, index uint32) (*Wallet, error) {
	if !chain.Valid() {
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedChain, chain)
	}
	mnemonic = strings.Join(strings.Fields(mnemonic), " ")
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, ErrInvalidMnemonic
	}
	switch chain {
	case ChainSolana:
		kp, err := solana.DeriveKeypair(mnemonic, index)
		if err != nil {
			return nil, err
		}
		return &Wallet{
			Chain: chain, Address: kp.Address, PublicKey: kp.PublicKey, PrivateKey: kp.PrivateKey,
			Mnemonic: mnemonic, DerivationPath: hd.DefaultSolanaPath(index), Index: index,
		}, nil
	case ChainTron:
		kp, err := tron.DeriveKeypair(mnemonic, index)
		if err != nil {
			return nil, err
		}
		return &Wallet{
			Chain: chain, Address: kp.Address, PublicKey: kp.PublicKey, PrivateKey: kp.PrivateKey,
			Mnemonic: mnemonic, DerivationPath: hd.DefaultTronPath(index), Index: index,
		}, nil
	}
	return nil, fmt.Errorf("%w: %q", ErrUnsupportedChain, chain)
}

// FromPrivateKey 由私钥 hex 导入钱包。
func FromPrivateKey(chain Chain, privHex string) (*Wallet, error) {
	if !chain.Valid() {
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedChain, chain)
	}
	switch chain {
	case ChainSolana:
		kp, err := solana.KeypairFromPrivateKey(privHex)
		if err != nil {
			return nil, err
		}
		return &Wallet{Chain: chain, Address: kp.Address, PublicKey: kp.PublicKey, PrivateKey: kp.PrivateKey}, nil
	case ChainTron:
		kp, err := tron.KeypairFromPrivateKey(privHex)
		if err != nil {
			return nil, err
		}
		return &Wallet{Chain: chain, Address: kp.Address, PublicKey: kp.PublicKey, PrivateKey: kp.PrivateKey}, nil
	}
	return nil, fmt.Errorf("%w: %q", ErrUnsupportedChain, chain)
}

// ValidateAddress 校验指定链的地址格式。
func ValidateAddress(chain Chain, addr string) error {
	switch chain {
	case ChainSolana:
		return solana.ValidateAddress(addr)
	case ChainTron:
		return tron.ValidateAddress(addr)
	}
	return fmt.Errorf("%w: %q", ErrUnsupportedChain, chain)
}

// Balance 查询余额，返回十进制字符串（按 asset.Decimals）。
func Balance(ctx context.Context, chain Chain, addr string, asset Asset, endpoint string) (string, error) {
	if !chain.Valid() {
		return "", fmt.Errorf("%w: %q", ErrUnsupportedChain, chain)
	}
	if asset.Chain != chain {
		return "", assetChainMismatch(asset, chain)
	}
	var (
		v   *big.Int
		err error
	)
	switch chain {
	case ChainSolana:
		v, err = solana.Balance(ctx, addr, asset.Token, asset.TokenContract, endpoint)
	case ChainTron:
		v, err = tron.Balance(ctx, endpoint, addr, asset.Token, asset.TokenContract)
	}
	if err != nil {
		return "", err
	}
	return FormatAmount(v, asset.Decimals), nil
}

// Transfer 转账（签名+广播），返回交易哈希。
//
// amount 为十进制字符串；endpoint 为对应链的 RPC/gRPC 端点。
func Transfer(ctx context.Context, chain Chain, privHex, to string, asset Asset, amount, endpoint string) (string, error) {
	if !chain.Valid() {
		return "", fmt.Errorf("%w: %q", ErrUnsupportedChain, chain)
	}
	if asset.Chain != chain {
		return "", assetChainMismatch(asset, chain)
	}
	if err := ValidateAddress(chain, to); err != nil {
		return "", err
	}
	amt, err := ParseAmount(amount, asset.Decimals)
	if err != nil {
		return "", err
	}
	if amt.Sign() <= 0 {
		return "", fmt.Errorf("%w: 金额必须大于 0", ErrInvalidAmount)
	}
	switch chain {
	case ChainSolana:
		if !amt.IsUint64() {
			return "", fmt.Errorf("%w: 金额超出 uint64", ErrInvalidAmount)
		}
		return solana.Transfer(ctx, privHex, to, asset.Token, asset.TokenContract, amt, asset.Decimals, endpoint)
	case ChainTron:
		if !amt.IsInt64() {
			return "", fmt.Errorf("%w: 金额超出 int64", ErrInvalidAmount)
		}
		return tron.Transfer(ctx, endpoint, privHex, to, asset.Token, asset.TokenContract, amt)
	}
	return "", fmt.Errorf("%w: %q", ErrUnsupportedChain, chain)
}
