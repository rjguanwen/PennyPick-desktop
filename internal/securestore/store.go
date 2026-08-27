// Package securestore 提供 Windows DPAPI 保护的密钥存储。
//
// 用于桌面版首次解锁数据库后，把主密码用当前 Windows 账户凭据加密保存
// （CryptProtectData，绑定当前用户），之后启动可自动解锁、无需每次输入。
// 换账户 / 换机器后 DPAPI 无法解密，将自动退回手动输入主密码。
package securestore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Save 使用 Windows DPAPI 加密保存密钥到文件（权限仅当前用户可读写）。
func Save(path, secret string) error {
	if secret == "" {
		return errors.New("secret is empty")
	}
	enc, err := protect([]byte(secret))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create secret dir: %w", err)
	}
	if err := os.WriteFile(path, enc, 0o600); err != nil {
		return fmt.Errorf("write secret file: %w", err)
	}
	return nil
}

// Load 读取并解密密钥文件。
// 文件不存在、损坏或 DPAPI 无法解密（换账户/换机器）时返回 false。
func Load(path string) (string, bool) {
	enc, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	plain, err := unprotect(enc)
	if err != nil {
		return "", false
	}
	return string(plain), true
}

// Delete 删除已保存的密钥文件（不存在时视为成功）。
func Delete(path string) error {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// protect 用当前用户凭据加密数据。
func protect(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("empty data")
	}
	in := windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, nil, 0, nil, 0, &out); err != nil {
		return nil, fmt.Errorf("dpapi protect: %w", err)
	}
	// 输出缓冲区由系统分配，须在 LocalFree 前拷贝出来
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	result := make([]byte, int(out.Size))
	copy(result, unsafe.Slice(out.Data, int(out.Size)))
	return result, nil
}

// unprotect 解密 CryptProtectData 加密的数据。
func unprotect(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("empty data")
	}
	in := windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, nil, nil, 0, nil, 0, &out); err != nil {
		return nil, fmt.Errorf("dpapi unprotect: %w", err)
	}
	// 输出缓冲区由系统分配，须在 LocalFree 前拷贝出来
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	result := make([]byte, int(out.Size))
	copy(result, unsafe.Slice(out.Data, int(out.Size)))
	return result, nil
}
