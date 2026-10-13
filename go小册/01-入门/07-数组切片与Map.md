# 07 · 数组、切片与 Map

这是 Go 里最常用的三件容器。数组几乎只在需要固定长度或底层存储时出现；日常数据集合用切片；键值用 map。把**切片的底层数组、长度、容量**搞懂，等于避开一半新手 bug。

---

## 1. 数组：值类型，长度是类型的一部分

```go
var a [3]int          // [0 0 0]
b := [3]int{1, 2, 3}
c := [...]int{4, 5}   // [2]int
```

`[3]int` 和 `[4]int` 是不同类型。赋值、传参都是**拷贝整个数组**。

```go
func zero(a [3]int) { a[0] = 0 }

x := [3]int{1, 2, 3}
zero(x)
fmt.Println(x[0]) // 仍是 1
```

要改原数组，传指针 `*[3]int`，或直接用切片。数组适合：哈希、固定协议头、作为切片的后备存储。

---

## 2. 切片：三元组

切片运行时可以想成：

```text
ptr  → 底层数组的某个位置
len  → 从 ptr 起可见多少元素
cap  → 从 ptr 起最多能长到多少（不换数组的前提下）
```

```go
s := []int{1, 2, 3, 4, 5}
t := s[1:4]          // [2 3 4]，len=3，cap=4（到原数组末尾）
fmt.Println(len(t), cap(t))
```

`make`：

```go
s := make([]int, 5)     // len=5, cap=5，元素是零值
s := make([]int, 0, 8)  // len=0, cap=8，预留空间
```

字面量 `[]int{1, 2}` 既分配数组也做出切片。

### 2.1 append 可能换底层数组

```go
a := []int{1, 2, 3}
b := a[:2]          // [1 2]，与 a 共享数组
b = append(b, 99)
fmt.Println(a)      // [1 2 99] —— 可能被改到！
```

若 `len == cap`，再 `append` 会分配**新数组**，旧切片看不到新元素。规则：

- 有剩余容量：原地写，可能改到别的切片。
- 容量不够：换新数组，通常按约 2 倍扩（大切片增长率会下降）。

**切片当返回值、或多个切片共享同一数组时，优先 `append` 到新切片或 `copy` 出独立副本。**

```go
func clone(s []int) []int {
	out := make([]int, len(s))
	copy(out, s)
	return out
}
```

### 2.2 nil 切片 vs 空切片

```go
var a []int          // nil，len=0, cap=0
b := []int{}         // 非 nil，len=0
c := make([]int, 0)  // 非 nil，len=0
```

`range`、`len`、`append` 对三者都安全。差别主要在 JSON：`nil` 编成 `null`，空切片编成 `[]`。API 里要稳定输出 `[]` 时，初始化成空切片而不是留下 nil。

### 2.3 删除、插入、截取

```go
// 删下标 i（不保序，O(1)）
s[i] = s[len(s)-1]
s = s[:len(s)-1]

// 删下标 i（保序）
s = append(s[:i], s[i+1:]...)

// 完整截取，避免残留容量导致内存不释放
s = s[:n:n] // len=n, cap=n
```

三下标切片 `s[low:high:max]` 把 cap 限制到 `max-low`，是防止「子切片 append 污染原切片」的利器。

### 2.4 1.21+ 的 slices 包

```go
import "slices"

slices.Contains(s, 3)
slices.Index(s, 3)
slices.Sort(s)
s = slices.Clone(s)
s = slices.Delete(s, 1, 3)
s = slices.Compact(s) // 去掉相邻重复
```

能用标准库就不要手写易错的删插。

---

## 3. Map：无序哈希表

```go
m := make(map[string]int)
m["a"] = 1

n := map[string]int{"a": 1, "b": 2}

v, ok := m["a"] // comma-ok，区分「没有」和「值为 0」
delete(m, "a")
```

规则：

- 键必须可比较（不能是切片、map、函数）。
- **零值 nil map 可读不可写**：`var m map[string]int; fmt.Println(m["x"])` 得 0；`m["x"]=1` panic。
- 遍历顺序随机。测试和序列化需要稳定顺序时，先把 key 提出来排序。
- 并发读写一个 map **必须同步**，否则可能直接崩溃。`sync.Map` 或外加 Mutex，见并发篇。

`make(map[K]V, hint)` 的第二个参数是容量提示，减少扩容。

```go
clear(m) // 1.21+，清空但复用内存
```

取指针当值要小心：

```go
m := map[string]int{"a": 1}
// p := &m["a"]  // 非法，map 中的值不可寻址
```

需要改复杂值时，用指针类型：`map[string]*User`，或取出、改、再写回。

---

## 4. 传参成本

| 类型 | 传参 | 含义 |
|------|------|------|
| 数组 | 拷贝全部元素 | 大数组不要直接传 |
| 切片 | 拷贝三元组（ptr/len/cap） | 改元素可见，append 换数组则调用方看不到新切片，必须返回 |
| map | 拷贝 map 头（指向同一哈希表） | 增删改对调用方可见 |

```go
func add(s []int) { s = append(s, 1) } // 调用方的 s 不变
func addPtr(s *[]int) { *s = append(*s, 1) }
func addRet(s []int) []int { return append(s, 1) } // 更地道
```

---

## 5. 套娃与常见结构

```go
graph := map[string][]string{
	"A": {"B", "C"},
	"B": {"C"},
}

// 分组
groups := make(map[string][]User)
for _, u := range users {
	groups[u.Role] = append(groups[u.Role], u)
}
```

`append` 到 nil 切片完全合法，所以分组时不必先 `make` 内层切片。

---

## 6. 一套对照

| 问题 | 数组 | 切片 | map |
|------|------|------|-----|
| 零值 | 元素全零 | nil，能 append | nil，能读不能写 |
| 相等 | 可比较 | 只能和 nil 比 | 只能和 nil 比 |
| 顺序 | 有 | 有 | 无 |
| 扩容 | 不能 | append | 自动 |
| JSON nil | 固定长 | `null` | `null` |

---

## 想一想

1. 为什么 `append` 必须写成 `s = append(s, x)`？
2. 子切片 `s[1:2]` 再 append，什么条件下会改到原来的 `s`？
3. 把切片当 map 的键为什么编不过？若需要「用一组数当键」怎么办？

## 练习

1. 实现 `func Dedup(s []string) []string`，保持首次出现顺序。
2. 实现一个不会「污染原切片」的 `func First(s []int, n int) []int`（限制 cap 或 clone）。
3. 统计一段文本里单词频率，输出出现次数最高的前 5 个（注意 map 无序）。

对应示例：`codes/02-slice-map`。

下一章：[字符串、字节与 rune](./08-字符串字节与rune.md)
