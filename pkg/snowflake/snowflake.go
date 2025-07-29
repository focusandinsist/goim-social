package snowflake

import (
	"fmt"
	"sync"
	"time"
)

// Snowflake ID生成器
// 64位ID结构：1位符号位(0) + 41位时间戳 + 10位机器ID + 12位序列号
type Snowflake struct {
	mutex     sync.Mutex
	epoch     int64 // 起始时间戳 (毫秒)
	machineID int64 // 机器ID (0-1023)
	sequence  int64 // 序列号 (0-4095)
	lastTime  int64 // 上次生成ID的时间戳
}

const (
	// 各部分位数
	machineBits  = 10 // 机器ID位数
	sequenceBits = 12 // 序列号位数

	// 最大值
	maxMachineID = (1 << machineBits) - 1 // 1023
	maxSequence  = (1 << sequenceBits) - 1 // 4095

	// 位移
	machineShift  = sequenceBits                    // 12
	timestampShift = sequenceBits + machineBits     // 22

	// 自定义起始时间 (2024-01-01 00:00:00 UTC)
	defaultEpoch = 1704067200000
)

// NewSnowflake 创建Snowflake实例
func NewSnowflake(machineID int64) (*Snowflake, error) {
	if machineID < 0 || machineID > maxMachineID {
		return nil, fmt.Errorf("机器ID必须在0-%d之间", maxMachineID)
	}

	return &Snowflake{
		epoch:     defaultEpoch,
		machineID: machineID,
		sequence:  0,
		lastTime:  0,
	}, nil
}

// Generate 生成下一个ID
func (s *Snowflake) Generate() int64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now().UnixMilli()

	// 时钟回拨检查
	if now < s.lastTime {
		panic(fmt.Sprintf("时钟回拨，拒绝生成ID。当前时间: %d, 上次时间: %d", now, s.lastTime))
	}

	if now == s.lastTime {
		// 同一毫秒内，序列号递增
		s.sequence = (s.sequence + 1) & maxSequence
		if s.sequence == 0 {
			// 序列号溢出，等待下一毫秒
			for now <= s.lastTime {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		// 新的毫秒，序列号重置
		s.sequence = 0
	}

	s.lastTime = now

	// 组装ID: 时间戳部分 | 机器ID部分 | 序列号部分
	id := ((now - s.epoch) << timestampShift) |
		(s.machineID << machineShift) |
		s.sequence

	return id
}

// ParseID 解析Snowflake ID
func (s *Snowflake) ParseID(id int64) (timestamp int64, machineID int64, sequence int64) {
	timestamp = (id >> timestampShift) + s.epoch
	machineID = (id >> machineShift) & maxMachineID
	sequence = id & maxSequence
	return
}

// GetInfo 获取ID信息
func (s *Snowflake) GetInfo(id int64) string {
	timestamp, machineID, sequence := s.ParseID(id)
	t := time.UnixMilli(timestamp)
	return fmt.Sprintf("ID: %d, 时间: %s, 机器ID: %d, 序列号: %d",
		id, t.Format("2006-01-02 15:04:05.000"), machineID, sequence)
}

// 全局Snowflake实例
var globalSnowflake *Snowflake

// InitGlobalSnowflake 初始化全局Snowflake实例
func InitGlobalSnowflake(machineID int64) error {
	var err error
	globalSnowflake, err = NewSnowflake(machineID)
	return err
}

// GenerateID 生成全局唯一ID
func GenerateID() int64 {
	if globalSnowflake == nil {
		panic("Snowflake未初始化，请先调用InitGlobalSnowflake")
	}
	return globalSnowflake.Generate()
}

// ParseGlobalID 解析全局ID
func ParseGlobalID(id int64) string {
	if globalSnowflake == nil {
		panic("Snowflake未初始化，请先调用InitGlobalSnowflake")
	}
	return globalSnowflake.GetInfo(id)
}
