// 这不是应用的一部分，而是一个独立的工具。
// 使用 `go run hash-password_test.go` 来运行它。

package utils

import (
	"fmt"
	"log"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	// 提示用户输入密码，并且不回显输入内容，更加安全
	//fmt.Print("请输入要加密的密码: ")
	//passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	//if err != nil {
	//	log.Fatalf("读取密码失败: %v", err)
	//}
	//fmt.Println() // 换行
	//
	passwordBytes := []byte("123456")

	// 使用 bcrypt 算法生成哈希。
	// bcrypt.DefaultCost 是一个推荐的、足够安全的计算强度。
	hashedPassword, err := bcrypt.GenerateFromPassword(passwordBytes, bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("生成哈希失败: %v", err)
	}

	// 打印生成的哈希字符串
	fmt.Println("\n--- 复制下面的哈希值到你的 config.json 文件中 ---")
	fmt.Printf("加密后的哈希: %s\n", string(hashedPassword))
}
