# Tianli-cdn-server 恭喜你发现屎山！！！
调用API文档：https://console-docs.apipost.cn/preview/877a53de056aef04/6f7d9d05f50db9e6

## 部署

1. 确保安装redis

2. 确保启用NSFW-api

   ```bash
   docker run -p 6012:3000 ghcr.io/arnidan/nsfw-api:latest
   ```

   

3. 前往release下载对应架构二进制文件

4. 运行可执行文件并配置保活进程，首次启动会自动创建`.env`配置文件，注意自行修改。

5. 配置保活进程，使程序运行在后台，Linux可使用例如`screen`

6. 程序会运行在`5012`端口，使用Nginx反向代理5012端口

## 文件清单（运行时程序自动创建）

1. `.env`：配置文件
2. `blacklist.json`：黑名单信息
3. `thesaurus.txt`：base64编码后的黑名单词库，主要为摄政词库

## 默认返回

1.  `/` 目录将会返回程序运行目录下的source/index.html，需自行配置并修改，包括`/index.js` `/index.css`
2.  `/favicon.ico` 图标，需放置在程序运行目录下
3.  `/static` 目录下文件将对应服务端运行目录。

## `.env`配置说明

| 配置项                  | 示例                      | 说明                      |
| ----------------------- | ------------------------- | ------------------------- |
| API_KEY                 | 114514s                   | 配置API密钥，用于API鉴权  |
| ENABLE_KEYWORD_CHECKING | true                      | 是否启用关键词检测        |
| ENABLE_NSFW_CHECKING    | true                      | 是否启用图片违禁检测      |
| JSDELIVR_PREFIX         | https://cdn.jsdelivr.net/ | 代理地址，注意`/`不要遗漏 |
| PORN                    | 0.6                       | 违禁阈值，一般0.6视为违规 |
| NPMMirrow_PREFIX        | https://registry.npmmirror.com/| npm代理地址 |
| GHRaw_PREFIX      | https://raw.githubusercontent.com/| Github raw代理地址 |
| PROXY_MODE    | jsd                      | 镜像模式，填写jsd为jsd镜像，填写local为自取源      |
| EXIPRES    | 6                      | 缓存过期时间      |


## 图片处理

1. 图片压缩，图片可以转webp，需跟随参数`?webp=true`
2. 图片主色调获取，需跟随参数`?get=theme`
   ```json
   {
	"theme": "#eaeaed"
   }
   ```
3. 图片高度宽度获取，需跟随参数`?get=size` 
   ```json
   {
	"width": 1660,
	"height": 302
   }
   ```


## 黑名单信息

blacklist.json用于存放黑名单信息，与二进制文件同目录，且在redis数据库中也加载有此json内容。

blacklist.json优先级高于redis存储内容。

可以通过修改json文件或者使用API增加黑名单信息。

当某一文件触发关键词后，会自动将信息加载进入黑名单并返回403。

当图片触发图片识别时，会自动将信息加载进入黑名单并返回403。
