package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

type Whitelist struct {
	PathList          []PathItem  `json:"pathlist"`
	ReferList         []ReferItem `json:"referlist"`
	AllowEmptyReferer bool        `json:"allowEmptyReferer,omitempty"`
}

func isPathWhitelisted(path string) bool {
	for _, item := range whitelist.PathList {
		for _, p := range item.Paths {
			match, err := regexp.MatchString(p, path)
			if err != nil {
				fmt.Printf("正则匹配错误：%s", err)
				continue
			}
			if match {
				return true
			}
		}
	}
	return false
}

func isRefererWhitelisted(referer string) bool {
	fmt.Printf("Checking referer: %s\n", referer) // 打印传入的 referer
	if whitelist.AllowEmptyReferer && referer == "" {
		return true
	}
	
	for _, item := range whitelist.ReferList {
		fmt.Printf("Against whitelist item: %s\n", item.Refer)
		if strings.Contains(referer, item.Refer) {
			return true
		}
	}
	return false
}

func getWhitelist(c *gin.Context) {
	c.JSON(http.StatusOK, whitelist)
}

func updatePathWhitelist(c *gin.Context) {
	// 检查API密钥
	if c.Query("key") != apiKey {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的API密钥"})
		return
	}

	// 解析请求数据
	var pathItem PathItem
	if err := c.ShouldBindJSON(&pathItem); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	// 更新白名单数据
	whitelist.PathList = append(whitelist.PathList, pathItem)

	// 将白名单数据存储到Redis
	syncWhitelistToDB()

	// 将白名单数据存储到whitelist.json文件
	whitelistData, _ := json.Marshal(whitelist)
	os.WriteFile("whitelist.json", whitelistData, 0644)

	c.JSON(http.StatusOK, gin.H{"message": "路径白名单已更新"})
}

func updateReferWhitelist(c *gin.Context) {
	// 检查API密钥
	if c.Query("key") != apiKey {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的API密钥"})
		return
	}

	// 解析请求数据
	var referItem ReferItem
	if err := c.ShouldBindJSON(&referItem); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	// 更新白名单数据
	whitelist.ReferList = append(whitelist.ReferList, referItem)

	// 将白名单数据存储到Redis
	syncWhitelistToDB()

	// 将白名单数据存储到whitelist.json文件
	whitelistData, _ := json.Marshal(whitelist)
	os.WriteFile("whitelist.json", whitelistData, 0644)

	c.JSON(http.StatusOK, gin.H{"message": "Referer白名单已更新"})
}

func syncWhitelistToDB() {
	// 将白名单数据存储到Redis
	whitelistData, _ := json.Marshal(whitelist)
	redisClient.Set("whitelist", string(whitelistData), 0)
}

func loadWhitelist() {
	// 从JSON文件加载白名单数据
	data, err := os.ReadFile("whitelist.json")
	if err != nil {
		fmt.Println("无法加载白名单数据:", err)
		return
	}

	if err := json.Unmarshal(data, &whitelist); err != nil {
		fmt.Println("无法解析白名单数据:", err)
		return
	}

	// 将白名单数据存储到Redis
	syncWhitelistToDB()
}
