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
#include <fstream>
#include <random>

template <typename T>
inline std::ostream& operator<<(std::ostream& os, const std::vector<T>& v) {
    os << "[";
    for (size_t i = 0; i < v.size(); ++i) {
        if (i > 0) os << ", ";
        os << v[i];
    }
    os << "]";
    return os;
}

inline std::ostream& operator<<(std::ostream& os, const std::exception& e) {
    os << e.what();
    return os;
}

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
    ss << std::boolalpha;
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
    ss << std::boolalpha;
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
    throw std::runtime_error(msg);
}

inline int64_t lcode_div(int64_t a, int64_t b) {
    if (b == 0) throw std::runtime_error("除以零错误");
    return a / b;
}

inline int64_t lcode_mod(int64_t a, int64_t b) {
    if (b == 0) throw std::runtime_error("除以零错误");
    return a % b;
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

// ==========================================
// 6. 后端引擎支持：动态数组、切片与容器操作
// ==========================================
inline std::vector<int64_t> range(int64_t end) {
    std::vector<int64_t> res;
    if (end > 0) {
        res.reserve(end);
        for (int64_t i = 0; i < end; ++i) res.push_back(i);
    }
    return res;
}

inline std::vector<int64_t> range(int64_t start, int64_t end) {
    std::vector<int64_t> res;
    if (end > start) {
        res.reserve(end - start);
        for (int64_t i = start; i < end; ++i) res.push_back(i);
    }
    return res;
}

template <typename T>
inline int64_t len(const std::vector<T>& v) {
    return static_cast<int64_t>(v.size());
}

inline int64_t len(const std::string& s) {
    return static_cast<int64_t>(s.size());
}

template <typename T, typename U>
inline std::vector<T>& push(std::vector<T>& v, const U& item) {
    v.push_back(static_cast<T>(item));
    return v;
}

template <typename T, typename U>
inline std::vector<T>& append(std::vector<T>& v, const U& item) {
    v.push_back(static_cast<T>(item));
    return v;
}

template <typename T>
inline T pop(std::vector<T>& v) {
    if (v.empty()) {
        panic("pop from empty vector");
    }
    T val = v.back();
    v.pop_back();
    return val;
}

// ==========================================
// 7. 后端引擎支持：文件系统操作 (std/fs)
// ==========================================
inline std::string fs_read_text(const std::string& path) {
    std::ifstream ifs(path);
    if (!ifs.is_open()) {
        panic("failed to open file for read: " + path);
    }
    std::ostringstream ss;
    ss << ifs.rdbuf();
    return ss.str();
}

inline bool fs_write_text(const std::string& path, const std::string& text) {
    std::ofstream ofs(path);
    if (!ofs.is_open()) {
        return false;
    }
    ofs << text;
    return true;
}

inline bool fs_exists(const std::string& path) {
    std::ifstream ifs(path);
    return ifs.good();
}

inline bool fs_remove(const std::string& path) {
    return std::remove(path.c_str()) == 0;
}

// ==========================================
// 8. 后端引擎支持：字符串处理 (std/str)
// ==========================================
inline int64_t str_len(const std::string& s) {
    return static_cast<int64_t>(s.size());
}

inline bool str_contains(const std::string& s, const std::string& sub) {
    return s.find(sub) != std::string::npos;
}

inline std::vector<std::string> str_split(const std::string& s, const std::string& delim) {
    std::vector<std::string> tokens;
    if (delim.empty()) {
        for (char c : s) tokens.push_back(std::string(1, c));
        return tokens;
    }
    size_t prev = 0, pos = 0;
    while ((pos = s.find(delim, prev)) != std::string::npos) {
        tokens.push_back(s.substr(prev, pos - prev));
        prev = pos + delim.length();
    }
    tokens.push_back(s.substr(prev));
    return tokens;
}

inline std::string str_trim(const std::string& s) {
    auto start = s.find_first_not_of(" \t\n\r");
    if (start == std::string::npos) return "";
    auto end = s.find_last_not_of(" \t\n\r");
    return s.substr(start, end - start + 1);
}

inline std::string str_upper(std::string s) {
    for (auto& c : s) c = static_cast<char>(std::toupper(c));
    return s;
}

inline std::string str_lower(std::string s) {
    for (auto& c : s) c = static_cast<char>(std::tolower(c));
    return s;
}

// ==========================================
// 9. 后端引擎支持：随机数生成 (std/random)
// ==========================================
inline int64_t random_int(int64_t min_v, int64_t max_v) {
    if (max_v <= min_v) return min_v;
    static std::mt19937_64 rng(std::random_device{}());
    std::uniform_int_distribution<int64_t> dist(min_v, max_v);
    return dist(rng);
}

inline double random_float() {
    static std::mt19937_64 rng(std::random_device{}());
    std::uniform_real_distribution<double> dist(0.0, 1.0);
    return dist(rng);
}

