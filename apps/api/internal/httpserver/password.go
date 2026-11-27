package httpserver

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// 口令哈希用 PBKDF2-HMAC-SHA256（OWASP 2023 建议的迭代量级 600k、32 字节输出）。
//
// 之前是单轮加盐 SHA-256：离线爆破一次 SHA-256 只要几十纳秒，种子里的口令
// 等于明文。PBKDF2 每次校验要跑 60 万轮，靠的不是「更难的算法」而是可控的
// 计算成本。格式固定为 pbkdf2$<iter>$<hex salt>$<hex digest>，迭代数随哈希
// 一起存，以后调高不用改老数据。
const (
	pbkdf2Iterations = 600000
	pbkdf2KeyLength  = 32
	pbkdf2SaltLength = 16

	// minPBKDF2Iterations 是接受下限：库里的记录若被改成 1 次迭代，
	// 等于把口令降级成可爆破状态，直接判定校验失败。
	minPBKDF2Iterations = 100000
	// maxPBKDF2Iterations 是接受上限：防止被塞成天文数字后每次登录都打满 CPU。
	maxPBKDF2Iterations = 5000000
)

func passwordHash(password string) (string, error) {
	salt := make([]byte, pbkdf2SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	digest, err := pbkdf2.Key(sha256.New, password, salt, pbkdf2Iterations, pbkdf2KeyLength)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("pbkdf2$%d$%s$%s", pbkdf2Iterations,
		hex.EncodeToString(salt), hex.EncodeToString(digest)), nil
}

// passwordMatches 只接受 pbkdf2$ 格式。历史上的 sha256$ 记录一律拒绝
// （不留兼容分支：留了就等于给旧的弱哈希留了一条长期有效的后门）。
func passwordMatches(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2" {
		return false
	}
	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations < minPBKDF2Iterations || iterations > maxPBKDF2Iterations {
		return false
	}
	salt, err := hex.DecodeString(parts[2])
	if err != nil || len(salt) == 0 {
		return false
	}
	want, err := hex.DecodeString(parts[3])
	if err != nil || len(want) != pbkdf2KeyLength {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, iterations, len(want))
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(got, want) == 1
}
