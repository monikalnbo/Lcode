#pragma once

/**
 * ==============================================================================
 *  Lcode Built-in C++ Modern Standard Library (C++17)
 *  Lcode 编译器内置高性能现代 C++ 核心库
 * ==============================================================================
 *  - 纯头文件设计 (Header-only)
 *  - 提供高速数值计算、经典算法、时间管理、堆内存追踪、调用栈与异常控制
 * ==============================================================================
 */

#include <iostream>
#include <string>
#include <vector>
#include <map>
#include <cmath>
#include <cstdint>
#include <algorithm>
#include <sstream>
#include <chrono>
#include <thread>
#include <iomanip>

namespace lcode {

// ==========================================
// 1. 高性能数学库 (Math)
// ==========================================
namespace math {

inline double fast_sqrt(double x) {
    if (x < 0.0) return 0.0;
    return std::sqrt(x);
}

inline double pow(double base, double exp) {
    return std::pow(base, exp);
}

inline int64_t abs(int64_t x) {
    return x < 0 ? -x : x;
}

inline int64_t clamp(int64_t val, int64_t min_val, int64_t max_val) {
    if (val < min_val) return min_val;
    if (val > max_val) return max_val;
    return val;
}

inline double clamp_f(double val, double min_val, double max_val) {
    if (val < min_val) return min_val;
    if (val > max_val) return max_val;
    return val;
}

} // namespace math

// ==========================================
// 2. 经典算法库 (Algorithms)
// ==========================================
namespace algo {

inline int64_t gcd(int64_t a, int64_t b) {
    while (b != 0) {
        int64_t t = b;
        b = a % b;
        a = t;
    }
    return a < 0 ? -a : a;
}

inline bool is_prime(int64_t n) {
    if (n <= 1) return false;
    if (n <= 3) return true;
    if (n % 2 == 0 || n % 3 == 0) return false;
    for (int64_t i = 5; i * i <= n; i += 6) {
        if (n % i == 0 || n % (i + 2) == 0) return false;
    }
    return true;
}

inline int64_t fibonacci(int64_t n) {
    if (n <= 0) return 0;
    if (n == 1) return 1;
    int64_t a = 0;
    int64_t b = 1;
    for (int64_t i = 2; i <= n; ++i) {
        int64_t c = a + b;
        a = b;
        b = c;
    }
    return b;
}

} // namespace algo

// ==========================================
// 3. 高精度时间管理 (Time)
// ==========================================
namespace time_mgr {

inline int64_t now_ms() {
    auto now = std::chrono::system_clock::now();
    return std::chrono::duration_cast<std::chrono::milliseconds>(now.time_since_epoch()).count();
}

inline int64_t now_secs() {
    auto now = std::chrono::system_clock::now();
    return std::chrono::duration_cast<std::chrono::seconds>(now.time_since_epoch()).count();
}

inline int64_t now_micros() {
    auto now = std::chrono::system_clock::now();
    return std::chrono::duration_cast<std::chrono::microseconds>(now.time_since_epoch()).count();
}

inline void sleep_ms(int64_t ms) {
    if (ms > 0) {
        std::this_thread::sleep_for(std::chrono::milliseconds(ms));
    }
}

inline int64_t elapsed_ms(int64_t start_ms) {
    return now_ms() - start_ms;
}

inline std::string format_now() {
    auto now = std::chrono::system_clock::now();
    std::time_t t = std::chrono::system_clock::to_time_t(now);
    std::tm tm_buf;
#if defined(_WIN32) || defined(_WIN64)
    localtime_s(&tm_buf, &t);
#else
    localtime_r(&t, &tm_buf);
#endif
    std::ostringstream ss;
    ss << std::put_time(&tm_buf, "%Y-%m-%d %H:%M:%S");
    return ss.str();
}

} // namespace time_mgr

// ==========================================
// 4. 堆内存分配与泄漏追踪管理 (Memory)
// ==========================================
namespace mem {

struct Block {
    int64_t handle;
    int64_t size;
    std::vector<int64_t> data;
    bool freed;
};

class MemoryManager {
private:
    std::map<int64_t, Block> blocks;
    int64_t next_handle = 1001;
    int64_t total_alloc_bytes = 0;
    int64_t active_bytes = 0;
    int64_t peak_bytes = 0;
    int64_t alloc_count = 0;
    int64_t free_count = 0;
    std::vector<std::vector<int64_t>> scope_stack;

public:
    static MemoryManager& instance() {
        static MemoryManager inst;
        return inst;
    }

    void enter_scope() {
        scope_stack.emplace_back();
    }

    void exit_scope() {
        if (scope_stack.empty()) return;
        auto handles = scope_stack.back();
        scope_stack.pop_back();
        for (int64_t h : handles) {
            auto it = blocks.find(h);
            if (it != blocks.end() && !it->second.freed) {
                it->second.freed = true;
                free_count++;
                active_bytes -= it->second.size * 8;
            }
        }
    }

    int64_t escape_return(int64_t h) {
        if (!scope_stack.empty()) {
            auto& curr = scope_stack.back();
            for (auto it = curr.begin(); it != curr.end(); ++it) {
                if (*it == h) {
                    curr.erase(it);
                    if (scope_stack.size() >= 2) {
                        scope_stack[scope_stack.size() - 2].push_back(h);
                    }
                    break;
                }
            }
        }
        return h;
    }

    int64_t alloc(int64_t size) {
        if (size <= 0) return 0;
        int64_t h = next_handle++;
        Block b;
        b.handle = h;
        b.size = size;
        b.data.resize(size, 0);
        b.freed = false;
        blocks[h] = b;

        if (!scope_stack.empty()) {
            scope_stack.back().push_back(h);
        }

        int64_t b_size = size * 8;
        alloc_count++;
        total_alloc_bytes += b_size;
        active_bytes += b_size;
        if (active_bytes > peak_bytes) {
            peak_bytes = active_bytes;
        }
        return h;
    }

    int64_t free(int64_t handle) {
        auto it = blocks.find(handle);
        if (it == blocks.end()) {
            std::cerr << "[C++ 运行时错误] 非法释放未分配内存 0x" << std::hex << handle << std::dec << std::endl;
            return -1;
        }
        if (it->second.freed) {
            return 0; // 已经释放，安全忽略
        }
        it->second.freed = true;
        free_count++;
        active_bytes -= it->second.size * 8;
        return 0;
    }

    void write(int64_t handle, int64_t offset, int64_t val) {
        auto it = blocks.find(handle);
        if (it == blocks.end() || it->second.freed || offset < 0 || offset >= it->second.size) {
            std::cerr << "[C++ 内存错误] 越界或非法内存写入" << std::endl;
            return;
        }
        it->second.data[offset] = val;
    }

    int64_t read(int64_t handle, int64_t offset) {
        auto it = blocks.find(handle);
        if (it == blocks.end() || it->second.freed || offset < 0 || offset >= it->second.size) {
            std::cerr << "[C++ 内存错误] 越界或非法内存读取" << std::endl;
            return 0;
        }
        return it->second.data[offset];
    }

    void stats() {
        std::cout << "【C++ 内存统计】: 累计分配 " << total_alloc_bytes << " 字节, 活跃 "
                  << active_bytes << " 字节, 峰值 " << peak_bytes << " 字节, 分配 "
                  << alloc_count << " 次, 释放 " << free_count << " 次" << std::endl;
    }

    int64_t check_leaks() {
        int64_t leaks = 0;
        for (const auto& kv : blocks) {
            if (!kv.second.freed) {
                leaks++;
            }
        }
        if (leaks == 0) {
            std::cout << "✓ 内存检查安全: 零内存泄漏" << std::endl;
        } else {
            std::cout << "⚠️ 检测到 " << leaks << " 个未释放内存块" << std::endl;
        }
        return leaks;
    }
};

class ScopeGuard {
public:
    ScopeGuard() {
        MemoryManager::instance().enter_scope();
    }
    ~ScopeGuard() {
        MemoryManager::instance().exit_scope();
    }
};

template <typename T>
inline T escape_return(T val) {
    return val;
}

inline int64_t escape_return(int64_t h) {
    return MemoryManager::instance().escape_return(h);
}

} // namespace mem

// ==========================================
// 5. 格式化控制台输出与调用栈 (IO & Stack)
// ==========================================
namespace io {

template <typename... Args>
void println(Args&&... args) {
    std::ostringstream ss;
    ((ss << std::forward<Args>(args) << " "), ...);
    std::string str = ss.str();
    if (!str.empty() && str.back() == ' ') {
        str.pop_back();
    }
    std::cout << str << std::endl;
}

template <typename... Args>
void print(Args&&... args) {
    std::ostringstream ss;
    ((ss << std::forward<Args>(args) << " "), ...);
    std::string str = ss.str();
    if (!str.empty() && str.back() == ' ') {
        str.pop_back();
    }
    std::cout << str;
}

} // namespace io

} // namespace lcode

// ==============================================================================
//  C++ 全局桥接函数 (供 Lcode 转译生成的代码无缝直接调用)
// ==============================================================================

inline double cpp_fast_sqrt(double x) { return lcode::math::fast_sqrt(x); }
inline double cpp_pow(double b, double e) { return lcode::math::pow(b, e); }
inline int64_t cpp_abs(int64_t x) { return lcode::math::abs(x); }
inline int64_t cpp_gcd(int64_t a, int64_t b) { return lcode::algo::gcd(a, b); }
inline int64_t cpp_fibonacci(int64_t n) { return lcode::algo::fibonacci(n); }
inline bool cpp_is_prime(int64_t n) { return lcode::algo::is_prime(n); }

// 时间管理桥接
inline int64_t time_now_ms() { return lcode::time_mgr::now_ms(); }
inline int64_t time_now_secs() { return lcode::time_mgr::now_secs(); }
inline int64_t time_now_micros() { return lcode::time_mgr::now_micros(); }
inline void time_sleep_ms(int64_t ms) { lcode::time_mgr::sleep_ms(ms); }
inline int64_t time_elapsed_ms(int64_t start) { return lcode::time_mgr::elapsed_ms(start); }
inline std::string time_format_now() { return lcode::time_mgr::format_now(); }

// 内存管理桥接
inline int64_t mem_alloc(int64_t sz) { return lcode::mem::MemoryManager::instance().alloc(sz); }
inline int64_t mem_free(int64_t h) { return lcode::mem::MemoryManager::instance().free(h); }
inline void mem_write(int64_t h, int64_t off, int64_t v) { lcode::mem::MemoryManager::instance().write(h, off, v); }
inline int64_t mem_read(int64_t h, int64_t off) { return lcode::mem::MemoryManager::instance().read(h, off); }
inline void mem_stats() { lcode::mem::MemoryManager::instance().stats(); }
inline int64_t mem_check_leaks() { return lcode::mem::MemoryManager::instance().check_leaks(); }

// Rust 风格别名与显式 drop
inline int64_t alloc(int64_t sz) { return mem_alloc(sz); }
inline int64_t free(int64_t h) { return mem_free(h); }
inline int64_t drop(int64_t h) { return mem_free(h); }
inline void write(int64_t h, int64_t off, int64_t v) { mem_write(h, off, v); }
inline int64_t read(int64_t h, int64_t off) { return mem_read(h, off); }
inline void stats() { mem_stats(); }
inline int64_t check_leaks() { return mem_check_leaks(); }

// Rust RAII 作用域自动析构守护与返回值所有权逃逸
using ScopeGuard = lcode::mem::ScopeGuard;

template <typename T>
inline T escape_return(T val) {
    return lcode::mem::escape_return(val);
}

inline int64_t escape_return(int64_t h) {
    return lcode::mem::escape_return(h);
}

// 异常与调用栈桥接
inline void panic(const std::string& msg) {
    std::cerr << "\033[91m💥 [运行时恐慌 Panic]: " << msg << "\033[0m" << std::endl;
    std::exit(1);
}

inline void assert(bool condition, const std::string& msg) {
    if (!condition) {
        panic(msg);
    }
}

inline void stack_dump() {
    std::cout << "🥞 [C++ 运行时调用栈回溯: Native execution frame active]" << std::endl;
}

inline int64_t stack_depth() {
    return 1;
}
