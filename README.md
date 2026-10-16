# Go Template

Go + Gin + GORM 服务模板。默认无需中间件即可运行，可选 MySQL / PostgreSQL、Redis、RocketMQ Producer。使用显式依赖注入、Uber Zap（`go.uber.org/zap`）JSON 日志、配置校验、请求 ID、健康检查与优雅退出。

## 快速开始

需要 Go 1.25+；容器方式需要 Docker Compose v2。

```sh
go mod tidy # 下载依赖并生成 go.sum，请提交 go.mod 和 go.sum
make run
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

`make build` 输出 `bin/server`；`make fmt` 格式化；`make check` 运行 vet、race 测试、覆盖率统计和格式检查。GitHub Actions 执行相同检查。项目未设置覆盖率百分比门槛。

## 目录与规范

```text
cmd/server/         程序入口、依赖组装、信号与退出管理
internal/config/    Viper 配置加载与校验
internal/platform/  GORM、Redis、RocketMQ 资源管理与日志
internal/httpapi/   Gin 路由、中间件、输入验证与 HTTP 测试
internal/user/      用户业务、Repository 接口与 GORM 实现
configs/           YAML 默认配置与 RocketMQ Broker 配置
migrations/        按数据库区分的显式 SQL 迁移
```

HTTP handler → service → repository。传递 `context.Context`，避免全局客户端；数据库操作使用 `WithContext`。新增业务按领域组织，接口由使用方定义，错误向上返回。使用 gofmt（Tab 缩进）、小写包名、导出符号文档、`*_test.go` 表驱动测试；不要为了目录而添加空抽象层。

## 配置与数据库

默认读取 `configs/config.yaml`；可通过 `go run ./cmd/server -config path/to/config.yaml` 指定，`-config ''` 仅使用默认值与环境变量。环境变量优先，如 `APP_LOG_LEVEL=debug`、`APP_SERVER_ADDRESS=:9000`。`.env` 仅供 Compose 插值，本地运行请显式 export。配置不记录密码或 DSN。

```sh
docker compose --profile postgres up -d --wait postgres
docker compose exec -T postgres psql -U app -d app < migrations/001_users.postgres.sql
APP_DATABASE_DRIVER=postgres APP_DATABASE_DSN='host=localhost user=app password=local-app-only dbname=app port=5432 sslmode=disable' make run
# MySQL 替代方案：
docker compose --profile mysql up -d --wait mysql
docker compose exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_PASSWORD" mysql -u app app' < migrations/001_users.mysql.sql
APP_DATABASE_DRIVER=mysql APP_DATABASE_DSN='app:local-app-only@tcp(localhost:3306)/app?charset=utf8mb4&parseTime=True&loc=UTC&timeout=5s' make run
```

每个数据库只执行对应迁移一次；模板不在启动时 AutoMigrate。生产环境应使用版本化迁移工具、专用迁移权限，并为 DSN 启用 TLS 和连接超时。

```sh
curl -X POST localhost:8080/api/v1/users -H 'Content-Type: application/json' -d '{"name":"Ada"}'
```

未启用数据库时该接口返回 503；请求失败不泄漏底层存储错误。示例接口无认证，公开部署前按业务增加身份认证和访问控制。

## Redis 与 RocketMQ

```sh
docker compose --profile redis up -d --wait redis
APP_REDIS_ENABLED=true make run
docker compose --profile rocketmq up -d namesrv broker
```

RocketMQ 默认供容器内客户端使用；本机运行前，将 `configs/broker.conf` 的 `brokerIP1` 改为 `127.0.0.1` 并重启 broker，然后设置 `APP_ROCKETMQ_ENABLED=true`。Broker 启动可能需要等待，再启动应用。客户端使用 Apache Go v2 原生协议，通过 `Resources.Producer.SendSync(ctx, primitive.NewMessage(topic, body))` 发送；按业务注入使用，不开放演示发送接口。消费者、重试幂等与死信处理由业务补充。

`/healthz` 仅判断进程存活，`/readyz` 检查已启用的数据库和 Redis；RocketMQ readiness 不做消息发送探测，Producer 启动成功不保证 Broker 可写。

## Docker

```sh
docker compose up -d --build app # 无中间件模式
# 使用容器数据库时，先启动并迁移对应服务，再启动 app：
APP_DATABASE_DRIVER=postgres APP_DATABASE_DSN='host=postgres user=app password=local-app-only dbname=app sslmode=disable' docker compose up -d --build app
```

Compose profile 只启动中间件，不自动启用应用客户端。容器地址使用服务名 `postgres`、`mysql`、`redis`、`namesrv`。应用以非 root 运行，SIGTERM 后等待 HTTP 请求结束再关闭资源。Zap JSON 日志写 stdout，由运行平台采集轮转；支持 `APP_LOG_LEVEL` 配置，记录启动、退出、资源初始化/清理及 HTTP 请求完成事件，包含请求 ID、状态码和耗时。每次修改必须更新关键注释与完成日志，详见 AGENTS.md。Compose 密码和自动建 Topic 仅供本地开发；生产使用密钥管理和持久化备份。

## Git 初始化

在项目根目录运行 `sh scripts/init-git.sh`，初始化 `main` 并配置 `origin` 为 `git@github.com:calm195/go-template.git`。脚本不提交或推送代码。

## 贡献与 Codex

请遵循 [AGENTS.md](AGENTS.md)。提交前执行 `make fmt && make check && make build`。依赖变化提交 `go.mod`、`go.sum`；PR 说明行为变化、验证结果和迁移影响。

参考：[Gin 优雅退出](https://github.com/gin-gonic/gin/blob/master/docs/doc.md)、[GORM 连接配置](https://gorm.io/docs/connecting_to_the_database.html)、[Apache RocketMQ Go 客户端](https://github.com/apache/rocketmq-client-go)。
