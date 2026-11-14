package etcd

import (
	"go.etcd.io/etcd/etcdutl/v3/snapshot"
	"go.uber.org/zap"
)

// CheckSnapshotStatus 使用 etcdutl 的只读校验检查快照并返回状态。
func CheckSnapshotStatus(filePath string) (snapshot.Status, error) {
	return snapshot.NewV3(zap.NewNop()).Status(filePath)
}
