// Package solana 是 Solana 链适配器（SOL 原生币 / SPL-USDT 代币）。
//
// 私钥以 64 字节 ed25519（seed||pubkey）十六进制表示。
package solana

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	solana "github.com/gagliardetto/solana-go"
	ata "github.com/gagliardetto/solana-go/programs/associated-token-account"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	bip39 "github.com/tyler-smith/go-bip39"

	"github.com/indigo-apple/wallet-sdk/internal/hd"
)

// 默认端点。
const (
	DefaultMainnetEndpoint = "https://api.mainnet-beta.solana.com"
	DefaultDevnetEndpoint  = "https://api.devnet.solana.com"
)

// Keypair 是 Solana 密钥对（十六进制编码）。
type Keypair struct {
	Address    string // Base58 地址
	PublicKey  string // 32 字节 ed25519 公钥，hex
	PrivateKey string // 64 字节 ed25519 私钥，hex
}

// DeriveKeypair 由助记词派生第 index 个账户（路径 m/44'/501'/index'/0'）。
func DeriveKeypair(mnemonic string, index uint32) (Keypair, error) {
	mnemonic = strings.Join(strings.Fields(mnemonic), " ")
	if !bip39.IsMnemonicValid(mnemonic) {
		return Keypair{}, errors.New("solana: 非法助记词")
	}
	seed := bip39.NewSeed(mnemonic, "")
	k, err := hd.DerivePathEd25519(seed, hd.DefaultSolanaPath(index))
	if err != nil {
		return Keypair{}, err
	}
	return keypairFromEd25519(ed25519.NewKeyFromSeed(k.Key)), nil
}

// KeypairFromPrivateKey 由私钥导入（接受 64 字节完整私钥或 32 字节种子）。
func KeypairFromPrivateKey(privHex string) (Keypair, error) {
	raw, err := decodeHex(privHex)
	if err != nil {
		return Keypair{}, err
	}
	switch len(raw) {
	case ed25519.PrivateKeySize:
		return keypairFromEd25519(ed25519.PrivateKey(raw)), nil
	case ed25519.SeedSize:
		return keypairFromEd25519(ed25519.NewKeyFromSeed(raw)), nil
	default:
		return Keypair{}, fmt.Errorf("solana: 私钥长度非法 %d（应为 32 或 64）", len(raw))
	}
}

func keypairFromEd25519(priv ed25519.PrivateKey) Keypair {
	pub := priv.Public().(ed25519.PublicKey)
	return Keypair{
		Address:    solana.PublicKeyFromBytes(pub).String(),
		PublicKey:  hex.EncodeToString(pub),
		PrivateKey: hex.EncodeToString(priv),
	}
}

// ValidateAddress 校验 Solana 地址（Base58 且为 32 字节公钥、非全零）。
func ValidateAddress(addr string) error {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return errors.New("solana: 地址为空")
	}
	pk, err := solana.PublicKeyFromBase58(addr)
	if err != nil {
		return fmt.Errorf("solana: 非法地址 %q: %w", addr, err)
	}
	if pk.IsZero() {
		return fmt.Errorf("solana: 地址为全零 %q", addr)
	}
	return nil
}

// Balance 查询余额，返回最小单位。isToken=false 时查原生 SOL(lamports)。
// 代币账户不存在时返回 0。
func Balance(ctx context.Context, addr string, isToken bool, mint, endpoint string) (*big.Int, error) {
	pk, err := solana.PublicKeyFromBase58(strings.TrimSpace(addr))
	if err != nil {
		return nil, fmt.Errorf("solana: 非法地址 %q: %w", addr, err)
	}
	c := rpc.New(endpoint)

	if !isToken {
		out, err := c.GetBalance(ctx, pk, rpc.CommitmentFinalized)
		if err != nil {
			return nil, fmt.Errorf("solana: 查询余额失败: %w", err)
		}
		return new(big.Int).SetUint64(out.Value), nil
	}

	mintPk, err := solana.PublicKeyFromBase58(strings.TrimSpace(mint))
	if err != nil {
		return nil, fmt.Errorf("solana: 非法 mint %q: %w", mint, err)
	}
	ataAddr, _, err := solana.FindAssociatedTokenAddress(pk, mintPk)
	if err != nil {
		return nil, fmt.Errorf("solana: 计算 ATA 失败: %w", err)
	}
	out, err := c.GetTokenAccountBalance(ctx, ataAddr, rpc.CommitmentFinalized)
	if err != nil {
		if isAccountNotFound(err) {
			return big.NewInt(0), nil
		}
		return nil, fmt.Errorf("solana: 查询代币余额失败: %w", err)
	}
	v, ok := new(big.Int).SetString(out.Value.Amount, 10)
	if !ok {
		return nil, fmt.Errorf("solana: 余额解析失败 %q", out.Value.Amount)
	}
	return v, nil
}

// BuildSignedTransfer 构造并签名一笔转账，但**不广播**。
//
// 主要用于测试（可交给 RPC 做 SimulateTransaction 验证），也可供上层自行广播。
// isToken=false 时 amount 为 lamports；否则为 SPL 代币最小单位。
// 代币转账会前置一条幂等的 ATA 创建指令（目标 ATA 已存在时无副作用）。
func BuildSignedTransfer(ctx context.Context, c *rpc.Client, privHex, to string, isToken bool, mint string, amount *big.Int, decimals int) (*solana.Transaction, error) {
	kp, err := KeypairFromPrivateKey(privHex)
	if err != nil {
		return nil, err
	}
	if amount == nil || amount.Sign() <= 0 {
		return nil, errors.New("solana: 金额必须大于 0")
	}
	if !amount.IsUint64() {
		return nil, errors.New("solana: 金额超出 uint64")
	}

	privBytes, err := hex.DecodeString(kp.PrivateKey)
	if err != nil {
		return nil, err
	}
	priv := solana.PrivateKey(privBytes)
	fromPk := priv.PublicKey()

	toPk, err := solana.PublicKeyFromBase58(strings.TrimSpace(to))
	if err != nil {
		return nil, fmt.Errorf("solana: 非法收款地址 %q: %w", to, err)
	}

	bh, err := c.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return nil, fmt.Errorf("solana: 获取 blockhash 失败: %w", err)
	}

	var ixs []solana.Instruction
	if !isToken {
		ixs = append(ixs, system.NewTransferInstruction(amount.Uint64(), fromPk, toPk).Build())
	} else {
		mintPk, err := solana.PublicKeyFromBase58(strings.TrimSpace(mint))
		if err != nil {
			return nil, fmt.Errorf("solana: 非法 mint %q: %w", mint, err)
		}
		srcAta, _, err := solana.FindAssociatedTokenAddress(fromPk, mintPk)
		if err != nil {
			return nil, err
		}
		dstAta, _, err := solana.FindAssociatedTokenAddress(toPk, mintPk)
		if err != nil {
			return nil, err
		}
		ixs = append(ixs, ata.NewCreateIdempotentInstruction(fromPk, toPk, mintPk).Build())
		ixs = append(ixs, token.NewTransferCheckedInstruction(
			amount.Uint64(), uint8(decimals), srcAta, mintPk, dstAta, fromPk, nil,
		).Build())
	}

	tx, err := solana.NewTransaction(ixs, bh.Value.Blockhash, solana.TransactionPayer(fromPk))
	if err != nil {
		return nil, fmt.Errorf("solana: 构造交易失败: %w", err)
	}
	if _, err := tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(fromPk) {
			return &priv
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("solana: 签名失败: %w", err)
	}
	return tx, nil
}

// Transfer 签名并广播转账，返回交易签名（Base58）。
//
// isToken=false 时 amount 为 lamports；否则为 SPL 代币最小单位。
func Transfer(ctx context.Context, privHex, to string, isToken bool, mint string, amount *big.Int, decimals int, endpoint string) (string, error) {
	c := rpc.New(endpoint)
	tx, err := BuildSignedTransfer(ctx, c, privHex, to, isToken, mint, amount, decimals)
	if err != nil {
		return "", err
	}
	sig, err := c.SendTransaction(ctx, tx)
	if err != nil {
		return "", fmt.Errorf("solana: 广播失败: %w", err)
	}
	return sig.String(), nil
}

func decodeHex(s string) ([]byte, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "0x")
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("solana: 非法 hex: %w", err)
	}
	return b, nil
}

func isAccountNotFound(err error) bool {
	if err == nil {
		return false
	}
	m := strings.ToLower(err.Error())
	return strings.Contains(m, "not found") || strings.Contains(m, "could not find account")
}
