package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
