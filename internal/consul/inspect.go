package consul

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hashicorp/consul/snapshot"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
)

// SnapshotInfo 包含 snapshot 的元数据信息
type SnapshotInfo struct {
	ID      string
	Size    int64
	Index   uint64
	Term    uint64
	Version raft.SnapshotVersion
}

// InspectSnapshot 检查并验证 Consul snapshot 文件
// 返回 snapshot 的元数据信息，如果文件无效则返回错误
func InspectSnapshot(filePath string) (*SnapshotInfo, error) {
	// 打开文件
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开 snapshot 文件失败: %w", err)
	}
	defer f.Close()

	var readFile *os.File
	var meta *raft.SnapshotMeta

	// 检查是否是内部 raw raft snapshot (state.bin)
	if filepath.Base(filePath) == "state.bin" {
		// 这是内部的 raw raft snapshot，不是 gzipped archive
		readFile = f

		// 假设 meta.json 在同一目录
		metaPath := filepath.Join(filepath.Dir(filePath), "meta.json")
		metaRaw, err := os.ReadFile(metaPath)
		if err != nil {
			return nil, fmt.Errorf("读取 meta.json 失败: %w", err)
		}

		var metaDecoded raft.SnapshotMeta
		if err := json.Unmarshal(metaRaw, &metaDecoded); err != nil {
			return nil, fmt.Errorf("解析 meta.json 失败: %w", err)
		}
		meta = &metaDecoded
	} else {
		// 使用 consul snapshot 包读取 gzipped archive snapshot
		logger := hclog.New(&hclog.LoggerOptions{
			Level:  hclog.Warn, // 只显示警告和错误
			Output: io.Discard, // 不输出日志
		})

		var err error
		readFile, meta, err = snapshot.Read(logger, f)
		if err != nil {
			return nil, fmt.Errorf("读取 snapshot 失败: %w", err)
		}
		defer func() {
			if readFile != nil {
				readFile.Close()
				os.Remove(readFile.Name())
			}
		}()
	}

	// 验证 meta 信息
	if meta == nil {
		return nil, fmt.Errorf("无法获取 snapshot 元数据")
	}

	// 尝试读取一些数据来验证文件完整性
	if readFile != nil {
		buf := make([]byte, 1024)
		if _, err := readFile.Read(buf); err != nil && err != io.EOF {
			return nil, fmt.Errorf("读取 snapshot 数据失败: %w", err)
		}
	}

	info := &SnapshotInfo{
		ID:      meta.ID,
		Size:    meta.Size,
		Index:   meta.Index,
		Term:    meta.Term,
		Version: meta.Version,
	}

	return info, nil
}

