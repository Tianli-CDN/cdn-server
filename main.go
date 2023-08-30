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
)

var (
	//go:embed source/thesaurus.txt
	thesaurusData embed.FS
)

func main() {

	loadconfig()

	// 初始化Redis客户端
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       5,
	})

	loadBlacklist()

	_, err := redisClient.Ping().Result()
	if err != nil {
		fmt.Println("无法连接到Redis:", err)
		return
	}
	router := gin.Default()

	// 添加CORS中间件
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	router.Use(cors.New(config))

	// 设置API路由组
	api := router.Group("/api")
	{
		api.GET("/blacklist", getBlacklist)
		api.POST("/update_path", updatePathBlacklist)
		api.POST("/update_refer", updateReferBlacklist)
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
