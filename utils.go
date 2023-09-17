package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func hitokoto() string {
	// 请求一言接口
	resp, err := http.Get("https://v1.hitokoto.cn/")
	if err != nil {
		return "一言接口请求失败"
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "一言接口响应读取失败"
	}

	// 解析JSON
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return "一言接口响应解析失败"
	}

	// 返回一言
	return data["hitokoto"].(string)
}

func checkKeywords(content string) bool {
	if !enableKeywordChecking {
		return false
	}

	thesaurusBytes, err := thesaurusData.ReadFile("thesaurus.txt")
	if err != nil {
		fmt.Println("无法读取词库文件:", err)
		return false
	}

	decodedThesaurusBytes, err := base64.StdEncoding.DecodeString(string(thesaurusBytes))
	if err != nil {
		fmt.Println("无法解密词库文件:", err)
		return false
	}

	thesaurus := strings.Split(string(decodedThesaurusBytes), "\n")

	for _, word := range thesaurus {
		if strings.Contains(content, word) {
			fmt.Println("匹配到关键词:", word)
			return true
		}
	}

	return false
}

func isBlacklistMode() bool {
	return RunMode == "blacklist"
}

func clearCache(c *gin.Context) {

	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的路径"})
		return
	}

	redisClient.Del(path)
	go redisClient.Del(path + ":content-type")
	go loadWhitelist()
	go loadBlacklist()

	c.JSON(http.StatusOK, gin.H{"message": "缓存已清除"})
}

func clearAllcache(c *gin.Context) {

	if c.Query("key") != apiKey {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的API密钥"})
		return
	}

	redisClient.FlushDB()

	go loadWhitelist()
	go loadBlacklist()

	c.JSON(http.StatusOK, gin.H{"message": "所有缓存已清除"})
}

func getCacheInfo(c *gin.Context) {
	keys, err := redisClient.Keys("*:content-type").Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法获取缓存信息"})
		return
	}

	cacheInfo := make(map[string]int)
	for _, key := range keys {
		contentType := strings.Split(redisClient.Get(key).Val(), ";")[0]
		cacheInfo[contentType]++
	}

	// 获取数据库大小
	dbSize := redisClient.DBSize().Val() / 1024 / 1024.0

	c.JSON(http.StatusOK, gin.H{"cache_info": cacheInfo, "db_size": dbSize})
}
