package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type NSFWResponse struct {
	Porn    float64 `json:"porn"`
	Sexy    float64 `json:"sexy"`
	Hentai  float64 `json:"hentai"`
	Neutral float64 `json:"neutral"`
	Drawing float64 `json:"drawing"`
}

type HTTPResponse struct {
	Body        []byte
	ContentType string
}

var (
	contentTypes string
	body         []byte
	isAPI        bool
)

func handleRequest(c *gin.Context) {
	// 获取请求路径和Referer
	path := c.Param("path")
	referer := c.Request.Referer()
	pathAll := fmt.Sprintf("%s%s", path, c.Param("filepath"))
	params := c.Request.URL.RawQuery
	urlAll := c.Request.URL.String()
	fmt.Println("参数：" + params)

	if isBlacklistMode() {
		// 检查路径黑名单
		if isPathBlacklisted("/" + pathAll) {
			c.JSON(http.StatusForbidden, gin.H{"error": "路径被禁止访问", "hitokoto": hitokoto()})
			return
		}

		// 检查Referer黑名单
		if isRefererBlacklisted(referer) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Referer被禁止访问", "hitokoto": hitokoto()})
			return
		}
	} else {
		// 检查路径黑名单
		if isPathBlacklisted("/" + pathAll) {
			c.JSON(http.StatusForbidden, gin.H{"error": "路径被禁止访问", "hitokoto": hitokoto()})
			return
		}
		// 检查路径白名单
		if !isPathWhitelisted("/" + pathAll) {
			c.JSON(http.StatusForbidden, gin.H{"error": "路径未被授权访问", "hitokoto": hitokoto()})
			return
		}

		// 检查Referer白名单
		if !isRefererWhitelisted(referer) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Referer未被授权访问", "hitokoto": hitokoto()})
			return
		}
	}
	// 检查是否存在缓存
	if data, err := redisClient.Get(urlAll).Result(); err == nil {
		// 缓存存在，直接返回数据
		contentType, _ := redisClient.Get(urlAll + ":content-type").Result()

		if strings.Contains(contentType, "image") || strings.Contains(contentType, "font") {
			c.Header("Cache-Control", "max-age=315360000")
			c.Header("Expires", time.Now().Add(315360000*time.Second).Format(http.TimeFormat))
		} else {
			ttl, _ := redisClient.TTL(urlAll).Result()
			c.Header("Cache-Control", fmt.Sprintf("max-age=%d", int(ttl.Seconds())))
			c.Header("Expires", time.Now().Add(ttl).Format(http.TimeFormat))
		}

		c.Data(http.StatusOK, contentType, []byte(data))
		return
	}

	// 检查当前模式是否为代理模式
	if proxyMode == "jsd" {
		// 调用http请求函数
		httpResponse, err := makeJSDRequest(pathAll)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "请求失败", "hitokoto": hitokoto()})
			return
		}
		contentTypes = httpResponse.ContentType
		body = httpResponse.Body
	} else if proxyMode == "local" {
		// 调用http请求函数
		httpResponse, err := makeLocalRequest(pathAll)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "请求失败", "hitokoto": hitokoto()})
			return
		}
		contentTypes = httpResponse.ContentType
		body = httpResponse.Body
	}

	if params != "" && isImage(contentTypes) {
		isAPI = true
		if params == "webp=true" {
			fmt.Println("转换webp:", pathAll)
			// 转webp
			webpData, err := convertImageToWebp(body)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "图片转换失败", "hitokoto": hitokoto()})
				return
			}
			body = webpData
		} else {
			// 调用图片处理函数
			fmt.Println("处理图片:", pathAll, "参数:", params)
			imageData, contentType := handleImageRequest(body, params)
			if imageData != nil {
				body = imageData
				contentTypes = contentType
			}
		}
	}

	// 检查是否需要进行词库匹配，异步
	if !isAPI && (strings.Contains(contentTypes, "text/html") || strings.Contains(contentTypes, "text/plain") || strings.Contains(contentTypes, "application/json")) {
		go func() {
			if checkKeywords(string(body)) {
				// 更新黑名单数据
				blacklist.PathList = append(blacklist.PathList, PathItem{Path: "/" + pathAll, Reason: "内容包含违规关键词"})

				// 将黑名单数据存储到Redis
				syncBlacklistToDB()

				// 将黑名单数据存储到blacklist.json文件
				blacklistData, _ := json.Marshal(blacklist)
				os.WriteFile("blacklist.json", blacklistData, 0644)
			}
		}()
	}

	// 检查是否需要进行图片 NSFW 检查，异步，当文件类型为png jpg jpeg webp触发
	if !isAPI && (strings.Contains(contentTypes, "image/png") || strings.Contains(contentTypes, "image/jpg") || strings.Contains(contentTypes, "image/jpeg") || strings.Contains(contentTypes, "image/webp")) {
		fmt.Println("检查图片 NSFW:", pathAll)
		go func() {

			nsfwResult, err := detectNSFW(body, pathAll)
			if err != nil {
				fmt.Println("检查图片 NSFW 失败:", err)
				return
			}

			if nsfwResult.NSFW {
				// 更新黑名单数据
				blacklist.PathList = append(blacklist.PathList, PathItem{Path: "/" + pathAll, Reason: "涩图，封禁"})

				// 将黑名单数据存储到Redis
				syncBlacklistToDB()

				// 将黑名单数据存储到blacklist.json文件
				blacklistData, _ := json.Marshal(blacklist)
				os.WriteFile("blacklist.json", blacklistData, 0644)
			}
		}()
	}

	// 异步存储到Redis
	go func() {
		// 设置缓存时间,通过读取配置项EXPIRES的值来设置缓存时间
		expiresTime, err := strconv.Atoi(expiresTimeStr)
		if err != nil {
			expiresTime = 6
		}

		cacheTime := time.Duration(expiresTime) * time.Hour
		if strings.Contains(urlAll, "@") {
			cacheTime = 7 * 24 * time.Hour
		}
		redisClient.Set(urlAll, string(body), cacheTime)
		redisClient.Set(urlAll+":content-type", contentTypes, cacheTime)
	}()

	// 返回响应内容
	c.Data(http.StatusOK, contentTypes, body)
}

func makeJSDRequest(pathAll string) (*HTTPResponse, error) {

	url := fmt.Sprintf(jsdelivrPrefix+"%s", pathAll)
	fmt.Println("源请求URL：" + url)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36 Edg/116.0.1938.5412")
	req.Header.Set("Referer", "https://baidu.com")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
	} else if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")

	response := &HTTPResponse{
		Body:        body,
		ContentType: contentType,
	}

	return response, nil
}

func makeLocalRequest(pathAll string) (*HTTPResponse, error) {
	firstDir := pathAll[:strings.Index(pathAll, "/")]
	pathAll = pathAll[strings.Index(pathAll, "/")+1:]

	if firstDir == "gh" {
		pack := pathAll[:strings.Index(pathAll[strings.Index(pathAll, "/")+1:], "/")+1]
		file := pathAll[strings.Index(pathAll[strings.Index(pathAll, "/")+1:], "/")+1:]
		re := regexp.MustCompile(`@([^/]+)`)
		match := re.FindStringSubmatch(file)
		var version string
		if len(match) > 1 {
			version = match[1]
			file = re.ReplaceAllString(file, "")
		} else {
			version = "main"
		}

		// 拼接URL，源：https://raw.githubusercontent.com/%s/%s/%s
		url := fmt.Sprintf("%s%s/%s%s", ghrawPrefix, pack, version, file)
		fmt.Println("源请求URL：" + url)

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("创建请求失败: %v", err)
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36 Edg/116.0.1938.5412")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("请求失败: %v", err)
		}
		defer resp.Body.Close()

		// 读取响应内容		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("读取响应失败: %v", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
		} else if resp.StatusCode >= http.StatusBadRequest {
			return nil, fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
		}

		contentType := resp.Header.Get("Content-Type")

		response := &HTTPResponse{
			Body:        body,
			ContentType: contentType,
		}

		return response, nil

	} else if firstDir == "npm" {
		packWithVersion := pathAll[:strings.Index(pathAll, "/")]
		pack := packWithVersion[:strings.Index(packWithVersion, "@")]
		version := packWithVersion[strings.Index(packWithVersion, "@")+1:]
		file := pathAll[strings.Index(pathAll, "/")+1:]
		// 拼接URL，源：https://registry.npmmirror.com/%s/%s/files/dist/%s
		url := fmt.Sprintf("%s%s/%s/files/dist/%s", npmPrefix, pack, version, file)
		fmt.Println("源请求URL：" + url)
		// 创建自定义请求
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("创建请求失败: %v", err)
		}

		req.Header.Set("User-Agent", "npm/7.20.6 node/v14.17.6 win32 x64")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("请求失败: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("读取响应失败: %v", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
		} else if resp.StatusCode >= http.StatusBadRequest {
			return nil, fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
		}

		contentType := resp.Header.Get("Content-Type")

		response := &HTTPResponse{
			Body:        body,
			ContentType: contentType,
		}

		return response, nil

	}

	return nil, fmt.Errorf("请求失败，状态码: %d", 404)
}

// 判断是否为图片格式
func isImage(contentType string) bool {
	return strings.Contains(contentType, "image")
}
