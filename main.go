package main

import (
	"embed"
	"fmt"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

var (
	redisClient *redis.Client
	blacklist   Blacklist
	whitelist   Whitelist
)

var (
	//go:embed thesaurus.txt
	thesaurusData embed.FS
)

func main() {

	loadconfig()

	// 初始化Redis客户端
	redisClient = redis.NewClient(&redis.Options{
		Addr:     Redis_addr,
		Password: Redis_password,
		DB:       Redis_DB_int, // 使用DB5作为缓存数据库
	})
	loadWhitelist()
	loadBlacklist()
	loadAdvance()

	_, err := redisClient.Ping().Result()
	if err != nil {
		fmt.Println("无法连接到Redis:", err)
		return
	}
	router := gin.Default()
	gin.SetMode(gin.ReleaseMode)

	// 添加CORS中间件
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	router.Use(cors.New(config))

	// 设置API路由组
	api := router.Group("/api")
	{
		api.GET("/blacklist", getBlacklist)
		api.POST("/blacklist/update_path", updatePathBlacklist)
		api.POST("/blacklist/update_refer", updateReferBlacklist)
		api.GET("/whitelist", getWhitelist)
		api.POST("/whitelist/update_path", updatePathWhitelist)
		api.POST("/whitelist/update_refer", updateReferWhitelist)
		api.POST("/clear_cache", clearCache)
		api.POST("/clear_all_cache", clearAllcache)
		api.GET("/get_advance", getAdvance)
		api.POST("/set_advance", setAdvance)
		api.GET("/cache_info", getCacheInfo)
	}

	// 设置请求处理函数
	router.Any("/:path/*filepath", handleRequest)

	router.Static("/static", "./static")
	router.StaticFile("/favicon.ico", "./favicon.ico")
	router.StaticFile("/", "./source/index.html")
	router.StaticFile("/index.js", "./source/index.js")
	router.StaticFile("/index.css", "./source/index.css")
	router.Run(":5012")
}
