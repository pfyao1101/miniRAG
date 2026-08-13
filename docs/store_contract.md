# MemoryStore 行为契约

## 1. 文档目的

本文档定义当前 `MemoryStore` 对调用者可观察的行为，作为单元测试、gRPC 错误映射和后续持久化实现的共同基线。

本契约只描述当前已实现的语义，不包含 WAL、Segment、ANN、metadata filter、复制或分片。

## 2. 数据模型

```go
type Record struct {
    ID       string
    Vector   []float32
    Text     string
    Metadata map[string]string
}

type Result struct {
    ID    string
    Score float32
}
```

当前 Store 接口为：

```go
type VectorStore interface {
    Insert(ctx context.Context, record Record) error
    Get(ctx context.Context, id string) (Record, error)
    Delete(ctx context.Context, id string) error
    Search(ctx context.Context, query []float32, k int) ([]vector.Result, error)
}
```

`Search` 只返回 ID 和分数，不返回文本、metadata 或向量。调用者如果需要完整记录，需要再调用 `Get`。

## 3. Store 创建

### `NewMemoryStore(dimension int)`

- `dimension` 必须大于 0。
- `dimension <= 0` 时返回 `ErrInvalidDimension`。
- 一个 Store 实例的向量维度在创建后固定，当前不支持动态修改。
- 新建 Store 不包含任何记录。

## 4. Insert 契约

### 成功条件

记录必须同时满足：

- `ctx` 在调用开始时未取消且未超时。
- `record.ID` 不为空字符串。
- `len(record.Vector)` 等于 Store 的固定维度。
- `record.Vector` 不是零向量。
- Store 中不存在相同 ID。

`Text` 可以为空字符串，`Metadata` 可以为 `nil` 或空 map。

### 错误行为

| 场景 | 错误 |
|---|---|
| context 在入口处已取消 | `context.Canceled` |
| context 在入口处已超时 | `context.DeadlineExceeded` |
| ID 为空 | `ErrEmptyID` |
| 向量维度不匹配 | `ErrVectorDimensionMismatch` |
| 零向量 | `vector.ErrZeroVector` |
| ID 已存在 | `ErrDuplicateID` |

重复 Insert 不会覆盖原记录；当前没有 Update 或 Upsert 语义。

### 所有权

Insert 成功时，Store 会复制 `Vector` 和 `Metadata`。调用者之后修改原始 slice 或 map，不会改变 Store 内部的记录。

## 5. Get 契约

### 成功行为

- ID 存在时返回完整 `Record`。
- 返回记录中的 `Vector` 和 `Metadata` 是副本。
- 调用者修改返回值，不会改变 Store 内部状态。

### 错误行为

| 场景 | 错误 | 返回记录 |
|---|---|---|
| context 在入口处已取消或超时 | context 原始错误 | `Record{}` |
| ID 为空 | `ErrEmptyID` | `Record{}` |
| ID 不存在 | `ErrRecordNotFound` | `Record{}` |

## 6. Delete 契约

- ID 存在时，Delete 删除该记录并返回 `nil`。
- ID 为空时返回 `ErrEmptyID`。
- ID 不存在时返回 `ErrRecordNotFound`。
- Delete 当前不是幂等删除；对同一 ID 连续调用两次，第二次返回 `ErrRecordNotFound`。
- Delete 成功后，后续 Get 返回 `ErrRecordNotFound`。

当前删除是内存 map 中的物理删除，尚无 tombstone、version 或恢复语义。

## 7. Search 契约

### 输入条件

- `len(query)` 必须等于 Store 维度。
- query 不能是零向量。
- `k` 必须大于 0。

### 搜索与排序

- 当前对 Store 中的全部记录进行精确余弦相似度扫描。
- 结果首先按 `Score` 降序排列。
- 分数相同时按 `ID` 字典序升序排列。
- `k` 大于记录数时，返回所有记录。
- Store 为空且 query、`k` 有效时，返回长度为 0 的结果和 `nil` 错误。调用者不应依赖空结果的 slice 是 `nil` 还是 non-nil。
- Search 不修改 query，也不暴露 Store 内部的 vector 或 metadata。

### 错误行为

| 场景 | 错误 | 结果 |
|---|---|---|
| context 已取消或超时 | context 原始错误 | `nil` |
| query 维度不匹配 | `ErrVectorDimensionMismatch` | `nil` |
| query 是零向量 | `vector.ErrZeroVector` | `nil` |
| `k <= 0` | `vector.ErrInvalidK` | `nil` |

### 复杂度

记录数为 `N`、向量维度为 `D`、返回数量上限为 `K` 时：

- 相似度扫描：`O(ND)`。
- 当前先构造全部 `N` 个候选，再做大小为 `K` 的最小堆 Top-K：`O(N log K)`。
- 总时间：`O(ND + N log K)`。
- 当前候选结果额外空间为 `O(N)`，Top-K 堆和输出为 `O(K)`。

## 8. Context 与取消语义

当前 context 取消是尽力而为（best effort），不是事务性保证。

- Insert、Get 和 Delete 在进入方法时检查一次 `ctx.Err()`。
- 它们在等待 `Mutex/RWMutex` 期间不会被 context 直接唤醒。
- 如果 Insert 或 Delete 通过了入口检查，随后在等锁时 context 被取消，当前实现仍可能在获得锁后完成修改。
- Search 除入口检查外，还会在每条记录扫描前检查 context，并在 Top-K 前再检查一次。
- Search 的取消响应延迟上界不是固定时间；它至少受单条向量相似度计算时间和调度延迟影响。

因此，调用者不应把“客户端已看到 deadline/canceled”理解为“写操作一定没有发生”。如果后续需要这种保证，必须单独设计幂等键、序列号或事务语义。

## 9. 并发与一致性

- `MemoryStore` 通过 `sync.RWMutex` 支持并发调用。
- Insert 和 Delete 持有写锁。
- Get 持有读锁，并在读锁内完成记录复制。
- Search 在整个扫描和 Top-K 计算期间持有读锁。
- 多个 Get/Search 可以并发执行；Insert/Delete 与其他写操作、Get 或 Search 互斥。
- 一次 Search 不会观察到扫描中途的 Insert/Delete，因为写者在 Search 释放读锁前无法修改 map。
- 长时间 Search 可能增加 Insert/Delete 的等待时间；当前尚无写延迟或锁竞争的正式性能保证。

当前单个 Store 进程内的所有操作都由同一把 `RWMutex` 同步；写操作串行执行，读操作可以并发执行。该语义不包含跨进程、持久化或分布式一致性保证。

## 10. 数值边界与已知限制

当前实现：

- 使用 `float64` 累加点积和模长，最终分数转为 `float32`。
- 不要求输入向量预先归一化。
- 不检查 `NaN` 或正负无穷大。
- 对包含 `NaN`/`Inf` 的向量，Insert 可能成功，Search 分数及排序语义不作保证。
- 没有定义最大向量维度、最大文本长度、metadata 数量或总记录数。

在这些边界被明确之前，gRPC 层不应额外声称已拒绝 `NaN`/`Inf` 或已限制请求大小。

## 11. 当前不保证的能力

以下内容不属于当前契约：

- 进程退出后的数据持久化。
- Insert/Delete 的跨网络重试幂等性。
- Update、Upsert 或 BatchInsert。
- 记录版本、sequence、tombstone 或快照。
- metadata filter、rerank 或 ANN/HNSW。
- 多 Store 实例、多进程或多节点之间的一致性。
- 吞吐、p95/p99 延迟、最大容量或公平性 SLA。

## 12. 与 gRPC 状态码的当前映射

Store 本身不依赖 gRPC。当前 gRPC adapter 将领域错误映射为：

| Store/context 错误 | gRPC code |
|---|---|
| `context.Canceled` | `Canceled` |
| `context.DeadlineExceeded` | `DeadlineExceeded` |
| `ErrEmptyID` | `InvalidArgument` |
| `ErrVectorDimensionMismatch` | `InvalidArgument` |
| `vector.ErrEmptyVector` | `InvalidArgument` |
| `vector.ErrZeroVector` | `InvalidArgument` |
| `vector.ErrInvalidK` | `InvalidArgument` |
| `ErrDuplicateID` | `AlreadyExists` |
| `ErrRecordNotFound` | `NotFound` |
| 未识别的内部错误 | `Internal`，不暴露内部错误文本 |

该映射属于 transport adapter，不应反向迫使 Store 引入 gRPC 类型。

## 13. Store 契约的验证边界

当前自动化验证覆盖：

- Store 构造、CRUD、Search 和主要错误路径。
- Insert/Get 的防御性复制。
- Top-K 相同分数的稳定次序。
- 并发 Insert/Get/Search/Delete 与重复 Insert。
- Go race detector。

上述自动化检查证明当前功能与基本并发正确性，但不构成性能 SLA、持久性保证或跨进程一致性证明。gRPC 联合测试、进程生命周期、Linux 实验和后续持久化测试各自有独立的验收边界，不由本 Store 契约代替。

本契约发生有意义的行为变更时，应先更新测试和本文档，再让 gRPC、持久化和分布式实现继承新语义。
