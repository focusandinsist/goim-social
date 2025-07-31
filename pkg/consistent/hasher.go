package consistent

import (
	"hash/crc64"
	"hash/fnv"
)

// CRC64Hasher 使用CRC64算法的哈希器
type CRC64Hasher struct {
	table *crc64.Table
}

// NewCRC64Hasher 创建新的CRC64哈希器
func NewCRC64Hasher() *CRC64Hasher {
	return &CRC64Hasher{
		table: crc64.MakeTable(crc64.ISO),
	}
}

// Sum64 计算字节切片的64位哈希值
func (h *CRC64Hasher) Sum64(data []byte) uint64 {
	return crc64.Checksum(data, h.table)
}

// FNVHasher 使用FNV算法的哈希器
type FNVHasher struct{}

// NewFNVHasher 创建新的FNV哈希器
func NewFNVHasher() *FNVHasher {
	return &FNVHasher{}
}

// Sum64 计算字节切片的64位哈希值
func (h *FNVHasher) Sum64(data []byte) uint64 {
	hash := fnv.New64a()
	hash.Write(data)
	return hash.Sum64()
}
