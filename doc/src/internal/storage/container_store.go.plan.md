# container_store.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/storage/container_store.go
- 文档文件: doc/src/internal/storage/container_store.go.plan.md
- 文件类型: Go 源码
- 所属模块: storage

## 2. 核心职责
- 该文件实现了容器注册表的磁盘持久化存储层 `ContainerStore`，管理所有已注册容器的生命周期数据。所有容器信息存储在单个文件 `~/.eino-agent/containers.json` 中。提供容器的增删改查操作、按 SSH 地址检索和已注册 Docker ID 提取等功能，所有操作通过读写锁保证线程安全。支持应用重启后的容器重连和复用。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `model.Container` 容器对象；容器 ID (string)；SSH 主机地址和端口
- 输出结果: 容器列表 `[]model.Container`、单个容器 `*model.Container`、Docker ID 列表 `[]string`；磁盘文件的创建/修改

## 4. 关键实现细节
- 结构体/接口定义:
  - `ContainerStore` — 容器存储结构体，持有文件路径 `path`、内存中的容器列表 `containers` 和读写锁 `mu`
- 导出函数/方法:
  - `NewContainerStore() (*ContainerStore, error)` — 创建存储实例，加载已有数据
  - `(s *ContainerStore) List() []model.Container` — 列出所有容器（返回副本）
  - `(s *ContainerStore) Get(id string) (*model.Container, error)` — 按 ID 获取容器
  - `(s *ContainerStore) Add(container *model.Container) error` — 注册新容器并持久化
  - `(s *ContainerStore) Update(container *model.Container) error` — 更新容器信息并持久化
  - `(s *ContainerStore) Remove(id string) error` — 从注册表中删除容器
  - `(s *ContainerStore) FindBySSH(host string, port int) []model.Container` — 按 SSH 地址查找容器
  - `(s *ContainerStore) RegisteredDockerIDs() []string` — 获取所有非销毁状态的 Docker ID
- 未导出函数:
  - `(s *ContainerStore) load() error` — 从磁盘加载容器数据
  - `(s *ContainerStore) save() error` — 将容器数据持久化到磁盘
- Wails 绑定方法: 无（通过 ContainerService 和 SandboxService 间接使用）
- 事件发射: 无

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/model` (Container, ContainerStatus, ContainerDestroyed 类型)
- 外部依赖:
  - `encoding/json` (JSON 序列化)
  - `fmt` (错误格式化)
  - `os`, `path/filepath` (文件系统操作)
  - `sync` (读写锁)
- 关键配置:
  - 存储文件: `~/.eino-agent/containers.json`

## 6. 变更影响面
- 修改存储路径会影响已有用户的容器注册数据
- `RegisteredDockerIDs` 的过滤逻辑变更会影响容器清理策略（`setup.go` 使用该方法避免清理已注册容器）
- `FindBySSH` 被沙箱连接流程使用，用于检测是否存在可复用的容器
- 该存储层被 `service.ContainerService`、`service.SandboxService` 和 `service.SessionService` 使用
- 内存中的容器列表与磁盘文件的一致性依赖于每次写操作后立即持久化

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 采用全量写入策略（每次修改后重写整个文件），容器数量极大时可考虑增量更新。
- `load` 方法在初始化时调用，加载失败时静默初始化为空列表，可考虑添加日志警告。
- `Add` 方法未做 ID 唯一性校验，依赖调用方确保不重复注册。
- `List` 返回副本保证线程安全，但 `FindBySSH` 直接 append 到 result 切片，返回的是值拷贝（Container 为值类型），同样安全。
