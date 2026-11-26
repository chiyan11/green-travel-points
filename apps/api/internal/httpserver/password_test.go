package httpserver

import (
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
)

// 审查 P1-10：口令哈希从单轮 SHA-256 换成 PBKDF2-HMAC-SHA256。
// 这一组用例锁死四件事：格式、正确口令能过、错口令/被篡改的哈希过不去、
// 迭代数被改小（降级攻击）或旧 sha256$ 记录一律拒绝。

func TestPasswordHashRoundTrip(t *testing.T) {
	hash, err := passwordHash("demo123456")
	if err != nil {
		t.Fatalf("生成哈希失败: %v", err)
	}
	if !passwordMatches(hash, "demo123456") {
		t.Fatal("正确口令未通过校验")
	}
	if passwordMatches(hash, "demo123457") {
		t.Fatal("错误口令通过了校验")
	}
}

func TestPasswordHashFormat(t *testing.T) {
	hash, err := passwordHash("demo123456")
	if err != nil {
		t.Fatalf("生成哈希失败: %v", err)
	}
	parts := strings.Split(hash, "$")
	if len(parts) != 4 {
		t.Fatalf("格式 = %q，期望 pbkdf2$<iter>$<salt>$<digest>", hash)
	}
	if parts[0] != "pbkdf2" || parts[1] != "600000" {
		t.Fatalf("算法/迭代数 = %q/%q，期望 pbkdf2/600000", parts[0], parts[1])
	}
	if len(parts[2]) != 32 || len(parts[3]) != 64 {
		t.Fatalf("盐/摘要长度 = %d/%d 个 hex 字符，期望 32/64", len(parts[2]), len(parts[3]))
	}
}

func TestPasswordHashUsesDistinctSalts(t *testing.T) {
	first, err := passwordHash("demo123456")
	if err != nil {
		t.Fatalf("生成哈希失败: %v", err)
	}
	second, err := passwordHash("demo123456")
	if err != nil {
		t.Fatalf("生成哈希失败: %v", err)
	}
	if first == second {
		t.Fatal("两次哈希相同，说明没有随机盐")
	}
	for _, h := range []string{first, second} {
		if !passwordMatches(h, "demo123456") {
			t.Fatalf("带盐哈希未通过校验: %s", h)
		}
	}
}

func TestPasswordRejectsLegacySHA256(t *testing.T) {
	// PR #25 之前种子里的格式（也是 upgrade 前的存量格式），必须被拒绝。
	legacy := "sha256$726211e994a96a50be61dd7bd08aedee$e5b7d2e7e88ce334803ac33029c83cb451059ee6207f55309ecce7d0e8ca80d4"
	if passwordMatches(legacy, "demo123456") {
		t.Fatal("旧 sha256$ 哈希仍能通过校验，等于留着弱哈希后门")
	}
}

func TestPasswordRejectsTamperedDigest(t *testing.T) {
	hash, err := passwordHash("demo123456")
	if err != nil {
		t.Fatalf("生成哈希失败: %v", err)
	}
	parts := strings.Split(hash, "$")
	flipped := "0"
	if strings.HasSuffix(parts[3], "0") {
		flipped = "1"
	}
	parts[3] = parts[3][:len(parts[3])-1] + flipped
	if passwordMatches(strings.Join(parts, "$"), "demo123456") {
		t.Fatal("摘要被改过一个字符仍然通过校验")
	}
}

func TestPasswordRejectsLowIterationCount(t *testing.T) {
	// 用 1 次迭代构造一条哈希：计算全对，但迭代数低于下限，属于降级攻击。
	weak, err := pbkdf2.Key(sha256.New, "demo123456", []byte("0123456789abcdef"), 1000, 32)
	if err != nil {
		t.Fatalf("构造弱哈希失败: %v", err)
	}
	encoded := fmt.Sprintf("pbkdf2$1000$%s$%s",
		hex.EncodeToString([]byte("0123456789abcdef")), hex.EncodeToString(weak))
	if passwordMatches(encoded, "demo123456") {
		t.Fatal("迭代数 1000 的哈希仍被接受，应拒绝低迭代记录")
	}
}
