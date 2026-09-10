package eval

import (
	"fmt"
)

// MemoryBlock 虚拟堆内存块
type MemoryBlock struct {
	Handle    int64   // 内存块唯一句柄
	Size      int     // 块大小 (字长元素数量)
	Data      []int64 // 数据缓冲区
	Freed     bool    // 是否已被释放
	CallerFn  string  // 分配发生时的函数名
	Line      int     // 分配发生行号
}

// MemoryStats 内存统计指标
type MemoryStats struct {
	TotalAllocatedBytes int64 // 累计分配字节数 (每个元素 8 字节)
	ActiveBytes         int64 // 当前活跃内存字节数
	PeakBytes           int64 // 峰值占用内存字节数
	AllocCount          int   // 累计分配次数
	FreeCount           int   // 累计释放次数
}

// MemoryManager 运行时虚拟内存管理器
type MemoryManager struct {
	blocks      map[int64]*MemoryBlock
	nextHandle  int64
	stats       MemoryStats
}

// NewMemoryManager 创建内存管理器
func NewMemoryManager() *MemoryManager {
	return &MemoryManager{
		blocks:     make(map[int64]*MemoryBlock),
		nextHandle: 1001, // 句柄从 1001 开始分配，避免 0 空指针
	}
}

// Alloc 分配指定大小的内存块
func (mm *MemoryManager) Alloc(size int, callerFn string, line int) (int64, error) {
	if size <= 0 {
		return 0, fmt.Errorf("内存分配错误: 申请分配的大小必须大于 0 (传入大小: %d)", size)
	}

	handle := mm.nextHandle
	mm.nextHandle++

	block := &MemoryBlock{
		Handle:   handle,
		Size:     size,
		Data:     make([]int64, size),
		Freed:    false,
		CallerFn: callerFn,
		Line:     line,
	}

	mm.blocks[handle] = block

	bytesAlloc := int64(size * 8)
	mm.stats.AllocCount++
	mm.stats.TotalAllocatedBytes += bytesAlloc
	mm.stats.ActiveBytes += bytesAlloc
	if mm.stats.ActiveBytes > mm.stats.PeakBytes {
		mm.stats.PeakBytes = mm.stats.ActiveBytes
	}

	return handle, nil
}

// Free 释放指定句柄的内存块
func (mm *MemoryManager) Free(handle int64) error {
	block, ok := mm.blocks[handle]
	if !ok {
		return fmt.Errorf("非法内存释放 (Invalid Free): 句柄 0x%X 未曾分配", handle)
	}
	if block.Freed {
		return fmt.Errorf("内存重复释放 (Double Free): 句柄 0x%X 已经释放过，请勿重复释放", handle)
	}

	block.Freed = true
	freedBytes := int64(block.Size * 8)
	mm.stats.FreeCount++
	mm.stats.ActiveBytes -= freedBytes

	return nil
}

// Write 向内存块指定偏移写入值
func (mm *MemoryManager) Write(handle int64, offset int, value int64) error {
	block, ok := mm.blocks[handle]
	if !ok {
		return fmt.Errorf("内存段错误 (Segmentation Fault): 无法访问未分配的内存句柄 0x%X", handle)
	}
	if block.Freed {
		return fmt.Errorf("释放后使用错误 (Use After Free): 句柄 0x%X 已被释放，禁止写入", handle)
	}
	if offset < 0 || offset >= block.Size {
		return fmt.Errorf("内存越界写入 (Out of Bounds): 句柄 0x%X 大小为 %d，尝试写入偏移量 %d", handle, block.Size, offset)
	}

	block.Data[offset] = value
	return nil
}

// Read 从内存块指定偏移读取值
func (mm *MemoryManager) Read(handle int64, offset int) (int64, error) {
	block, ok := mm.blocks[handle]
	if !ok {
		return 0, fmt.Errorf("内存段错误 (Segmentation Fault): 无法访问未分配的内存句柄 0x%X", handle)
	}
	if block.Freed {
		return 0, fmt.Errorf("释放后使用错误 (Use After Free): 句柄 0x%X 已被释放，禁止读取", handle)
	}
	if offset < 0 || offset >= block.Size {
		return 0, fmt.Errorf("内存越界读取 (Out of Bounds): 句柄 0x%X 大小为 %d，尝试读取偏移量 %d", handle, block.Size, offset)
	}

	return block.Data[offset], nil
}

// Stats 获取内存统计数据
func (mm *MemoryManager) Stats() MemoryStats {
	return mm.stats
}

// CheckLeaks 检查未释放的内存泄漏块
func (mm *MemoryManager) CheckLeaks() []*MemoryBlock {
	leaks := make([]*MemoryBlock, 0)
	for _, b := range mm.blocks {
		if !b.Freed {
			leaks = append(leaks, b)
		}
	}
	return leaks
}
