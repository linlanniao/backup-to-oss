package etcd

import (
	"context"
	"fmt"
	"os"

	"github.com/coreos/go-semver/semver"
	"go.etcd.io/etcd/pkg/v3/traceutil"
	"go.etcd.io/etcd/server/v3/lease"
	"go.etcd.io/etcd/server/v3/storage/backend"
	"go.etcd.io/etcd/server/v3/storage/mvcc"
	"go.uber.org/zap"
)

// noOpCluster 是一个 no-op cluster 实现，用于只读的 snapshot status 检查
type noOpCluster struct{}

func (c *noOpCluster) Version() *semver.Version {
	// 返回一个默认版本，对于只读操作来说足够了
	return semver.New("3.6.0")
}

// SnapshotStatusInfo 包含 etcd snapshot 的状态信息
type SnapshotStatusInfo struct {
	Hash      uint32 // snapshot 的哈希值
	Revision  int64  // 修订版本
	TotalKey  int    // 总键数
	TotalSize int64  // 总大小
}

// CheckSnapshotStatus 检查并验证 etcd snapshot 文件的状态
// 参考 etcd 的 snapshot status 实现
// 返回 snapshot 的状态信息，如果文件无效则返回错误
func CheckSnapshotStatus(filePath string) (*SnapshotStatusInfo, error) {
	// 检查文件是否存在
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("获取 snapshot 文件信息失败: %w", err)
	}
	if fileInfo.Size() == 0 {
		return nil, fmt.Errorf("snapshot 文件为空")
	}

	// 创建一个简单的 logger（不输出日志）
	lg := zap.NewNop()

	// 打开 snapshot 文件作为 backend
	be := backend.NewDefaultBackend(lg, filePath)
	defer be.Close()

	// 创建 no-op cluster 用于 lessor
	cluster := &noOpCluster{}

	// 创建 lessor（租约管理器）用于 snapshot 恢复
	// 对于只读的 status 检查，我们需要创建一个 lessor 来避免 panic
	lessor := lease.NewLessor(lg, be, cluster, lease.LessorConfig{})
	defer lessor.Stop()

	// 创建 mvcc store 来读取 snapshot
	store := mvcc.NewStore(lg, be, lessor, mvcc.StoreConfig{})
	defer store.Close()

	// 获取当前修订版本
	rev := store.Rev()

	// 使用 Read 方法创建一个只读事务来验证 snapshot
	// 使用 traceutil.TODO() 而不是 nil，避免 nil pointer dereference
	txn := store.Read(mvcc.ConcurrentReadTxMode, traceutil.TODO())
	defer txn.End()

	// 计算总键数和总大小
	totalKey := 0
	totalSize := int64(0)

	// 首先使用 Count 选项来获取总键数（更高效）
	countResult, err := txn.Range(context.Background(), []byte{}, []byte{0xff}, mvcc.RangeOptions{Count: true})
	if err != nil {
		return nil, fmt.Errorf("读取 snapshot 计数失败: %w", err)
	}
	totalKey = countResult.Count

	// 如果 Count 为 0，尝试获取实际的键值对来计算
	if totalKey == 0 {
		// 使用 Limit 来限制返回的键值对数量，避免内存占用过大
		// 这里我们只读取一部分来验证文件完整性
		result, err := txn.Range(context.Background(), []byte{}, []byte{0xff}, mvcc.RangeOptions{Limit: 1000})
		if err != nil {
			return nil, fmt.Errorf("读取 snapshot 数据失败: %w", err)
		}
		totalKey = len(result.KVs)
		// 计算总大小（键值对的总大小）
		for _, kv := range result.KVs {
			totalSize += int64(len(kv.Key) + len(kv.Value))
		}
	} else {
		// 如果 Count > 0，使用文件大小作为总大小的近似值
		// 因为遍历所有键值对可能很耗时
		totalSize = fileInfo.Size()
	}

	// 如果总大小为 0，使用文件大小作为近似值
	if totalSize == 0 {
		totalSize = fileInfo.Size()
	}

	// 计算哈希值（使用修订版本和总大小的简单哈希）
	hash := uint32((rev + totalSize) % 0xFFFFFFFF)

	info := &SnapshotStatusInfo{
		Hash:      hash,
		Revision:  rev,
		TotalKey:  totalKey,
		TotalSize: totalSize,
	}

	return info, nil
}
