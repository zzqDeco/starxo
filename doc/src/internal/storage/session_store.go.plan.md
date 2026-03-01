# session_store.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/storage/session_store.go
- 文档文件: doc/src/internal/storage/session_store.go.plan.md
- 文件类型: Go 源码
- 所属模块: storage

## 2. 核心职责
- 该文件实现了会话的磁盘持久化存储层 `SessionStore`，管理会话元数据和对话消息的读写。每个会话以独立目录存储在 `~/.eino-agent/sessions/{id}/` 下，包含 `session.json`（元数据）、`messages.json`（对话历史）和 `display.json`（前端展示数据）三个文件。提供会话的 CRUD 操作、消息的保存/加载以及前端展示数据的持久化，所有操作通过读写锁保证线程安全。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: 会话 ID (string)、会话标题 (string)、`model.Session` 对象、`[]model.PersistedMessage` 消息列表、前端展示数据 (JSON string)
- 输出结果: `model.Session` 对象、`[]model.PersistedMessage` 消息列表、前端展示数据字符串；磁盘文件的创建/修改/删除

## 4. 关键实现细节
- 结构体/接口定义:
  - `SessionStore` — 会话存储结构体，持有基础目录路径 `baseDir` 和读写锁 `mu`
- 导出函数/方法:
  - `NewSessionStore() (*SessionStore, error)` — 创建存储实例，确保目录存在
  - `(s *SessionStore) List() ([]model.Session, error)` — 列出所有会话（按更新时间降序）
  - `(s *SessionStore) Get(id string) (*model.Session, error)` — 按 ID 获取会话
  - `(s *SessionStore) Create(title string) (*model.Session, error)` — 创建新会话（生成 UUID 前 8 位作为 ID）
  - `(s *SessionStore) Update(sess *model.Session) error` — 更新会话元数据（自动更新 UpdatedAt）
  - `(s *SessionStore) Delete(id string) error` — 删除会话及其全部数据
  - `(s *SessionStore) SaveMessages(sessionID string, messages []model.PersistedMessage) error` — 保存对话消息
  - `(s *SessionStore) LoadMessages(sessionID string) ([]model.PersistedMessage, error)` — 加载对话消息
  - `(s *SessionStore) SaveDisplayData(sessionID string, data string) error` — 保存前端展示数据
  - `(s *SessionStore) LoadDisplayData(sessionID string) (string, error)` — 加载前端展示数据
- 未导出函数:
  - `(s *SessionStore) loadSession(id string) (*model.Session, error)` — 从磁盘读取会话元数据
  - `(s *SessionStore) saveSession(sess *model.Session) error` — 将会话元数据写入磁盘
- Wails 绑定方法: 无（通过 SessionService 间接暴露）
- 事件发射: 无

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/model` (Session, PersistedMessage 类型)
- 外部依赖:
  - `encoding/json` (JSON 序列化)
  - `fmt` (错误格式化)
  - `os`, `path/filepath` (文件系统操作)
  - `sort` (会话列表排序)
  - `sync` (读写锁)
  - `time` (时间戳生成)
  - `github.com/google/uuid` (会话 ID 生成)
- 关键配置:
  - 存储目录: `~/.eino-agent/sessions/`

## 6. 变更影响面
- 修改存储路径或目录结构会影响已有用户的会话数据访问
- 修改 `Create` 方法的 ID 生成策略可能导致 ID 冲突风险变化
- `SaveDisplayData`/`LoadDisplayData` 的格式变更会影响前端会话恢复
- 该存储层被 `service.SessionService` 使用，接口变更需同步更新服务层
- `List` 方法遍历目录结构，会话数量过多时可能有性能问题

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 会话 ID 使用 UUID 前 8 位（`uuid.New().String()[:8]`），在会话数量极大时存在碰撞风险，可考虑使用完整 UUID。
- `List` 方法对损坏的会话数据采用静默跳过策略 (`continue`)，可考虑添加日志记录。
- `SaveDisplayData` 接收原始 JSON 字符串而非结构体，未做格式校验，依赖调用方确保数据合法性。
- 存储目录名 `.eino-agent` 与 `config.Store` 共享，应保持一致。
