// Package tron 是 TRON 链适配器（TRX 原生币 / TRC20-USDT 代币）。
//
// 依赖 gotron-sdk 的 gRPC 客户端与签名器；私钥以 32 字节 secp256k1 十六进制表示。
package tron

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	api "github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/signer"
	"github.com/fbsobreira/gotron-sdk/pkg/txcore"
	bip39 "github.com/tyler-smith/go-bip39"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/indigo-apple/wallet-sdk/internal/hd"
)

// 默认 gRPC 端点。
const (
	DefaultMainnetEndpoint = "grpc.trongrid.io:50051"
	DefaultNileEndpoint    = "grpc.nile.trongrid.io:50051"
)

// DefaultFeeLimit 是 TRC20 转账默认的 TRX 能量费用上限（100 TRX）。
const DefaultFeeLimit int64 = 100_000_000

// Keypair 是 TRON 密钥对（十六进制编码）。
type Keypair struct {
	Address    string // Base58Check 地址（T 开头）
	PublicKey  string // 33 字节压缩公钥，hex
	PrivateKey string // 32 字节私钥，hex
}

// DeriveKeypair 由助记词派生第 index 个账户（路径 m/44'/195'/0'/0/index）。
func DeriveKeypair(mnemonic string, index uint32) (Keypair, error) {
	mnemonic = strings.Join(strings.Fields(mnemonic), " ")
	if !bip39.IsMnemonicValid(mnemonic) {
		return Keypair{}, errors.New("tron: 非法助记词")
	}
	seed := bip39.NewSeed(mnemonic, "")
	k, err := hd.DerivePath(seed, hd.DefaultTronPath(index))
	if err != nil {
		return Keypair{}, err
	}
	return keypairFromPrivBytes(k.Key), nil
}

// KeypairFromPrivateKey 由 32 字节私钥 hex 导入。
func KeypairFromPrivateKey(privHex string) (Keypair, error) {
	priv, err := privateKeyFromHex(privHex)
	if err != nil {
		return Keypair{}, err
	}
	return keypairFromECDSA(priv), nil
}

func privateKeyFromHex(privHex string) (*ecdsa.PrivateKey, error) {
	raw, err := decodeHex(privHex)
	if err != nil {
		return nil, err
	}
	if len(raw) != 32 {
		return nil, fmt.Errorf("tron: 私钥长度非法 %d（应为 32）", len(raw))
	}
	return secp256k1.PrivKeyFromBytes(raw).ToECDSA(), nil
}

func keypairFromPrivBytes(b []byte) Keypair {
	norm := make([]byte, 32)
	copy(norm[32-len(b):], b)
	return keypairFromECDSA(secp256k1.PrivKeyFromBytes(norm).ToECDSA())
}

func keypairFromECDSA(priv *ecdsa.PrivateKey) Keypair {
	kb := make([]byte, 32)
	priv.D.FillBytes(kb)
	sp := secp256k1.PrivKeyFromBytes(kb)
	return Keypair{
		Address:    address.PubkeyToAddress(priv.PublicKey).String(),
		PublicKey:  hex.EncodeToString(sp.PubKey().SerializeCompressed()),
		PrivateKey: hex.EncodeToString(kb),
	}
}

// ValidateAddress 校验 TRON Base58Check 地址。
func ValidateAddress(addr string) error {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return errors.New("tron: 地址为空")
	}
	a, err := address.Base58ToAddress(addr)
	if err != nil {
		return fmt.Errorf("tron: 非法地址 %q: %w", addr, err)
	}
	if !a.IsValid() {
		return fmt.Errorf("tron: 非法地址 %q", addr)
	}
	return nil
}

// Balance 查询余额，返回最小单位（TRX 为 sun）。isToken=false 查原生 TRX。
// 账户不存在时返回 0。
func Balance(ctx context.Context, endpoint, addr string, isToken bool, contract string) (*big.Int, error) {
	addr = strings.TrimSpace(addr)
	if err := ValidateAddress(addr); err != nil {
		return nil, err
	}
	c, err := dial(endpoint)
	if err != nil {
		return nil, err
	}
	defer c.Stop()

	if !isToken {
		acct, err := c.GetAccountCtx(ctx, addr)
		if err != nil {
			if isAccountNotFound(err) {
				return big.NewInt(0), nil
			}
			return nil, fmt.Errorf("tron: 查询 TRX 余额失败: %w", err)
		}
		return big.NewInt(acct.GetBalance()), nil
	}
	bal, err := c.TRC20ContractBalanceCtx(ctx, addr, strings.TrimSpace(contract))
	if err != nil {
		if isAccountNotFound(err) {
			return big.NewInt(0), nil
		}
		return nil, fmt.Errorf("tron: 查询 TRC20 余额失败: %w", err)
	}
	return bal, nil
}

// Transfer 构造、签名并广播转账，返回交易 ID（hex）。
//
// isToken=false 时用 TRX（amount 为 sun）；否则 TRC20 转账（amount 为最小单位）。
func Transfer(ctx context.Context, endpoint, privHex, to string, isToken bool, contract string, amount *big.Int) (string, error) {
	priv, err := privateKeyFromHex(privHex)
	if err != nil {
		return "", err
	}
	kp := keypairFromECDSA(priv)
	to = strings.TrimSpace(to)
	if err := ValidateAddress(to); err != nil {
		return "", err
	}
	if amount == nil || amount.Sign() <= 0 {
		return "", errors.New("tron: 金额必须大于 0")
	}

	c, err := dial(endpoint)
	if err != nil {
		return "", err
	}
	defer c.Stop()

	var txExt *api.TransactionExtention
	if isToken {
		txExt, err = c.TRC20SendCtx(ctx, kp.Address, to, strings.TrimSpace(contract), amount, DefaultFeeLimit)
	} else {
		if !amount.IsInt64() {
			return "", errors.New("tron: 金额超出 int64")
		}
		txExt, err = c.TransferCtx(ctx, kp.Address, to, amount.Int64())
	}
	if err != nil {
		return "", fmt.Errorf("tron: 构造交易失败: %w", err)
	}

	s, err := signer.NewPrivateKeySigner(priv)
	if err != nil {
		return "", fmt.Errorf("tron: 创建签名器失败: %w", err)
	}
	signed, err := s.Sign(txExt.GetTransaction())
	if err != nil {
		return "", fmt.Errorf("tron: 签名失败: %w", err)
	}

	ret, err := c.BroadcastCtx(ctx, signed)
	if err != nil {
		return "", fmt.Errorf("tron: 广播失败: %w", err)
	}
	if !ret.GetResult() {
		return "", fmt.Errorf("tron: 广播被拒绝 code=%d msg=%s", ret.GetCode(), string(ret.GetMessage()))
	}
	txid, err := txcore.TransactionID(signed)
	if err != nil {
		return "", fmt.Errorf("tron: 计算 txid 失败: %w", err)
	}
	return txid, nil
}

func dial(endpoint string) (*client.GrpcClient, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, errors.New("tron: 未配置节点地址")
	}
	c := client.NewGrpcClient(endpoint)
	if err := c.Start(grpc.WithTransportCredentials(insecure.NewCredentials())); err != nil {
		return nil, fmt.Errorf("tron: 连接节点 %s 失败: %w", endpoint, err)
	}
	return c, nil
}

func decodeHex(s string) ([]byte, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "0x")
	if s == "" {
		return nil, errors.New("tron: 空私钥")
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("tron: 非法 hex: %w", err)
	}
	return b, nil
}

func isAccountNotFound(err error) bool {
	if err == nil {
		return false
	}
	m := strings.ToLower(err.Error())
	return strings.Contains(m, "not found") || strings.Contains(m, "account not exist")
}
