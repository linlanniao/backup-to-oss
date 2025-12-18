package consul

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/hashicorp/consul-net-rpc/go-msgpack/codec"
	"github.com/hashicorp/consul/agent/consul/fsm"
	"github.com/hashicorp/consul/agent/structs"
	"github.com/hashicorp/consul/snapshot"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
)

// TypeStats 包含每种类型的统计信息
type TypeStats struct {
	Name  string
	Count int
	Size  int64
}

// SnapshotInfo 包含 snapshot 的元数据信息和详细统计
type SnapshotInfo struct {
	ID        string
	Size      int64
	Index     uint64
	Term      uint64
	Version   raft.SnapshotVersion
	Stats     []TypeStats // 各种类型的统计信息
	TotalSize int64       // 总大小
}

// countingReader 用于跟踪读取的字节数
type countingReader struct {
	wrappedReader io.Reader
	read          int64
}

func (r *countingReader) Read(p []byte) (n int, err error) {
	n, err = r.wrappedReader.Read(p)
	if err == nil {
		r.read += int64(n)
	}
	return n, err
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

	// 解析 snapshot 获取详细统计信息
	statsMap := make(map[structs.MessageType]TypeStats)
	var totalSize int64

	if readFile != nil {
		// 重置文件指针到开头
		if _, err := readFile.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("重置文件指针失败: %w", err)
		}

		cr := &countingReader{wrappedReader: readFile}
		handler := func(header *fsm.SnapshotHeader, msg structs.MessageType, dec *codec.Decoder) error {
			name := structs.MessageType.String(msg)
			stats := statsMap[msg]
			if stats.Name == "" {
				stats.Name = name
			}

			var val interface{}
			if err := dec.Decode(&val); err != nil {
				return fmt.Errorf("解码消息类型 %v 失败: %w", name, err)
			}

			// 计算本次读取的大小
			size := cr.read - totalSize
			stats.Size += size
			stats.Count++
			totalSize = cr.read
			statsMap[msg] = stats

			return nil
		}

		if err := fsm.ReadSnapshot(cr, handler); err != nil {
			// 如果解析失败，仍然返回基本信息
			return &SnapshotInfo{
				ID:        meta.ID,
				Size:      meta.Size,
				Index:     meta.Index,
				Term:      meta.Term,
				Version:   meta.Version,
				Stats:     []TypeStats{},
				TotalSize: 0,
			}, nil
		}
	}

	// 将统计信息转换为切片并排序
	stats := make([]TypeStats, 0, len(statsMap))
	for _, s := range statsMap {
		stats = append(stats, s)
	}
	// 按大小降序排序，如果大小相同则按名称排序
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Size == stats[j].Size {
			return stats[i].Name < stats[j].Name
		}
		return stats[i].Size > stats[j].Size
	})

	info := &SnapshotInfo{
		ID:        meta.ID,
		Size:      meta.Size,
		Index:     meta.Index,
		Term:      meta.Term,
		Version:   meta.Version,
		Stats:     stats,
		TotalSize: totalSize,
	}

	return info, nil
}
