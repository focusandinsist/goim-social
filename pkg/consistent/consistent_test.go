package consistent

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestConsistentHash 测试一致性哈希基本功能
func TestConsistentHash(t *testing.T) {
	// 创建网关成员
	members := []Member{
		NewGatewayMember("gateway-1", "192.168.1.1", 8080),
		NewGatewayMember("gateway-2", "192.168.1.2", 8080),
		NewGatewayMember("gateway-3", "192.168.1.3", 8080),
	}

	// 创建配置
	config := Config{
		Hasher:            NewCRC64Hasher(),
		PartitionCount:    271,
		ReplicationFactor: 20,
		Load:              1.25,
	}

	// 创建一致性哈希环
	ring := New(members, config)

	// 测试键定位
	testKeys := []string{
		"user:1001",
		"user:1002",
		"user:1003",
		"room:2001",
		"room:2002",
	}

	fmt.Println("键分布测试:")
	for _, key := range testKeys {
		member := ring.LocateKey([]byte(key))
		if member != nil {
			fmt.Printf("键 %s -> 成员 %s\n", key, member.String())
		}
	}

	// 测试负载分布
	fmt.Println("\n负载分布:")
	loadDist := ring.LoadDistribution()
	for member, load := range loadDist {
		fmt.Printf("成员 %s: 负载 %.2f\n", member, load)
	}

	// 测试添加成员
	fmt.Println("\n添加新成员...")
	newMember := NewGatewayMember("gateway-4", "192.168.1.4", 8080)
	ring.Add(newMember)

	// 再次测试键分布
	fmt.Println("\n添加成员后的键分布:")
	for _, key := range testKeys {
		member := ring.LocateKey([]byte(key))
		if member != nil {
			fmt.Printf("键 %s -> 成员 %s\n", key, member.String())
		}
	}

	// 测试移除成员
	fmt.Println("\n移除成员...")
	ring.Remove("gateway-2:192.168.1.2:8080")

	// 再次测试键分布
	fmt.Println("\n移除成员后的键分布:")
	for _, key := range testKeys {
		member := ring.LocateKey([]byte(key))
		if member != nil {
			fmt.Printf("键 %s -> 成员 %s\n", key, member.String())
		}
	}

	// 测试获取最接近的N个成员
	fmt.Println("\n获取最接近的成员:")
	closest, err := ring.GetClosestN([]byte("user:1001"), 2)
	if err != nil {
		t.Errorf("获取最接近成员失败: %v", err)
	} else {
		for i, member := range closest {
			fmt.Printf("第%d接近的成员: %s\n", i+1, member.String())
		}
	}
}

// TestConsistentHashWithFNV 测试使用FNV哈希的一致性哈希
func TestConsistentHashWithFNV(t *testing.T) {
	members := []Member{
		NewGatewayMember("gateway-1", "127.0.0.1", 8080),
		NewGatewayMember("gateway-2", "127.0.0.1", 8081),
	}

	config := Config{
		Hasher:            NewFNVHasher(),
		PartitionCount:    100,
		ReplicationFactor: 10,
		Load:              1.5,
	}

	ring := New(members, config)

	// 测试平均负载
	avgLoad := ring.AverageLoad()
	fmt.Printf("平均负载: %.2f\n", avgLoad)

	// 测试成员列表
	allMembers := ring.GetMembers()
	fmt.Printf("成员数量: %d\n", len(allMembers))
	for _, member := range allMembers {
		fmt.Printf("成员: %s\n", member.String())
	}
}

// BenchmarkLocateKey 基准测试键定位性能
func BenchmarkLocateKey(b *testing.B) {
	members := []Member{
		NewGatewayMember("gateway-1", "192.168.1.1", 8080),
		NewGatewayMember("gateway-2", "192.168.1.2", 8080),
		NewGatewayMember("gateway-3", "192.168.1.3", 8080),
		NewGatewayMember("gateway-4", "192.168.1.4", 8080),
		NewGatewayMember("gateway-5", "192.168.1.5", 8080),
	}

	config := Config{
		Hasher:            NewCRC64Hasher(),
		PartitionCount:    271,
		ReplicationFactor: 20,
		Load:              1.25,
	}

	ring := New(members, config)
	key := []byte("user:1001")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ring.LocateKey(key)
	}
}

// BenchmarkAddRemove 基准测试添加和移除成员的性能
func BenchmarkAddRemove(b *testing.B) {
	members := []Member{
		NewGatewayMember("gateway-1", "192.168.1.1", 8080),
		NewGatewayMember("gateway-2", "192.168.1.2", 8080),
	}

	config := Config{
		Hasher:            NewCRC64Hasher(),
		PartitionCount:    271,
		ReplicationFactor: 20,
		Load:              1.25,
	}

	ring := New(members, config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 添加成员
		newMember := NewGatewayMember(fmt.Sprintf("gateway-%d", i+3), "192.168.1.10", 8080)
		ring.Add(newMember)

		// 移除成员
		ring.Remove(newMember.String())
	}
}

// BenchmarkConcurrentRead 基准测试并发读取性能
func BenchmarkConcurrentRead(b *testing.B) {
	members := []Member{
		NewGatewayMember("gateway-1", "192.168.1.1", 8080),
		NewGatewayMember("gateway-2", "192.168.1.2", 8080),
		NewGatewayMember("gateway-3", "192.168.1.3", 8080),
		NewGatewayMember("gateway-4", "192.168.1.4", 8080),
		NewGatewayMember("gateway-5", "192.168.1.5", 8080),
	}

	config := Config{
		Hasher:            NewCRC64Hasher(),
		PartitionCount:    271,
		ReplicationFactor: 20,
		Load:              1.25,
	}

	ring := New(members, config)
	key := []byte("user:1001")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ring.LocateKey(key)
		}
	})
}

// BenchmarkGetMembers 基准测试获取成员列表的性能（测试缓存效果）
func BenchmarkGetMembers(b *testing.B) {
	members := []Member{
		NewGatewayMember("gateway-1", "192.168.1.1", 8080),
		NewGatewayMember("gateway-2", "192.168.1.2", 8080),
		NewGatewayMember("gateway-3", "192.168.1.3", 8080),
		NewGatewayMember("gateway-4", "192.168.1.4", 8080),
		NewGatewayMember("gateway-5", "192.168.1.5", 8080),
	}

	config := Config{
		Hasher:            NewCRC64Hasher(),
		PartitionCount:    271,
		ReplicationFactor: 20,
		Load:              1.25,
	}

	ring := New(members, config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ring.GetMembers()
	}
}

// BenchmarkLargeScale 大规模基准测试
func BenchmarkLargeScale(b *testing.B) {
	// 创建100个成员
	var members []Member
	for i := 0; i < 100; i++ {
		members = append(members, NewGatewayMember(
			fmt.Sprintf("gateway-%d", i),
			fmt.Sprintf("192.168.%d.%d", i/254+1, i%254+1),
			8080,
		))
	}

	config := Config{
		Hasher:            NewCRC64Hasher(),
		PartitionCount:    1000, // 更多分区
		ReplicationFactor: 50,   // 更多虚拟节点
		Load:              1.25,
	}

	ring := New(members, config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := []byte(fmt.Sprintf("user:%d", i))
		ring.LocateKey(key)
	}
}

// TestConcurrentSafety 测试并发安全性
func TestConcurrentSafety(t *testing.T) {
	members := []Member{
		NewGatewayMember("gateway-1", "192.168.1.1", 8080),
		NewGatewayMember("gateway-2", "192.168.1.2", 8080),
	}

	config := Config{
		Hasher:            NewCRC64Hasher(),
		PartitionCount:    271,
		ReplicationFactor: 20,
		Load:              1.25,
	}

	ring := New(members, config)

	// 并发测试GetMembers的race condition修复
	t.Run("GetMembers并发安全", func(t *testing.T) {
		const numGoroutines = 50
		const numCalls = 100

		var wg sync.WaitGroup
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < numCalls; j++ {
					members := ring.GetMembers()
					if len(members) == 0 {
						t.Errorf("GetMembers返回空列表")
					}
				}
			}()
		}
		wg.Wait()
	})

	// 并发读写测试
	t.Run("并发读写安全", func(t *testing.T) {
		const duration = 2 * time.Second
		const numReaders = 20
		const numWriters = 2

		var wg sync.WaitGroup
		stop := make(chan struct{})

		// 启动读取goroutine
		for i := 0; i < numReaders; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for {
					select {
					case <-stop:
						return
					default:
						key := fmt.Sprintf("user:%d", id)
						ring.LocateKey([]byte(key))
						ring.GetMembers()
					}
				}
			}(i)
		}

		// 启动写入goroutine
		for i := 0; i < numWriters; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				counter := 0
				for {
					select {
					case <-stop:
						return
					default:
						if counter%2 == 0 {
							member := NewGatewayMember(
								fmt.Sprintf("temp-%d-%d", id, counter),
								fmt.Sprintf("10.0.%d.%d", id, counter%254+1),
								8080,
							)
							ring.Add(member)
						} else {
							memberName := fmt.Sprintf("temp-%d-%d:10.0.%d.%d:8080",
								id, counter-1, id, (counter-1)%254+1)
							ring.Remove(memberName)
						}
						counter++
						time.Sleep(10 * time.Millisecond)
					}
				}
			}(i)
		}

		time.Sleep(duration)
		close(stop)
		wg.Wait()
	})
}
