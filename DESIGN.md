# wallet-sdk · 设计文档

> 多链钱包 SDK（Solana + TRON），用于「担保额度」业务系统的链上部分。
> 目标：创建/导入钱包、地址校验、余额查询、签名转账（出金）。

## 1. 范围

| 能力 | 说明 |
|---|---|
| 助记词 | BIP39（12/24 词），中英文词表 |
| 创建钱包 | 返回 address / publicKey / privateKey / mnemonic / derivationPath |
| 导入钱包 | 助记词 或 私钥 |
| 地址校验 | 按链规则校验格式 |
| 余额查询 | 原生币 + USDT（SPL / TRC20） |
| 签名转账 | 出金：签名 + 广播，返回 txHash |

不在范围（本期）：多签、硬件钱包、合约调用（非 USDT）、批量归集调度（后续）。

## 2. 链与币种

| 链 | 原生币 | 稳定币 | 曲线 | 派生路径 | 地址格式 |
|---|---|---|---|---|---|
| Solana | SOL(9) | SPL-USDT(6) mint `Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB` | Ed25519 | `m/44'/501'/{i}'/0'` | Base58 ed25519 pubkey |
| TRON | TRX(6) | TRC20-USDT(6) 合约 `TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t` | secp256k1 | `m/44'/195'/{i}/0/0` | Base58Check(`0x41`+keccak256(pub)[12:]) |

## 3. 技术选型

- **Go 1.26**（dev 容器 `trust-escrow-dev:go1.26`，GOTOOLCHAIN=local）。
- **Solana**：`github.com/gagliardetto/solana-go` **v1.25.0**（最新）——交易构造 / ATA / RPC。
- **TRON**：`github.com/fbsobreira/gotron-sdk` **v0.26.0**——地址 / 签名 / gRPC 客户端。
  - 私钥派生用本仓库自研 `internal/hd`（BIP32），并与 gotron-sdk 做交叉验证。
- 自研依赖：
  - `github.com/tyler-smith/go-bip39`（助记词）
  - `github.com/mr-tron/base58`（Base58/Base58Check）
  - `github.com/decred/dcrd/dcrec/secp256k1/v4`（TRON 密钥/签名底层）
  - `golang.org/x/crypto`（keccak256）

> 早期在宿主机 Go 1.21 下被迫降级（solana-go v1.14.0、gotron-sdk 不可用）；
> 改用 Go 1.26 dev 容器后已回到最新库，TRON 无需手写。


## 4. 目录结构

```
wallet-sdk/
  go.mod
  DESIGN.md
  types.go            # Chain / Asset / Wallet / Option
  errors.go           # 错误码
  wallet.go           # 统一门面：GenerateMnemonic / NewWallet / FromMnemonic / FromPrivateKey
  address.go          # 统一地址校验入口
  amount.go           # 金额十进制 ↔ 最小单位（int64/BigInt）
  internal/hd/        # 助记词 → 种子 → 派生（BIP32 secp256k1 / SLIP-0010 ed25519）
  internal/base58x/   # Base58Check 工具
  solana/             # Solana 适配器
  tron/               # TRON 适配器
  cmd/walletctl/      # CLI
```

## 5. 公开 API（门面，根包 `walletsdk`）

```go
func GenerateMnemonic(wordCount int) (string, error) // 12/24
func NewWallet(chain Chain, opts ...Option) (*Wallet, error)          // 新助记词+钱包
func FromMnemonic(chain Chain, mnemonic string, index uint32) (*Wallet, error)
func FromPrivateKey(chain Chain, privHex string) (*Wallet, error)
func ValidateAddress(chain Chain, addr string) error

func Balance(ctx, chain Chain, addr string, asset Asset, rpcURL string) (string, error)
func Transfer(ctx, w *Wallet, to string, asset Asset, amount string, rpcURL string) (txHash string, err error)
```

`Wallet`：
```go
type Wallet struct {
    Chain          Chain
    Address        string
    PublicKey      string // hex
    PrivateKey     string // hex
    Mnemonic       string // 仅助记词派生时填充
    DerivationPath string
    Index          uint32
}
```

## 6. 分阶段（每阶段带测试）

- **P0 加密核心** `internal/hd`：BIP39 种子、BIP32(secp256k1) 派生 —— BIP32 官方向量。
- **P1 地址** Solana/TRON 地址派生与校验 —— 向量 + 边界。
- **P2 余额** RPC 查询（只读）。
- **P3 转账** 签名 + 广播（Solana native/SPL、TRON TRX/TRC20）。
- **P4 CLI** `walletctl`。
- **P5 集成测试** devnet / nile 真实广播。

## 7. 风险

- Solana/TRON 交易构造涉及金额精度与地址字节序，出金前必须小额实测。
- 私钥仅在本 SDK 内部以明文内存存在；**落库加密由业务系统负责（KMS）**。
- TronGrid 公共节点有速率限制，生产需自建/付费节点。
