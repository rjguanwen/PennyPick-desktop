# 拾财 PennyPick（桌面版）

个人记账桌面应用（Windows）：轻松记下每一笔消费，按月设置预算预警，多维度统计分析，支持账单导出与还款管理。由 Wails（Go + WebView2）打包为单文件桌面程序，前端与后端逻辑与 Web 版（`pennypick/`）保持一致。

## 功能简介

1. **快速记账**：大数字键盘输入、常用分类置顶、支持「保存后继续记」，两三秒完成一笔；收入/支出一键切换。
2. **月度预算预警**：支持**总预算**与**分类预算**两档（分类预算如「餐饮每月 800 元」），均按月份设置金额与预警阈值（如 80%），达到阈值或超支时在预算页、首页、记账页及时提醒。
3. **多维度统计**：分类占比饼图、按日/按月收支趋势、账户收支分布、月度概览（支出/收入/结余/日均/笔数），以及**标签统计**。
4. **账单标签**：自定义标签库，每条账单可打多个标签（上限 8 个），支持按标签筛选、按标签统计，导出 CSV 含标签列。
5. **还款管理**：账户可标记为「先用后还」（信用卡/花呗等）并设置每月还款日；还款页按月标记已还款；**本月有支出的账户**在超过还款日仍未标记时提醒（首页同步提醒），无支出账户标注「本月无支出」。
6. **账单管理**：按月份、类型、分类、账户、标签、关键字筛选账单；按日分组展示；支持编辑与删除。
7. **账单导出**：桌面端通过系统「另存为」对话框导出 CSV（UTF-8 BOM，Excel 可直接打开），支持时间范围与类型筛选。
8. **分类与账户**：预置常用收支分类与账户，可自定义分类名称、图标、颜色；账户支持设置是否「先用后还」及每月还款日。
9. **多用户**：支持注册多个账号，每个用户数据相互隔离（默认账号 `admin / admin123`）。

## 技术栈

- 桌面壳：Wails v2（Go + WebView2 Runtime），单文件打包
- 前端：Vue 3 + Element Plus + Vite + Pinia + Vue Router + ECharts
- 后端：Go（Gin + GORM），数据库 SQLite（内嵌本机服务，监听 `127.0.0.1` 随机端口）
- 认证：JWT（Bearer Token，有效期 30 天）

## 目录结构

```
pennypick-desktop/
├── main.go            # Wails 入口：内嵌 Gin HTTP（随机端口）+ 桌面窗口
├── app.go             # Wails 绑定：GetAPIBaseURL / ExportBills
├── internal/          # 业务逻辑（与 Web 版 backend-go/internal 保持同步）
│   ├── config/        # 配置（环境变量）
│   ├── database/      # SQLite 连接、迁移、默认数据初始化
│   ├── model/         # 数据模型（用户/分类/账户/账单/标签/预算/还款）
│   ├── middleware/    # JWT 认证、CORS
│   └── handler/       # 业务处理（认证/分类/账户/账单/标签/预算/统计/还款/导出）
├── frontend/          # Vue 前端（构建后由 Wails 嵌入 exe）
│   └── src/
│       ├── api/       # axios 封装
│       ├── router/    # 路由
│       ├── stores/    # Pinia
│       ├── utils/     # 格式化工具
│       ├── layout/    # 主布局（侧边栏 + 顶部栏）
│       ├── components/# 通用组件
│       └── views/     # 页面（首页/记账/账单/还款/统计/预算/分类/设置）
├── build/             # Wails 构建配置（应用图标、Windows 资源等）
├── wails.json         # Wails 构建配置
└── go.mod             # Go 模块（pennypickdesktop）
```

## 运行

### 直接运行（推荐）

已构建产物：`build\bin\pennypick.exe`（约 23MB 单文件），复制到任意 Windows 机器双击即可运行。

依赖：Windows 10/11 + WebView2 Runtime（Win10/11 一般已内置）。

### 开发调试

前置要求：Go 1.23+、Node.js 18+、Wails CLI。

```bash
# 1. 安装 Wails CLI（本机 Go 代理建议 goproxy.cn）
go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2

# 2. 安装前端依赖并进入开发模式（自动打开桌面窗口，前端热更新）
cd frontend && npm install && cd ..
wails dev
```

## 构建打包

```bash
# Windows 64 位可执行文件
wails build -platform windows/amd64 -clean

# 产物：build\bin\pennypick.exe
```

> 注意：构建前请确认没有正在运行的 `pennypick.exe`（文件被占用会导致静默构建失败）。  
> 如需修改应用图标：替换"build/appicon.png"，并执行"wails generate icons --input appicon.png   --windowsfilename windows/icons.ico    --macfilename darwin/icons.icns" 

## 数据存储

| 项目 | 位置 |
| --- | --- |
| 数据库 | `%APPDATA%\PennyPick\pennypick.db`（SQLite） |
| 运行日志 | `%APPDATA%\PennyPick\pennypick.log` |

- 首次启动自动建库建表、创建默认账号（`admin / admin123`）并初始化预置分类/账户。
- 桌面版与 Web 版数据相互独立；如需沿用 Web 版数据，将 Web 版 `pennypick.db` 复制到上述目录即可。

## 数据模型

| 表 | 说明 |
| --- | --- |
| `users` | 用户（个人记账，多用户数据隔离） |
| `categories` | 分类（expense 支出 / income 收入，含图标与颜色） |
| `accounts` | 账户（含 `is_credit` 先用后还标识、`repay_day` 每月还款日） |
| `bills` | 账单（类型、金额、分类、账户、发生时间、备注） |
| `tags` | 标签（标签库，按用户隔离，名称唯一） |
| `bill_tags` | 账单-标签多对多关联 |
| `repayments` | 账户月度还款记录（user+account+month 唯一） |
| `budgets` | 月度总预算（唯一索引 user_id + month，含预警阈值） |
| `category_budgets` | 分类预算（唯一索引 user_id + month + category_id，独立预警） |

## 主要接口

后端为内嵌本机服务（`127.0.0.1` 随机端口），前端通过 Wails 绑定获取地址后以 `/api` 前缀调用。主要接口：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/api/auth/login` | 登录（form-data） |
| POST | `/api/auth/register` | 注册（自动初始化预置分类/账户） |
| GET | `/api/categories` | 分类列表 |
| POST/PATCH/DELETE | `/api/categories[/:id]` | 分类增改删 |
| GET | `/api/accounts` | 账户列表 |
| POST/PATCH/DELETE | `/api/accounts[/:id]` | 账户增改删（支持先用后还/还款日） |
| GET/POST/PATCH/DELETE | `/api/tags[/:id]` | 标签列表/创建（幂等）/改名/删除 |
| GET | `/api/bills` | 账单列表（month/type/category_id/account_id/tag_id/keyword + 分页） |
| POST | `/api/bills` | 记一笔（支持 `tag_ids` 多标签） |
| PATCH/DELETE | `/api/bills/:id` | 账单改/删 |
| GET | `/api/repayments?month=` | 某月各信用账户还款状态（含 `month_expense`/`has_expense`/逾期判定） |
| POST/DELETE | `/api/repayments` | 标记/取消还款 |
| GET/PUT/DELETE | `/api/budgets` | 月度总预算查询/设置/删除 |
| GET/PUT/DELETE | `/api/budgets/categories` `/api/budgets/category` | 分类预算 |
| GET | `/api/stats/overview` | 月度概览 |
| GET | `/api/stats/by-category` | 分类统计 |
| GET | `/api/stats/trend` | 收支趋势 |
| GET | `/api/stats/accounts` | 账户收支统计 |
| GET | `/api/stats/tags` | 标签统计 |
| GET | `/api/export` | 导出 CSV（含标签列） |

完整接口见 `internal/handler/handler.go`。

## 与 Web 版的关系

- Web 版（`pennypick/`）为浏览器访问的 Web 应用，桌面版为独立客户端，两者功能保持一致。
- 两版的 `internal` 业务代码同步维护（仅模块名不同）；桌面版额外包含 Wails 绑定（`GetAPIBaseURL`、`ExportBills`）。
- 导出差异：桌面版使用系统「另存为」对话框，Web 版使用浏览器下载。
