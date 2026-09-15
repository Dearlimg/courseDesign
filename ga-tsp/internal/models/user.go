// Package models 定义跨层共享的数据结构，不依赖任何业务层。
package models

// User 是对外暴露的用户信息（响应 DTO），不含敏感字段。
type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

// Account 是含密码哈希的持久化对象（PO），仅 dao 与 logic 使用。
type Account struct {
	User
	PasswordHash []byte
}
