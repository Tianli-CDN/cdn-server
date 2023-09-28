package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

var (
	apiKey                string
	enableKeywordChecking bool
	enableNSFWChecking    bool
	pornStr               string
	jsdelivrPrefix        string
	expiresTimeStr        string
	proxyMode             string
	ghrawPrefix           string
	npmPrefix             string
	Redis_addr            string
	Redis_password        string
	Redis_DB              string
	Redis_DB_int          int
	RunMode               string
	RejectionMethod       string
	RedirectUrl           string
)

func createConfigFile() {
	file, err := os.Create(".env")
	if err != nil {
		fmt.Println("无法创建.env文件:", err)
		return
	}
	defer file.Close()

	file.WriteString(fmt.Sprintf("API_KEY=%d\n", uuid.New().ID()))
	file.WriteString("ENABLE_KEYWORD_CHECKING=true\n")
	file.WriteString("ENABLE_NSFW_CHECKING=true\n")
	file.WriteString("PORN=0.6\n")
	file.WriteString("JSDELIVR_PREFIX=https://cdn.jsdelivr.net/\n")
	file.WriteString("EXIPRES=6\n")
	file.WriteString("PROXY_MODE=jsd\n")
	file.WriteString("GHRaw_PREFIX=https://raw.githubusercontent.com/\n")
	file.WriteString("NPMMirrow_PREFIX=https://registry.npmmirror.com/\n")
	file.WriteString("REDIS_ADDR=localhost:6379\n")
	file.WriteString("REDIS_PASSWORD=\n")
	file.WriteString("REDIS_DB=5\n")
	file.WriteString("RUN_MODE=blacklist\n")
	file.WriteString("REJECTION_METHOD=403\n")
	file.WriteString("301_URL=https://cdn.jsdelivr.net/\n")
}

func loadconfig() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("无法加载.env文件:", err)
		createConfigFile()
		colorRedBold := "\033[1;31m"
		colorReset := "\033[0m"
		fmt.Println(colorRedBold + "创建默认.env文件成功，请修改配置后重启程序" + colorReset)
		os.Exit(1)
		return
	}
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		fmt.Println("未找到.env文件，创建中...")
		createConfigFile()
		colorRedBold := "\033[1;31m"
		colorReset := "\033[0m"

		fmt.Println(colorRedBold + "创建.env文件成功，请修改配置后重启程序" + colorReset)
		os.Exit(1)
		return
	}
	apiKey = os.Getenv("API_KEY")
	enableKeywordChecking = os.Getenv("ENABLE_KEYWORD_CHECKING") == "true"
	enableNSFWChecking = os.Getenv("ENABLE_NSFW_CHECKING") == "true"
	pornStr = os.Getenv("PORN")
	jsdelivrPrefix = os.Getenv("JSDELIVR_PREFIX")
	expiresTimeStr := os.Getenv("EXIPRES")
	proxyMode = os.Getenv("PROXY_MODE")
	ghrawPrefix = os.Getenv("GHRaw_PREFIX")
	npmPrefix = os.Getenv("NPMMirrow_PREFIX")
	Redis_addr = os.Getenv("REDIS_ADDR")
	if Redis_addr == "" {
		Redis_addr = "localhost:6379"
	}
	Redis_password = os.Getenv("REDIS_PASSWORD")
	Redis_DB = os.Getenv("REDIS_DB")
	Redis_DB_int, err = strconv.Atoi(Redis_DB)
	if err != nil {
		Redis_DB_int = 5
	}
	RunMode = os.Getenv("RUN_MODE")
	RejectionMethod = os.Getenv("REJECTION_METHOD")
	RedirectUrl = os.Getenv("301_URL")

	if apiKey == "" || pornStr == "" || jsdelivrPrefix == "" || expiresTimeStr == "" || proxyMode == "" || Redis_addr == "" || Redis_DB == "" || RunMode == "" {
		fmt.Println("配置文件错误，已重置为默认配置")
		createConfigFile()
		os.Exit(1)
		return
	}

	fmt.Println("API密钥:", apiKey)
	fmt.Println("是否启用关键词检查:", enableKeywordChecking)
	fmt.Println("是否启用图片 NSFW 检查:", enableNSFWChecking)
	fmt.Println("涩图阈值:", pornStr)
	fmt.Println("jsDelivr镜像地址:", jsdelivrPrefix)
	fmt.Println("缓存时间:", expiresTimeStr)
	if proxyMode == "jsd" {
		fmt.Println("代理模式: jsDelivr")
	} else if proxyMode == "local" {
		fmt.Println("代理模式: 自取源")
		fmt.Println("GitHub Raw镜像地址:", ghrawPrefix)
		fmt.Println("NPM镜像地址:", npmPrefix)
	} else if proxyMode == "advance" {
		fmt.Println("代理模式: 高级")
	} else {
		fmt.Println("代理模式: 未知")
	}
	fmt.Println("Redis地址:", Redis_addr)
	fmt.Println("Redis密码:", Redis_password)
	fmt.Println("Redis数据库:", Redis_DB)
	fmt.Println("运行模式:", RunMode)
	fmt.Println("拒绝方式:", RejectionMethod)
	if RejectionMethod == "301" {
		fmt.Println("301跳转地址:", RedirectUrl)
	}
}
