//go:build live

package tron

import (
	"context"
	"os"
	"testing"
	"time"
)

// 真实链上冒烟测试：Nile 测试网查余额（只读，验证 gRPC 连通与地址可用）。
// 运行：go test -tags live -run TestLive_Nile -v ./tron/
func TestLive_NileBalance(t *testing.T) {
	endpoint := os.Getenv("TRON_ENDPOINT")
	if endpoint == "" {
		endpoint = DefaultNileEndpoint
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	kp, err := DeriveKeypair(testMnemonic, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("地址: %s", kp.Address)

	bal, err := Balance(ctx, endpoint, kp.Address, false, "") // TRX
	if err != nil {
		t.Fatalf("查询 TRX 余额失败: %v", err)
	}
	t.Logf("TRX 余额(sun): %s", bal)

	usdt, err := Balance(ctx, endpoint, kp.Address, true, "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	if err != nil {
		t.Logf("查询 USDT 余额失败（可忽略）: %v", err)
	} else {
		t.Logf("USDT 余额: %s", usdt)
	}
}
