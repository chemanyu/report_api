package core

import (
	"bytes"
	"crypto/aes"
	"encoding/base64"
	"fmt"
)

// AES-128-ECB加密
// Encrypt 使用AES-128-ECB模式加密文本并进行Base64编码。
// input 是要加密的明文，base64key 是Base64编码的密钥字符串。
// 返回加密后的Base64编码字符串以及可能出现的错误。
func EncryptAES128ECB(input string, base64key string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(base64key)
	if err != nil {
		return "", err
	}

	// 检查密钥长度是否为16字节（128位）
	if len(key) != 16 {
		return "", fmt.Errorf("invalid key size: %d bytes", len(key))
	}

	cipherBlock, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	plaintext := []byte(input)
	// 分块加密
	blockSize := cipherBlock.BlockSize()
	plaintext, err = pkcs7Pad(plaintext, blockSize)
	if len(plaintext)%blockSize != 0 {
		return "", fmt.Errorf("plaintext is not a multiple of the block size")
	}

	ciphertext := make([]byte, len(plaintext))
	for i := 0; i < len(plaintext); i += blockSize {
		cipherBlock.Encrypt(ciphertext[i:i+blockSize], plaintext[i:i+blockSize])
	}

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// 使用 PKCS#7 进行填充，这样任何长度的数据都可以被加密。
func pkcs7Pad(data []byte, blocksize int) ([]byte, error) {
	if blocksize <= 0 {
		return nil, fmt.Errorf("invalid blocksize")
	}
	if data == nil || len(data) == 0 {
		return nil, fmt.Errorf("invalid data")
	}
	padlen := blocksize - (len(data) % blocksize)
	padtext := bytes.Repeat([]byte{byte(padlen)}, padlen)
	return append(data, padtext...), nil
}
