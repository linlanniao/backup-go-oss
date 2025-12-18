package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"os"
	"os/exec"

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
// 使用 etcdutl 命令获取准确的状态信息，确保与官方工具一致
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

	// 尝试使用 etcdutl 命令获取状态信息（优先使用，确保与官方工具一致）
	statusInfo, err := getStatusViaEtcdutl(filePath)
	if err == nil {
		return statusInfo, nil
	}

	// 如果 etcdutl 不可用，回退到使用 etcdctl
	statusInfo, err = getStatusViaEtcdctl(filePath)
	if err == nil {
		return statusInfo, nil
	}

	// 如果命令行工具都不可用，使用内部实现（可能不够准确）
	return getStatusViaInternal(filePath)
}

// getStatusViaEtcdutl 使用 etcdutl 命令获取 snapshot 状态
func getStatusViaEtcdutl(filePath string) (*SnapshotStatusInfo, error) {
	cmd := exec.Command("etcdutl", "snapshot", "status", filePath, "--write-out=json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("执行 etcdutl 命令失败: %w", err)
	}

	return parseStatusJSON(output)
}

// getStatusViaEtcdctl 使用 etcdctl 命令获取 snapshot 状态（兼容旧版本）
func getStatusViaEtcdctl(filePath string) (*SnapshotStatusInfo, error) {
	cmd := exec.Command("etcdctl", "snapshot", "status", filePath, "--write-out=json")
	cmd.Env = append(os.Environ(), "ETCDCTL_API=3")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("执行 etcdctl 命令失败: %w", err)
	}

	return parseStatusJSON(output)
}

// parseStatusJSON 解析 JSON 格式的状态信息
// 处理可能包含警告信息的输出（如 etcdctl 的弃用警告）
func parseStatusJSON(output []byte) (*SnapshotStatusInfo, error) {
	// 查找 JSON 对象的开始位置（跳过可能的警告信息）
	start := 0
	for i := 0; i < len(output); i++ {
		if output[i] == '{' {
			start = i
			break
		}
	}

	var result struct {
		Hash      uint32 `json:"hash"`
		Revision  int64  `json:"revision"`
		TotalKey  int    `json:"totalKey"`
		TotalSize int64  `json:"totalSize"`
	}

	if err := json.Unmarshal(output[start:], &result); err != nil {
		return nil, fmt.Errorf("解析 JSON 输出失败: %w", err)
	}

	return &SnapshotStatusInfo{
		Hash:      result.Hash,
		Revision:  result.Revision,
		TotalKey:  result.TotalKey,
		TotalSize: result.TotalSize,
	}, nil
}

// getStatusViaInternal 使用内部实现获取 snapshot 状态（备用方案）
func getStatusViaInternal(filePath string) (*SnapshotStatusInfo, error) {
	// 创建一个简单的 logger（不输出日志）
	lg := zap.NewNop()

	// 打开 snapshot 文件作为 backend
	be := backend.NewDefaultBackend(lg, filePath)
	defer be.Close()

	// 创建 no-op cluster 用于 lessor
	cluster := &noOpCluster{}

	// 创建 lessor（租约管理器）用于 snapshot 恢复
	lessor := lease.NewLessor(lg, be, cluster, lease.LessorConfig{})
	defer lessor.Stop()

	// 创建 mvcc store 来读取 snapshot
	store := mvcc.NewStore(lg, be, lessor, mvcc.StoreConfig{})
	defer store.Close()

	// 获取当前修订版本
	rev := store.Rev()

	// 使用 Read 方法创建一个只读事务来验证 snapshot
	txn := store.Read(mvcc.ConcurrentReadTxMode, traceutil.TODO())
	defer txn.End()

	// 遍历所有键值对来计算总键数、总大小和哈希值
	totalKey := 0
	totalSize := int64(0)
	hash := crc32.NewIEEE()

	// 使用 Range 遍历所有键值对
	// 注意：这里需要遍历所有键值对才能得到准确的统计信息
	key := []byte{}
	end := []byte{0xff}

	for {
		result, err := txn.Range(context.Background(), key, end, mvcc.RangeOptions{Limit: 1000})
		if err != nil {
			return nil, fmt.Errorf("读取 snapshot 数据失败: %w", err)
		}

		if len(result.KVs) == 0 {
			break
		}

		for _, kv := range result.KVs {
			totalKey++
			kvSize := int64(len(kv.Key) + len(kv.Value))
			totalSize += kvSize

			// 更新哈希值（包含键和值）
			hash.Write(kv.Key)
			hash.Write(kv.Value)
		}

		// 如果返回的键值对数量少于 Limit，说明已经遍历完所有键
		if len(result.KVs) < 1000 {
			break
		}

		// 设置下一次查询的起始键（使用最后一个键的下一个键）
		key = append(result.KVs[len(result.KVs)-1].Key, 0)
	}

	info := &SnapshotStatusInfo{
		Hash:      hash.Sum32(),
		Revision:  rev,
		TotalKey:  totalKey,
		TotalSize: totalSize,
	}

	return info, nil
}
