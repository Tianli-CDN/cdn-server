# Tianli-cdn-server (静态资源缓存服务端)

## 恭喜你发现屎山！！！

## 调用API文档：[API](https://console-docs.apipost.cn/preview/877a53de056aef04/6f7d9d05f50db9e6)

注意：库包含CGO，不支持交叉编译（MAC OS除外，但需要使用交叉编译链），同时尽量不要使用linux编译，部分系统可能会缺glibc，如需自行编译请参考github action配置。

此项目为新手练手项目，欢迎各位大佬PR 批评指正。

## docker部署

### 带redis环境

适用于无redis的服务器

### 不携带redis环境

适用于主机包含redis的服务器

1. 拉取镜像

   ```shell
   
   ```

   

2. 1. 在你所需的文件目录新建`.env`文件，注意参考仓库内`.env`配置
   2. 创建`/source/index.html` `/source/index.js` `/source/index.css`
   3. 如果您有需要，请参考仓库内并配置`whitelist.json` `advance.json` `blacklist.json`

3. 运行docker容器，注意将`/yourpath/`替换为你的文件目录。

   ```shell
   docker run -d --network=host -p 5012:5012 -v /yourpath/:/app/ tianlicdn-image
   ```

   

## 二进制 部署

1. 确保安装redis

2. 可选启用NSFW-api

   ```bash
   docker run -p 6012:3000 ghcr.io/arnidan/nsfw-api:latest
   ```

   

3. 前往release下载对应架构二进制文件

4. 运行可执行文件并配置保活进程，首次启动会自动创建`.env`配置文件，注意自行修改。

5. 配置保活进程，使程序运行在后台，Linux可使用例如`screen`

6. 程序会运行在`5012`端口，使用Nginx反向代理5012端口

## 文件清单

1. `.env`：配置文件（请参考仓库配置）
2. `blacklist.json`：黑名单信息（请参考仓库配置）
3. `thesaurus.txt`：base64编码后的黑名单词库，主要为摄政词库
4. `whitelist.json`：白名单信息（请参考仓库配置）
5. `advance.json`：高级缓存配置项（请参考仓库配置）

## 默认返回

1.  `/` 目录将会返回程序运行目录下的source/index.html，需自行配置并修改，包括`/index.js` `/index.css`。
2.  `/favicon.ico` 图标，需放置在程序运行目录下。
3.  `/static` 目录下文件将对应服务端运行目录。

## `.env`配置说明

| 配置项                  | 示例                      | 说明                      |
| ----------------------- | ------------------------- | ------------------------- |
| API_KEY                 | 114514s                   | 配置API密钥，用于API鉴权，建议复杂 |
| ENABLE_KEYWORD_CHECKING | true                      | 是否启用关键词检测        |
| ENABLE_NSFW_CHECKING    | true                      | 是否启用图片违禁检测      |
| JSDELIVR_PREFIX         | https://cdn.jsdelivr.net/ | 代理地址，注意`/`不要遗漏 |
| PORN                    | 0.6                       | 违禁阈值，一般0.6视为违规 |
| NPMMirrow_PREFIX        | https://registry.npmmirror.com/| npm代理地址 |
| GHRaw_PREFIX      | https://raw.githubusercontent.com/| Github raw代理地址 |
| PROXY_MODE    | jsd                      | 镜像模式，填写jsd为jsd镜像，填写local为自取源，填写advance为高级配置，需修改advance.json配置项。支持多网关并发请求，服务端会返回最快响应。且支持自行配置更多缓存内容。 |
| EXIPRES    | 6                      | 缓存过期时间      |
| REDIS_ADDR      | localhost:6379 | redis服务器地址及端口 |
| REDIS_PASSWORD    | 114514                     | redis密码，可以为空      |
| REDIS_DB    | 5                      | redis使用数据库名，int，确保没有冲突再填写      |
| RUN_MODE    | blacklist                      | 运行模式，可选blacklist or whitelist，运行模式为白名单或黑名单，白名单时将以白名单内内容做为校验，同时黑路径黑名单也会生效，黑名单与白名单参考blacklist.json和whitelist.json     |
| REJECTION_METHOD | 403 | 拒绝方式：301或403，当填写301时还需要自行配置301_URL（比如该referer或者path不在白名单中或者处于黑名单中，将会以你设置的其中一种状态码作为处理） |
| 301_URL | https://cdn.jsdelivr.net/ | 当REJECTION_METHOD=301时，将会把非白名单请求重定向至配置的url |

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



## 高级配置

高级配置时不止可以缓存jsd资源，可以自行配置更多静态资源缓存
