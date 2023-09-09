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
	"sync"
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

type AdvanceConfig struct {
	PathList []struct {
		Paths []string `json:"paths"`
		URL   []string `json:"url"`
	} `json:"pathlist"`
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
	} else if proxyMode == "advance" {
		// 调用advance模式处理函数
		advanceResponse, err := makeAdvanceRequest(pathAll)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "请求失败", "hitokoto": hitokoto()})
			// 打印错误信息
			fmt.Println(err)
			return
		}
		contentTypes = advanceResponse.ContentType
		body = advanceResponse.Body
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "请求失败", "hitokoto": hitokoto()})
		// 打印错误信息
		fmt.Println("未知的代理模式")
		return
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
				blacklist.PathList = append(blacklist.PathList, PathItem{Paths: []string{"/" + pathAll}, Reason: "内容包含违规关键词"})

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
				blacklist.PathList = append(blacklist.PathList, PathItem{Paths: []string{"/" + pathAll}, Reason: "涩图，封禁"})

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

func makeAdvanceRequest(pathAll string) (*HTTPResponse, error) {

	fmt.Println("请求路径：" + pathAll)

	go loadAdvance()
	advanceData, err := redisClient.Get("advance.json").Bytes()
	if err != nil {
		return nil, fmt.Errorf("读取advance.json文件失败: %v", err)
	}

	var advanceConfig AdvanceConfig
	err = json.Unmarshal(advanceData, &advanceConfig)
	if err != nil {
		return nil, fmt.Errorf("解析advance.json数据失败: %v", err)
	}

	// 遍历配置列表，查找匹配的路径
	for _, config := range advanceConfig.PathList {
		for _, pathPattern := range config.Paths {
			match, err := regexp.MatchString(pathPattern, pathAll)
			if err != nil {
				return nil, fmt.Errorf("正则表达式匹配失败: %v", err)
			}
			pathAll = strings.Replace(pathAll, pathPattern, "", 1)
			if match {
				// 并发请求多个URL，返回最快的响应
				responses := make(chan *HTTPResponse, len(config.URL))
				errors := make(chan error, len(config.URL))
				var wg sync.WaitGroup

				for _, url := range config.URL {
					wg.Add(1)
					go func(url string) {
						defer wg.Done()
						httpResponse, err := makeRequest(url + pathAll)
						if err != nil {
							errors <- fmt.Errorf("请求失败: %v", err)
						} else {
							responses <- httpResponse
						}
					}(url)
				}

				// 等待所有请求完成
				go func() {
					wg.Wait()
					close(responses)
					close(errors)
				}()

				for {
					select {
					case <-responses:
						// 有请求完成，直接返回
						return <-responses, nil
					case err := <-errors:
						// 请求出错，继续等待其他请求完成
						fmt.Println(err)
					}
				}

			}
		}
	}

	return nil, fmt.Errorf("未找到匹配的路径")
}

func makeRequest(url string) (*HTTPResponse, error) {
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

// 判断是否为图片格式
func isImage(contentType string) bool {
	return strings.Contains(contentType, "image")
}

// 读取本地advance.json文件同步到Redis
func loadAdvance() {
	advanceData, err := os.ReadFile("advance.json")
	if err != nil {
		fmt.Println("无法读取advance.json文件:", err)
		return
	}

	redisClient.Set("advance.json", string(advanceData), 0)
}

func getAdvance(c *gin.Context) {
	advanceData, err := redisClient.Get("advance.json").Bytes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取advance.json文件失败", "hitokoto": hitokoto()})
		return
	}

	var advanceConfig AdvanceConfig
	err = json.Unmarshal(advanceData, &advanceConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "解析advance.json数据失败", "hitokoto": hitokoto()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": advanceConfig})
}

func setAdvance(c *gin.Context) {
	var advanceConfig AdvanceConfig
	if c.Query("key") != apiKey {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的API密钥"})
		return
	}

	err := c.ShouldBindJSON(&advanceConfig)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "解析JSON数据失败", "hitokoto": hitokoto()})
		return
	}

	advanceData, err := json.Marshal(advanceConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "解析JSON数据失败", "hitokoto": hitokoto()})
		return
	}

	err = os.WriteFile("advance.json", advanceData, 0644)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入advance.json文件失败", "hitokoto": hitokoto()})
		return
	}

	redisClient.Set("advance.json", string(advanceData), 0)

	c.JSON(http.StatusOK, gin.H{"data": "设置成功"})
}
