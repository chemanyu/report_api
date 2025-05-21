package common

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
)

// 获取真实IP
func GetAddr(c *gin.Context) string {

	c_ip := c.GetHeader("X-Forwarded-For")

	if c_ip != "" {
		return c_ip
	}
	c_ip = c.ClientIP()
	if c_ip != "" {
		return c_ip
	}

	return ""
}

// 到达转化参数拼接
func AutoArriveParams(target_link, code string) string {
	//判断是否与锚点
	md := strings.Split(target_link, "#")

	raw := strings.Split(md[0], "?")
	//到达转化参数拼接
	var arriveParams string
	if (len(md) == 1 && len(raw) == 1) || (len(md) == 1 && len(raw) > 1) || md[len(md)-1] == "" {
		//没有锚点和queryRaw，锚点拼接参数    || 有QueryRaw 没锚点，锚点拼接参数 || 锚点没设置时
		if md[len(md)-1] == "" { //有#号没内容
			arriveParams = target_link + "&" + code
		} else {
			arriveParams = target_link + "#&" + code
		}

	} else if len(md) > 1 && len(raw) == 1 {
		//有锚点，没QueryRaw，Raw拼接参数
		if !strings.HasSuffix(raw[0], "/") {
			raw[0] = raw[0] + "/"
		}
		arriveParams = raw[0] + "?" + code + "#" + md[len(md)-1]
	} else if len(md) > 1 && len(raw) > 1 {
		if strings.Index(raw[len(raw)-1], "&") > -1 {
			//原链接有queryRaw 进&追加
			arriveParams = md[0] + code
		} else {
			//原链接无queryRaw 直接进行拼接
			arriveParams = md[0] + "&" + code
		}
		arriveParams = arriveParams + "#" + md[len(md)-1]
	}
	return arriveParams
}

//@author: [piexlmax](https://github.com/piexlmax)
//@function: MD5V
//@description: md5加密
//@param: str []byte
//@return: string

func MD5V(str []byte, b ...byte) string {
	h := md5.New()
	h.Write(str)
	return hex.EncodeToString(h.Sum(b))
}

func StrMd5(str string) string {
	return MD5V([]byte(str))
}

func StrSha256(str, key string) string {
	// 创建SHA256哈希对象
	hash := hmac.New(sha256.New, []byte(key))
	// 写入待编码的数据
	hash.Write([]byte(str))
	// 获取SHA256编码结果
	hashBytes := hash.Sum(nil)

	// 将编码结果转换为十六进制字符串
	encodedStr := hex.EncodeToString(hashBytes)
	return encodedStr
}

func Generate_UUID() (string, error) {
	newUUID, err := uuid.NewUUID()
	if err != nil {
		return "", err
	}
	return newUUID.String(), nil
}

// 生成用户ID
func Generate_uid(str string) string {
	if str == "" {
		return ""
	}

	str = strings.ToLower(str)
	if len(str) != 32 {
		str = StrMd5(str)
	} else {
		if ok, _ := regexp.MatchString("(?i)(^[0-9a-f]{32}$)", str); !ok {
			str = StrMd5(str)
		}
	}

	str = StrMd5(str)
	dst, _ := hex.DecodeString(str)
	return base64.StdEncoding.EncodeToString(dst[4:13])
}

var ivspec = []byte("0000000000000000")

func AESEncodeStr(src, key string) string {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		fmt.Println("key error1", err)
	}
	if src == "" {
		fmt.Println("plain content empty")
	}
	ecb := cipher.NewCBCEncrypter(block, ivspec)
	content := []byte(src)
	content = PKCS5Padding(content, block.BlockSize())
	crypted := make([]byte, len(content))
	ecb.CryptBlocks(crypted, content)
	return hex.EncodeToString(crypted)
}

func AESDecodeStr(crypt, key string) string {
	crypted, err := hex.DecodeString(strings.ToLower(crypt))
	if err != nil || len(crypted) == 0 {
		fmt.Println("plain content empty")
	}
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		fmt.Println("key error1", err)
	}
	ecb := cipher.NewCBCDecrypter(block, ivspec)
	decrypted := make([]byte, len(crypted))
	ecb.CryptBlocks(decrypted, crypted)

	return string(PKCS5Trimming(decrypted))
}

func PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func PKCS5Trimming(encrypt []byte) []byte {
	padding := encrypt[len(encrypt)-1]
	return encrypt[:len(encrypt)-int(padding)]
}

func InArray(element string, array []string) bool {
	for _, v := range array {
		if v == element {
			return true
		}
	}
	return false
}

func MapKeys(query string, mappings map[string]string) (string, error) {

	// 解析原始查询字符串为url.Values对象
	values, err := url.ParseQuery(query)
	if err != nil {
		return "", err
	}

	newUrl := url.Values{}
	// 遍历url.Values对象，映射key到另一个名称
	for oldkey, oldValues := range values {
		if newKey, ok := mappings[oldkey]; ok {
			// 将新的key及对应的值添加到url.Values对象中
			for _, oldValue := range oldValues {
				if oldValue != "" {
					newUrl.Set(newKey, oldValue)
				}
			}
		}
		//delete(values, oldkey) // 删除原始的key
	}

	// 构建映射后的查询字符串
	mappedQuery := newUrl.Encode()

	return mappedQuery, nil
}

func GetLastValue(values url.Values, key string) string {
	vs, ok := values[key]
	if !ok || len(vs) == 0 {
		return ""
	}
	// 返回切片中的最后一个元素
	return vs[len(vs)-1]
}

func ZstdCompressAndBase64Encode(data []byte) (string, error) {
	// 压缩数据
	var b bytes.Buffer
	encoder, err := zstd.NewWriter(&b, zstd.WithEncoderLevel(4))
	if err != nil {
		return "", err
	}
	if _, err := encoder.Write(data); err != nil {
		return "", err
	}
	encoder.Close()

	// 使用Base64编码
	str := base64.URLEncoding.EncodeToString(b.Bytes())

	// URL编码
	escapedStr := url.QueryEscape(str)
	return escapedStr, nil
}

// 解压缩并进行Base64解码
func ZstdDecompressAndBase64Decode(encoded string) ([]byte, error) {
	// GIN 自动会解码 不需要重复URL解码
	decodedStr, err := url.QueryUnescape(encoded)
	if err != nil {
		return nil, err
	}

	// Base64解码
	data, err := base64.URLEncoding.DecodeString(decodedStr)
	if err != nil {
		return nil, err
	}

	// 解压缩数据
	rData := bytes.NewReader(data)
	decoder, err := zstd.NewReader(rData)
	if err != nil {
		return nil, err
	}
	defer decoder.Close()

	decompressedData, err := ioutil.ReadAll(decoder)
	if err != nil {
		return nil, err
	}

	return decompressedData, nil
}

// isValidIP 验证 IP 地址是否有效
func IsValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}
