package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Blacklist struct {
	PathList  []PathItem  `json:"pathlist"`
	ReferList []ReferItem `json:"referlist"`
}

type PathItem struct {
	Paths  []string `json:"paths"`
	Reason string   `json:"reason"`
}

type ReferItem struct {
	Refer  string `json:"refer"`
	Reason string `json:"reason"`
}

func syncBlacklistToDB() {
	blacklistData, _ := json.Marshal(blacklist)
	redisClient.Set("blacklist", string(blacklistData), 0)
}

func isPathBlacklisted(path string) bool {
	for _, item := range blacklist.PathList {
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

func isRefererBlacklisted(referer string) bool {
	for _, item := range blacklist.ReferList {
		if strings.Contains(referer, item.Refer) {
			return true
		}
	}
	return false
}

func getBlacklist(c *gin.Context) {
	c.JSON(http.StatusOK, blacklist)
}

func updatePathBlacklist(c *gin.Context) {
	if c.Query("key") != apiKey {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的API密钥"})
		return
	}

	var pathItem PathItem
	if err := c.ShouldBindJSON(&pathItem); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	blacklist.PathList = append(blacklist.PathList, pathItem)

	syncBlacklistToDB()

	blacklistData, _ := json.Marshal(blacklist)
	os.WriteFile("blacklist.json", blacklistData, 0644)

	c.JSON(http.StatusOK, gin.H{"message": "路径黑名单已更新"})
}

func updateReferBlacklist(c *gin.Context) {
	if c.Query("key") != apiKey {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的API密钥"})
		return
	}

	var referItem ReferItem
	if err := c.ShouldBindJSON(&referItem); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	blacklist.ReferList = append(blacklist.ReferList, referItem)

	syncBlacklistToDB()

	blacklistData, _ := json.Marshal(blacklist)
	os.WriteFile("blacklist.json", blacklistData, 0644)

	c.JSON(http.StatusOK, gin.H{"message": "Referer黑名单已更新"})
}

func loadBlacklist() {

	data, err := os.ReadFile("blacklist.json")
	if err != nil {
		fmt.Println("无法加载黑名单数据:", err)
		return
	}

	if err := json.Unmarshal(data, &blacklist); err != nil {
		fmt.Println("无法解析黑名单数据:", err)
		return
	}

	syncBlacklistToDB()

	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for range ticker.C {
			syncBlacklistToDB()
		}
	}()
}
