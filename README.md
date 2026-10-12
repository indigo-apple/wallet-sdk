# wallet-sdk

多链钱包 SDK（**Solana** + **TRON**），提供创建/导入钱包、地址校验、余额查询、签名转账能力。

## 环境

开发统一走 Go 1.26 dev 容器（见项目根 `../.devcontainer`）：

```bash
cd ~/2T/trust-escrow && ./dev.sh      # 进入容器
```

## 支持的链与资产

| 链 | 原生币 | 稳定币 | 地址 | 默认派生路径 |
|---|---|---|---|---|
| Solana | SOL (9) | SPL-USDT (6) | Base58 ed25519 | `m/44'/501'/{i}'/0'` |
| TRON | TRX (6) | TRC20-USDT (6) | Base58Check (`T...`) | `m/44'/195'/0'/0/{i}` |

## 快速开始（Go）

```go
import (
    sdk "github.com/indigo-apple/wallet-sdk"
    "github.com/indigo-apple/wallet-sdk/solana"
)

// 1. 生成钱包
w, _ := sdk.NewWallet(sdk.ChainSolana, 12)
fmt.Println(w.Address, w.PrivateKey, w.Mnemonic)

// 2. 由助记词派生（多账户）
w2, _ := sdk.FromMnemonic(sdk.ChainTron, mnemonic, 0)

// 3. 校验地址
err := sdk.ValidateAddress(sdk.ChainSolana, addr)

// 4. 查余额（返回十进制字符串）
bal, _ := sdk.Balance(ctx, sdk.ChainSolana, w.Address, sdk.SOL, solana.DefaultMainnetEndpoint)

// 5. 转账（签名 + 广播，返回 txHash）
tx, _ := sdk.Transfer(ctx, sdk.ChainSolana, w.PrivateKey, to, sdk.SOL, "1.5", solana.DefaultMainnetEndpoint)
```

### 预置资产

`SOL` / `USDTSolana` / `TRX` / `USDTTRC20`。

## CLI

```bash
go run ./cmd/walletctl mnemonic --words 12
go run ./cmd/walletctl new --chain solana
go run ./cmd/walletctl from-mnemonic --chain tron --mnemonic "..." --index 0
go run ./cmd/walletctl from-privkey --chain solana --key <hex>
go run ./cmd/walletctl validate --chain solana --address <addr>
go run ./cmd/walletctl balance  --chain solana --address <addr> --asset SOL
go run ./cmd/walletctl transfer --chain tron --key <hex> --to <addr> --asset USDT --amount 1.5
```

所有命令支持 `--json`。

## 端点

| 链 | 测试网 | 主网 |
|---|---|---|
| Solana | `https://api.devnet.solana.com` | `https://api.mainnet-beta.solana.com` |
| TRON | `grpc.nile.trongrid.io:50051` | `grpc.trongrid.io:50051` |

> TRON 走 **gRPC**；生产建议自建节点或使用 TronGrid 付费额度。

## 测试

```bash
# 单元测试（含官方向量交叉验证）
go test -race -count=1 ./...

# 真实链上冒烟（需要网络）
go test -tags live -count=1 -run TestLive -v ./solana/ ./tron/
```

`internal/hd` 用 **BIP32 / SLIP-0010 官方测试向量**校验；`solana`、`tron` 包额外与上游库
（solana-go / gotron-sdk）做**派生结果交叉验证**。

## 安全说明

- **私钥仅以明文存在于内存 / 调用方**。本 SDK 不做密钥存储；落库加密（KMS）由业务系统负责。
- `sdk.Wallet.Public()` 返回剥离私钥/助记词的副本，可安全下发客户端。
- TRON 转账 `DefaultFeeLimit = 100 TRX`，如需调整请改常量或后续暴露为选项。

## 依赖

- `github.com/gagliardetto/solana-go` — Solana 交易构造 / RPC
- `github.com/fbsobreira/gotron-sdk` — TRON gRPC / 签名 / 地址
- `github.com/tyler-smith/go-bip39`、`github.com/mr-tron/base58`、`github.com/decred/dcrd/dcrec/secp256k1/v4`
