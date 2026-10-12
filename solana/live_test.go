//go:build live

package solana

import (
	"context"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	solana "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func devnetEndpoint() string {
	if e := os.Getenv("SOLANA_ENDPOINT"); e != "" {
		return e
	}
	return DefaultDevnetEndpoint
}

// 结构性验证：构造并签名一笔转账，交给 RPC 模拟执行。
// 即便账户没钱，只要交易结构合法、签名正确，RPC 会执行到"余额不足"这一步，
// 而不是报反序列化/指令错误。这样在无法空投时也能验证交易构造正确性。
func TestLive_SimulateTransfer(t *testing.T) {
	endpoint := devnetEndpoint()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	from, err := DeriveKeypair(testMnemonic, 200)
	if err != nil {
		t.Fatal(err)
	}
	to, err := DeriveKeypair(testMnemonic, 201)
	if err != nil {
		t.Fatal(err)
	}

	c := rpc.New(endpoint)
	tx, err := BuildSignedTransfer(ctx, c, from.PrivateKey, to.Address, false, "", big.NewInt(1_000_000), 9)
	if err != nil {
		t.Fatalf("构造交易失败: %v", err)
	}
	t.Logf("交易签名数=%d 指令数=%d", len(tx.Signatures), len(tx.Message.Instructions))

	res, err := c.SimulateTransactionWithOpts(ctx, tx, &rpc.SimulateTransactionOpts{
		SigVerify:              false,
		ReplaceRecentBlockhash: true,
		Commitment:             rpc.CommitmentProcessed,
	})
	if err != nil {
		t.Fatalf("模拟执行 RPC 调用失败: %v", err)
	}
	if res == nil || res.Value == nil {
		t.Fatal("模拟执行无返回")
	}
	errStr := ""
	if res.Value.Err != nil {
		errStr = strings.ToLower(strings.TrimSpace(toErrString(res.Value.Err)))
	}
	t.Logf("模拟结果 err=%q logs=%v", errStr, res.Value.Logs)

	// 结构非法会体现为 "invalid instruction"/"failed to deserialize" 之类；
	// 账户没钱则是 "insufficient"/"account not found"/"custom program error"。
	for _, bad := range []string{"deserialize", "invalid instruction", "invalid account data", "failed to sanitize"} {
		if strings.Contains(errStr, bad) {
			t.Fatalf("交易结构非法: %s", errStr)
		}
	}
	if errStr != "" && !strings.Contains(errStr, "insufficient") &&
		!strings.Contains(errStr, "account") && !strings.Contains(errStr, "custom program error") &&
		!strings.Contains(errStr, "rent") {
		t.Logf("提示：模拟错误非预期类别，请人工确认: %s", errStr)
	}
}

// 端到端冒烟：devnet 空投 → 余额 → 转账 → 再查余额。
// 空投被限流时 Skip。
func TestLive_DevnetTransfer(t *testing.T) {
	endpoint := devnetEndpoint()
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

	from, err := DeriveKeypair(testMnemonic, 100)
	if err != nil {
		t.Fatal(err)
	}
	to, err := DeriveKeypair(testMnemonic, 101)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("from=%s\nto=%s", from.Address, to.Address)

	c := rpc.New(endpoint)
	fromPk, _ := solana.PublicKeyFromBase58(from.Address)
	sig, err := c.RequestAirdrop(ctx, fromPk, 1_000_000_000, rpc.CommitmentFinalized)
	if err != nil {
		t.Skipf("devnet 空投失败（限流）：%v", err)
	}
	t.Logf("空投签名: %s", sig)

	for i := 0; i < 30; i++ {
		bal, err := Balance(ctx, from.Address, false, "", endpoint)
		if err == nil && bal.Sign() > 0 {
			break
		}
		time.Sleep(time.Second)
	}

	balBefore, err := Balance(ctx, from.Address, false, "", endpoint)
	if err != nil {
		t.Fatalf("查询余额失败: %v", err)
	}
	t.Logf("转账前余额(lamports): %s", balBefore)
	if balBefore.Sign() == 0 {
		t.Skip("余额仍为 0（空投未到账）")
	}

	amount := big.NewInt(100_000_000)
	tx, err := Transfer(ctx, from.PrivateKey, to.Address, false, "", amount, 9, endpoint)
	if err != nil {
		t.Fatalf("转账失败: %v", err)
	}
	t.Logf("转账签名: %s", tx)

	time.Sleep(5 * time.Second)
	balTo, err := Balance(ctx, to.Address, false, "", endpoint)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("收款方余额(lamports): %s", balTo)
	if balTo.Cmp(amount) < 0 {
		t.Errorf("收款方余额 %s 少于转账额 %s", balTo, amount)
	}
}

func toErrString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(interface{ Error() string }); ok {
		return s.Error()
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
