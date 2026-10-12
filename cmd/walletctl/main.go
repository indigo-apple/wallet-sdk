// Command walletctl 是 wallet-sdk 的命令行工具。
//
// 用法示例：
//
//	walletctl mnemonic
//	walletctl new --chain solana
//	walletctl from-mnemonic --chain tron --mnemonic "..." --index 0
//	walletctl from-privkey --chain solana --key <hex>
//	walletctl validate --chain solana --address <addr>
//	walletctl balance --chain solana --address <addr> --asset SOL
//	walletctl transfer --chain tron --key <hex> --to <addr> --asset USDT --amount 1.5
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	sdk "github.com/indigo-apple/wallet-sdk"
	"github.com/indigo-apple/wallet-sdk/solana"
	"github.com/indigo-apple/wallet-sdk/tron"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "mnemonic":
		err = cmdMnemonic(args)
	case "new":
		err = cmdNew(args)
	case "from-mnemonic":
		err = cmdFromMnemonic(args)
	case "from-privkey":
		err = cmdFromPrivkey(args)
	case "validate":
		err = cmdValidate(args)
	case "balance":
		err = cmdBalance(args)
	case "transfer":
		err = cmdTransfer(args)
	case "-h", "--help", "help":
		usage()
		return
	default:
		err = fmt.Errorf("未知命令 %q", cmd)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `walletctl — 多链钱包工具 (Solana / TRON)

命令:
  mnemonic       生成助记词              [--words 12]
  new            生成新钱包              --chain <solana|tron> [--words 12]
  from-mnemonic  由助记词派生钱包        --chain <...> --mnemonic "..." [--index 0]
  from-privkey   由私钥导入钱包          --chain <...> --key <hex>
  validate       校验地址                --chain <...> --address <addr>
  balance        查询余额                --chain <...> --address <addr> --asset <SOL|USDT|TRX> [--endpoint url]
  transfer       转账                    --chain <...> --key <hex> --to <addr> --asset <...> --amount 1.5 [--endpoint url]

所有命令支持 --json 输出 JSON。
`)
}

func printJSON(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func parseChain(s string) (sdk.Chain, error) {
	c := sdk.Chain(strings.ToLower(strings.TrimSpace(s)))
	if !c.Valid() {
		return "", fmt.Errorf("%w: %q", sdk.ErrUnsupportedChain, s)
	}
	return c, nil
}

func resolveAsset(chain sdk.Chain, symbol string) (sdk.Asset, error) {
	switch strings.ToUpper(strings.TrimSpace(symbol)) {
	case "SOL":
		return sdk.SOL, nil
	case "TRX":
		return sdk.TRX, nil
	case "USDT":
		if chain == sdk.ChainSolana {
			return sdk.USDTSolana, nil
		}
		return sdk.USDTTRC20, nil
	default:
		return sdk.Asset{}, fmt.Errorf("未知资产 %q（支持 SOL/TRX/USDT）", symbol)
	}
}

func defaultEndpoint(chain sdk.Chain) string {
	if chain == sdk.ChainSolana {
		return solana.DefaultMainnetEndpoint
	}
	return tron.DefaultMainnetEndpoint
}

func cmdMnemonic(args []string) error {
	fs := flag.NewFlagSet("mnemonic", flag.ContinueOnError)
	words := fs.Int("words", 12, "词数 12/15/18/21/24")
	asJSON := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return err
	}
	m, err := sdk.GenerateMnemonic(*words)
	if err != nil {
		return err
	}
	if *asJSON {
		return printJSON(map[string]string{"mnemonic": m})
	}
	fmt.Println(m)
	return nil
}

func cmdNew(args []string) error {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	chainStr := fs.String("chain", "", "链: solana|tron")
	words := fs.Int("words", 12, "词数")
	asJSON := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return err
	}
	chain, err := parseChain(*chainStr)
	if err != nil {
		return err
	}
	w, err := sdk.NewWallet(chain, *words)
	if err != nil {
		return err
	}
	return printWallet(w, *asJSON)
}

func cmdFromMnemonic(args []string) error {
	fs := flag.NewFlagSet("from-mnemonic", flag.ContinueOnError)
	chainStr := fs.String("chain", "", "链")
	mnemonic := fs.String("mnemonic", "", "助记词")
	index := fs.Uint("index", 0, "账户索引")
	asJSON := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return err
	}
	chain, err := parseChain(*chainStr)
	if err != nil {
		return err
	}
	w, err := sdk.FromMnemonic(chain, *mnemonic, uint32(*index))
	if err != nil {
		return err
	}
	return printWallet(w, *asJSON)
}

func cmdFromPrivkey(args []string) error {
	fs := flag.NewFlagSet("from-privkey", flag.ContinueOnError)
	chainStr := fs.String("chain", "", "链")
	key := fs.String("key", "", "私钥 hex")
	asJSON := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return err
	}
	chain, err := parseChain(*chainStr)
	if err != nil {
		return err
	}
	w, err := sdk.FromPrivateKey(chain, *key)
	if err != nil {
		return err
	}
	return printWallet(w, *asJSON)
}

func cmdValidate(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	chainStr := fs.String("chain", "", "链")
	addr := fs.String("address", "", "地址")
	asJSON := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return err
	}
	chain, err := parseChain(*chainStr)
	if err != nil {
		return err
	}
	verr := sdk.ValidateAddress(chain, *addr)
	if *asJSON {
		return printJSON(map[string]any{"valid": verr == nil, "error": errStr(verr)})
	}
	if verr != nil {
		return verr
	}
	fmt.Println("合法")
	return nil
}

func cmdBalance(args []string) error {
	fs := flag.NewFlagSet("balance", flag.ContinueOnError)
	chainStr := fs.String("chain", "", "链")
	addr := fs.String("address", "", "地址")
	assetStr := fs.String("asset", "", "资产 SOL|TRX|USDT")
	endpoint := fs.String("endpoint", "", "节点地址（默认主网）")
	asJSON := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return err
	}
	chain, err := parseChain(*chainStr)
	if err != nil {
		return err
	}
	asset, err := resolveAsset(chain, *assetStr)
	if err != nil {
		return err
	}
	if *endpoint == "" {
		*endpoint = defaultEndpoint(chain)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	bal, err := sdk.Balance(ctx, chain, *addr, asset, *endpoint)
	if err != nil {
		return err
	}
	if *asJSON {
		return printJSON(map[string]string{"chain": string(chain), "address": *addr, "asset": asset.Symbol, "balance": bal})
	}
	fmt.Printf("%s %s\n", bal, asset.Symbol)
	return nil
}

func cmdTransfer(args []string) error {
	fs := flag.NewFlagSet("transfer", flag.ContinueOnError)
	chainStr := fs.String("chain", "", "链")
	key := fs.String("key", "", "私钥 hex")
	to := fs.String("to", "", "收款地址")
	assetStr := fs.String("asset", "", "资产")
	amount := fs.String("amount", "", "金额")
	endpoint := fs.String("endpoint", "", "节点地址")
	asJSON := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return err
	}
	chain, err := parseChain(*chainStr)
	if err != nil {
		return err
	}
	asset, err := resolveAsset(chain, *assetStr)
	if err != nil {
		return err
	}
	if *endpoint == "" {
		*endpoint = defaultEndpoint(chain)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	tx, err := sdk.Transfer(ctx, chain, *key, *to, asset, *amount, *endpoint)
	if err != nil {
		return err
	}
	if *asJSON {
		return printJSON(map[string]string{"chain": string(chain), "txHash": tx})
	}
	fmt.Println(tx)
	return nil
}

func printWallet(w *sdk.Wallet, asJSON bool) error {
	if asJSON {
		return printJSON(w)
	}
	fmt.Printf("链:      %s\n", w.Chain)
	fmt.Printf("地址:    %s\n", w.Address)
	fmt.Printf("公钥:    %s\n", w.PublicKey)
	fmt.Printf("私钥:    %s\n", w.PrivateKey)
	if w.Mnemonic != "" {
		fmt.Printf("助记词:  %s\n", w.Mnemonic)
		fmt.Printf("派生路径:%s\n", w.DerivationPath)
	}
	return nil
}

func errStr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
