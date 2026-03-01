# store.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/config/store.go
- 文档文件: doc/src/internal/config/store.go.plan.md
- 文件类型: Go 源码
- 所属模块: config

## 2. 核心职责
- 该文件实现了应用配置的持久化存储层 `Store`，负责将 `AppConfig` 以 JSON 格式读写到用户主目录下的 `~/.starxo/config.json` 文件。提供线程安全的配置读取、更新和保存操作，支持原子性的 `Update` 方法（读-改-写一体化）。首次创建时若配置文件不存在则使用默认配置。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: 文件系统 `~/.starxo/config.json`；`Update` 方法接收修改函数 `func(*AppConfig)`
- 输出结果: `Get()` 返回配置副本；`Save()`/`Update()` 将配置持久化到磁盘

## 4. 关键实现细节
- 结构体/接口定义:
  - `Store` — 配置存储结构体，包含文件路径 `path`、配置指针 `config`、读写锁 `mu`
- 导出函数/方法:
  - `NewStore() (*Store, error)` — 创建存储实例，自动加载或初始化默认配置
  - `(s *Store) Load() error` — 从磁盘加载配置
  - `(s *Store) Save() error` — 将当前配置保存到磁盘
  - `(s *Store) Get() *AppConfig` — 返回配置的深拷贝副本
  - `(s *Store) Update(fn func(*AppConfig)) error` — 原子性更新配置并持久化
- Wails 绑定方法: 无（通过 SettingsService 间接暴露）
- 事件发射: 无

## 5. 依赖关系
- 内部依赖: `config.AppConfig`, `config.DefaultConfig` (同包)
- 外部依赖:
  - `encoding/json` (JSON 序列化)
  - `os` (文件读写)
  - `path/filepath` (路径构建)
  - `sync` (读写锁)
- 关键配置:
  - 配置目录: `~/.starxo/`
  - 配置文件: `~/.starxo/config.json`

## 6. 变更影响面
- 修改存储路径会影响配置文件的查找位置，需考虑已有用户的迁移
- 修改锁策略会影响并发安全性
- `Get()` 返回的是副本而非引用，修改此行为会影响所有调用方的线程安全假设
- 该 Store 被 `app.go`、`SettingsService`、`ChatService`、`SandboxService` 等多个组件使用

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- `Get()` 方法返回的是浅拷贝，若 `AppConfig` 中新增引用类型字段（如 map、slice），需确保拷贝的完整性。
- 当前 `Load()` 使用写锁而 `Save()` 使用读锁，`Save()` 应考虑使用写锁以保证文件写入的原子性。
- 配置文件目录名 `.starxo` 为硬编码，后续可考虑通过环境变量配置。
