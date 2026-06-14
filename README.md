# buildwebs233-server (Go 1.26)

## 设计目标
- 默认管理端口 `6640`
- 默认管理员账号 `root/root`
- 配置驱动：`server.yaml`
- 配置文件热重载
- HTML/页面变更可触发重载通知（通过 SSE）
- 具备低代码式页面构建基础能力
- 支持一键安装（PowerShell / Bash）
- 可逐步升级到 Vite 8 前端工程化

## 当前落地版本架构
- `cmd/buildwebs233-server/main.go`
  - 读取并监听 `server.yaml`
  - 提供管理 API 和页面渲染 API
  - 提供 SSE 重载通道 `/api/reload`
- `internal/config/`
  - `Config` 与默认值
  - `Manager` 负责配置热重载监听
- `internal/store/`
  - 文件持久化存储：`data/pages.json`、`data/templates.json`
- `internal/hotreload/`
  - SSE Hub：`/api/reload`
  - 客户端脚本：`/__reload-client.js`
- `internal/server/`
  - 后台接口：登录、页面管理、模板查询
  - 页面预览渲染：`/p/{slug}`
- `ui/`
  - 可选 Vite 8 前端脚手架（低代码编辑器最小化面板）

## 启动方式
### 开发
1. 编写页面编辑前端
   - `cd ui`
   - `npm install`
   - `npm run dev`
2. 启动后端
   - `go run ./cmd/buildwebs233-server -config server.yaml`

### 生产
1. 构建前端：`cd ui && npm run build`
2. 将产物放到 `web/` 目录
3. 构建后端：`go build ./cmd/buildwebs233-server`
4. 启动：`./buildwebs233-server -config server.yaml`

## 安装脚本
- Windows: `scripts/install-buildwebs233-server.ps1`
- Linux/macOS: `scripts/install-buildwebs233-server.sh`

## 自动化与发布
- `CI`: 自动执行工作流校验、`go vet`、`go test ./...`、前端 `typecheck/build`
- `Docs Pages`: 自动将 `docs/` 发布到 GitHub Pages
- `Release`: 推送 `v*` tag 时自动打包 Linux/Windows 预编译文件，供安装脚本直接下载

## 文件说明
- `server.yaml`：启动配置（账号密码、端口、监听路径、热重载路径）
- `internal/config/manager.go`：配置热重载监听入口
- `internal/hotreload/reloader.go`：EventSource 重载总线
- `internal/store/store.go`：页面与模板持久化层
- `internal/server/server.go`：API 路由与渲染页

## 关于“低代码+大量内置模板”的落地路径
1. 先将 `Page.Blocks` 升级为结构化区块（组件 schema + props + 样式）
2. 使用独立内容模型服务（可选 Postgres/Redis）
3. 接入模板市场机制
   - 模板仓库表
   - 模板预览图与分类
   - 用户收藏 / 复制模板
4. 扩展组件运行时（推荐接入 GrapesJS 或独立拖拽编辑器）
5. 用 CDN/OSS 存放模板资源与截图，后端只存 DSL 与元数据

## 扩展到多租户与多地区备案场景
- 多租户可采用：
  - `tenant_id` 维度隔离页面、用户、模板
  - 每租户独立数据库 schema 或独立表
- 地区备案配置：
  - `server.yaml` + 租户配置支持字段：`region`、`icp_info`、`compliance_checklist`
  - 接入行政区域工作流（提审步骤、备案号、审核状态）

## 风险与建议
- 当前版本以文件存储为起点，适合 MVP 与小规模部署
- 规模化后切到 Postgres + object storage（S3/OSS）更稳
- 登录目前是会话 cookie + in-memory token，生产建议接入 JWT/Redis Session
- 配置端口变更会触发 `reload` 事件，但进程仍需重启应用生效
