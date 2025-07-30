package consistent

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
)

const (
	// DefaultPartitionCount 默认分区数量
	DefaultPartitionCount int = 271
	// DefaultReplicationFactor 默认复制因子
	DefaultReplicationFactor int = 20
	// DefaultLoad 默认负载因子
	DefaultLoad float64 = 1.25
)

// ErrInsufficientMemberCount 表示成员数量不足的错误
var ErrInsufficientMemberCount = errors.New("成员数量不足")

// Hasher 负责为提供的字节切片生成无符号64位哈希值
// Hasher应该最小化冲突（为不同的字节切片生成相同的哈希值）
// 同时性能也很重要，优先选择快速函数
type Hasher interface {
	Sum64([]byte) uint64
}

// Member 表示一致性哈希环中的成员接口
type Member interface {
	String() string
}

// Config 表示控制一致性哈希包的配置结构
type Config struct {
	// Hasher 负责为提供的字节切片生成无符号64位哈希值
	Hasher Hasher

	// 键分布在分区中。质数有利于均匀分布键
	// 如果你有太多键，请选择一个大的PartitionCount
	PartitionCount int

	// 成员在一致性哈希环上被复制。这个数字表示一个成员
	// 在环上被复制多少次
	ReplicationFactor int

	// Load 用于计算平均负载
	Load float64
}

// Consistent 保存一致性哈希环成员的信息
type Consistent struct {
	mu sync.RWMutex

	config         Config
	hasher         Hasher
	sortedSet      []uint64
	partitionCount uint64
	loads          map[string]float64
	members        map[string]Member
	partitions     map[int]Member
	ring           map[uint64]Member

	// 缓存相关字段
	cachedMembers []Member
	membersDirty  bool
}

// New 创建并返回一个新的Consistent对象
func New(members []Member, config Config) *Consistent {
	if config.Hasher == nil {
		panic("Hasher不能为nil")
	}
	if config.PartitionCount == 0 {
		config.PartitionCount = DefaultPartitionCount
	}
	if config.ReplicationFactor == 0 {
		config.ReplicationFactor = DefaultReplicationFactor
	}
	if config.Load == 0 {
		config.Load = DefaultLoad
	}

	c := &Consistent{
		config:         config,
		members:        make(map[string]Member),
		partitionCount: uint64(config.PartitionCount),
		ring:           make(map[uint64]Member),
		membersDirty:   true,
	}

	c.hasher = config.Hasher
	for _, member := range members {
		c.add(member)
	}
	if members != nil {
		c.distributePartitions()
	}
	return c
}

// GetMembers 返回成员的线程安全副本。如果没有成员，返回空的Member切片
func (c *Consistent) GetMembers() []Member {
	// 先尝试读锁检查缓存
	c.mu.RLock()
	if !c.membersDirty && c.cachedMembers != nil {
		// 缓存有效，在读锁下安全地返回副本
		result := make([]Member, len(c.cachedMembers))
		copy(result, c.cachedMembers)
		c.mu.RUnlock()
		return result
	}
	c.mu.RUnlock() // 释放读锁，准备获取写锁

	// 获取写锁来更新缓存
	c.mu.Lock()
	defer c.mu.Unlock()

	// 获取写锁后，可能其他goroutine已经更新了缓存，需要再次检查
	if !c.membersDirty && c.cachedMembers != nil {
		result := make([]Member, len(c.cachedMembers))
		copy(result, c.cachedMembers)
		return result
	}

	// 创建成员列表的线程安全副本
	members := make([]Member, 0, len(c.members))
	for _, member := range c.members {
		members = append(members, member)
	}

	// 更新缓存（在写锁保护下安全）
	c.cachedMembers = make([]Member, len(members))
	copy(c.cachedMembers, members)
	c.membersDirty = false

	return members
}

// AverageLoad 暴露当前平均负载
func (c *Consistent) AverageLoad() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.averageLoad()
}

// averageLoad 计算平均负载（内部方法）
func (c *Consistent) averageLoad() float64 {
	if len(c.members) == 0 {
		return 0
	}

	avgLoad := float64(c.partitionCount/uint64(len(c.members))) * c.config.Load
	return math.Ceil(avgLoad)
}

// distributeWithLoad 根据负载分布分区
func (c *Consistent) distributeWithLoad(partID, idx int, partitions map[int]Member, loads map[string]float64) {
	avgLoad := c.averageLoad()
	var count int
	for {
		count++
		if count >= len(c.sortedSet) {
			// 用户需要减少分区数量，增加成员数量或增加负载因子
			panic("没有足够的空间来分布分区")
		}
		i := c.sortedSet[idx]
		member := c.ring[i]
		load := loads[member.String()]
		if load+1 <= avgLoad {
			partitions[partID] = member
			loads[member.String()]++
			return
		}
		idx++
		if idx >= len(c.sortedSet) {
			idx = 0
		}
	}
}

// distributePartitions 分布分区
func (c *Consistent) distributePartitions() {
	loads := make(map[string]float64)
	partitions := make(map[int]Member)

	bs := make([]byte, 8)
	for partID := uint64(0); partID < c.partitionCount; partID++ {
		binary.LittleEndian.PutUint64(bs, partID)
		key := c.hasher.Sum64(bs)
		idx := sort.Search(len(c.sortedSet), func(i int) bool {
			return c.sortedSet[i] >= key
		})
		if idx >= len(c.sortedSet) {
			idx = 0
		}
		c.distributeWithLoad(int(partID), idx, partitions, loads)
	}
	c.partitions = partitions
	c.loads = loads
}

// add 添加成员到哈希环（内部方法）
func (c *Consistent) add(member Member) {
	for i := 0; i < c.config.ReplicationFactor; i++ {
		key := []byte(fmt.Sprintf("%s%d", member.String(), i))
		h := c.hasher.Sum64(key)
		c.ring[h] = member
		c.sortedSet = append(c.sortedSet, h)
	}
	// 按升序排序哈希值
	sort.Slice(c.sortedSet, func(i int, j int) bool {
		return c.sortedSet[i] < c.sortedSet[j]
	})
	// 在此映射中存储成员有助于查找分区的备份成员
	c.members[member.String()] = member
	// 标记成员缓存为脏
	c.membersDirty = true
}

// Add 向一致性哈希环添加新成员（优化版本）
func (c *Consistent) Add(member Member) {
	// 先检查成员是否已存在（只需要读锁）
	c.mu.RLock()
	if _, ok := c.members[member.String()]; ok {
		c.mu.RUnlock()
		return
	}
	c.mu.RUnlock()

	// 在临时变量中计算新的分区分布（不需要锁）
	newPartitions, newLoads := c.calculatePartitionsWithNewMember(member)

	// 获取写锁，快速更新数据结构
	c.mu.Lock()
	defer c.mu.Unlock()

	// 再次检查成员是否已存在（双重检查）
	if _, ok := c.members[member.String()]; ok {
		return
	}

	// 添加成员到环中
	c.addToRing(member)

	// 快速更新分区和负载信息
	c.partitions = newPartitions
	c.loads = newLoads
	c.members[member.String()] = member
	c.membersDirty = true
}

// delSlice 从切片中删除值（使用二分查找优化）
func (c *Consistent) delSlice(val uint64) {
	// 使用二分查找定位元素位置
	idx := sort.Search(len(c.sortedSet), func(i int) bool {
		return c.sortedSet[i] >= val
	})

	// 检查是否找到了确切的值
	if idx < len(c.sortedSet) && c.sortedSet[idx] == val {
		// 删除找到的元素
		c.sortedSet = append(c.sortedSet[:idx], c.sortedSet[idx+1:]...)
	}
}

// Remove 从一致性哈希环中移除成员（优化版本）
func (c *Consistent) Remove(name string) {
	// 先检查成员是否存在（只需要读锁）
	c.mu.RLock()
	if _, ok := c.members[name]; !ok {
		c.mu.RUnlock()
		return
	}
	c.mu.RUnlock()

	// 在临时变量中计算移除成员后的分区分布（不需要锁）
	newPartitions, newLoads := c.calculatePartitionsWithoutMember(name)

	// 获取写锁，快速更新数据结构
	c.mu.Lock()
	defer c.mu.Unlock()

	// 再次检查成员是否存在（双重检查）
	if _, ok := c.members[name]; !ok {
		return
	}

	// 从环中移除成员
	c.removeFromRing(name)

	// 快速更新分区和负载信息
	delete(c.members, name)
	c.membersDirty = true

	if len(c.members) == 0 {
		// 一致性哈希环现在为空，重置分区表
		c.partitions = make(map[int]Member)
		c.loads = make(map[string]float64)
		return
	}

	c.partitions = newPartitions
	c.loads = newLoads
}

// LoadDistribution 暴露成员的负载分布
func (c *Consistent) LoadDistribution() map[string]float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 创建线程安全副本
	res := make(map[string]float64)
	for member, load := range c.loads {
		res[member] = load
	}
	return res
}

// FindPartitionID 返回给定键的分区ID
func (c *Consistent) FindPartitionID(key []byte) int {
	hkey := c.hasher.Sum64(key)
	return int(hkey % c.partitionCount)
}

// GetPartitionOwner 返回给定分区的所有者
func (c *Consistent) GetPartitionOwner(partID int) Member {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.getPartitionOwner(partID)
}

// getPartitionOwner 返回给定分区的所有者（非线程安全）
func (c *Consistent) getPartitionOwner(partID int) Member {
	member, ok := c.partitions[partID]
	if !ok {
		return nil
	}
	// 直接返回成员
	return member
}

// LocateKey 为给定键找到归属
func (c *Consistent) LocateKey(key []byte) Member {
	partID := c.FindPartitionID(key)
	return c.GetPartitionOwner(partID)
}

// getClosestN 获取最接近的N个成员（内部方法）
func (c *Consistent) getClosestN(partID, count int) ([]Member, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var res []Member
	if count > len(c.members) {
		return res, ErrInsufficientMemberCount
	}

	var ownerKey uint64
	owner := c.getPartitionOwner(partID)
	// 哈希并排序所有名称
	var keys []uint64
	kmems := make(map[uint64]Member)
	for name, member := range c.members {
		key := c.hasher.Sum64([]byte(name))
		if name == owner.String() {
			ownerKey = key
		}
		keys = append(keys, key)
		kmems[key] = member
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	// 找到键的所有者
	idx := 0
	for idx < len(keys) {
		if keys[idx] == ownerKey {
			key := keys[idx]
			res = append(res, kmems[key])
			break
		}
		idx++
	}

	// 找到最接近的（副本所有者）成员
	for len(res) < count {
		idx++
		if idx >= len(keys) {
			idx = 0
		}
		key := keys[idx]
		res = append(res, kmems[key])
	}
	return res, nil
}

// GetClosestN 返回哈希环中最接近键的N个成员
// 这对于查找复制成员可能很有用
func (c *Consistent) GetClosestN(key []byte, count int) ([]Member, error) {
	partID := c.FindPartitionID(key)
	return c.getClosestN(partID, count)
}

// GetClosestNForPartition 返回给定分区最接近的N个成员
// 这对于查找复制成员可能很有用
func (c *Consistent) GetClosestNForPartition(partID, count int) ([]Member, error) {
	return c.getClosestN(partID, count)
}

// addToRing 只将成员添加到哈希环中（不重新分布分区）
func (c *Consistent) addToRing(member Member) {
	for i := 0; i < c.config.ReplicationFactor; i++ {
		key := []byte(fmt.Sprintf("%s%d", member.String(), i))
		h := c.hasher.Sum64(key)
		c.ring[h] = member
		c.sortedSet = append(c.sortedSet, h)
	}
	// 按升序排序哈希值
	sort.Slice(c.sortedSet, func(i int, j int) bool {
		return c.sortedSet[i] < c.sortedSet[j]
	})
}

// calculatePartitionsWithNewMember 计算添加新成员后的分区分布
func (c *Consistent) calculatePartitionsWithNewMember(newMember Member) (map[int]Member, map[string]float64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 创建临时的环和排序集合
	tempRing := make(map[uint64]Member)
	tempSortedSet := make([]uint64, len(c.sortedSet))
	copy(tempSortedSet, c.sortedSet)

	// 复制现有成员到临时环
	for k, v := range c.ring {
		tempRing[k] = v
	}

	// 添加新成员到临时环
	for i := 0; i < c.config.ReplicationFactor; i++ {
		key := []byte(fmt.Sprintf("%s%d", newMember.String(), i))
		h := c.hasher.Sum64(key)
		tempRing[h] = newMember
		tempSortedSet = append(tempSortedSet, h)
	}

	// 排序临时集合
	sort.Slice(tempSortedSet, func(i int, j int) bool {
		return tempSortedSet[i] < tempSortedSet[j]
	})

	// 计算新的分区分布，传入正确的成员数量
	newMemberCount := len(c.members) + 1
	return c.calculatePartitionsWithRingAndMemberCount(tempRing, tempSortedSet, newMemberCount)
}

// removeFromRing 只从哈希环中移除成员（不重新分布分区）
func (c *Consistent) removeFromRing(name string) {
	for i := 0; i < c.config.ReplicationFactor; i++ {
		key := []byte(fmt.Sprintf("%s%d", name, i))
		h := c.hasher.Sum64(key)
		delete(c.ring, h)
		c.delSlice(h)
	}
}

// calculatePartitionsWithoutMember 计算移除成员后的分区分布（优化版本）
func (c *Consistent) calculatePartitionsWithoutMember(memberName string) (map[int]Member, map[string]float64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 预计算要删除成员的所有哈希值，存入map以便快速查找
	hashesToDelete := make(map[uint64]struct{})
	for i := 0; i < c.config.ReplicationFactor; i++ {
		key := []byte(fmt.Sprintf("%s%d", memberName, i))
		h := c.hasher.Sum64(key)
		hashesToDelete[h] = struct{}{}
	}

	// 创建临时的环和排序集合
	tempRing := make(map[uint64]Member)
	var tempSortedSet []uint64

	// 只遍历一次c.ring，使用map快速检查是否需要删除
	for k, v := range c.ring {
		if _, shouldDelete := hashesToDelete[k]; !shouldDelete {
			// 这个节点需要保留
			tempRing[k] = v
			tempSortedSet = append(tempSortedSet, k)
		}
	}

	// 排序临时集合
	sort.Slice(tempSortedSet, func(i int, j int) bool {
		return tempSortedSet[i] < tempSortedSet[j]
	})

	// 计算新的分区分布
	return c.calculatePartitionsWithRingAndMemberCount(tempRing, tempSortedSet, len(c.members)-1)
}

// calculatePartitionsWithRingAndMemberCount 使用给定的环和成员数量计算分区分布
func (c *Consistent) calculatePartitionsWithRingAndMemberCount(ring map[uint64]Member, sortedSet []uint64, memberCount int) (map[int]Member, map[string]float64) {
	loads := make(map[string]float64)
	partitions := make(map[int]Member)

	if memberCount == 0 {
		return partitions, loads
	}

	// 计算平均负载
	avgLoad := float64(c.partitionCount/uint64(memberCount)) * c.config.Load
	avgLoad = math.Ceil(avgLoad)

	bs := make([]byte, 8)
	for partID := uint64(0); partID < c.partitionCount; partID++ {
		binary.LittleEndian.PutUint64(bs, partID)
		key := c.hasher.Sum64(bs)
		idx := sort.Search(len(sortedSet), func(i int) bool {
			return sortedSet[i] >= key
		})
		if idx >= len(sortedSet) {
			idx = 0
		}

		// 分配分区，考虑负载均衡
		var count int
		for {
			count++
			if count >= len(sortedSet) {
				panic("没有足够的空间来分布分区")
			}
			i := sortedSet[idx]
			member := ring[i]
			load := loads[member.String()]
			if load+1 <= avgLoad {
				partitions[int(partID)] = member
				loads[member.String()]++
				break
			}
			idx++
			if idx >= len(sortedSet) {
				idx = 0
			}
		}
	}

	return partitions, loads
}
