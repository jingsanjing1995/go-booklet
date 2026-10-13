# 08 · 字符串、字节与 rune

Go 的字符串是**不可变的字节序列**，源码默认按 UTF-8 理解。中文、emoji 会让 `len`、下标、`for range` 的行为对不上「我看见的字符数」。这一章把三层表示拆开。

---

## 1. 三层表示

```text
string   不可变 []byte 的只读视图
[]byte   可变字节切片
rune     int32，一个 Unicode 码点
```

```go
s := "Go小册"
fmt.Println(len(s))                    // 8：G o 各 1，小 3，册 3
fmt.Println(utf8.RuneCountInString(s)) // 4
fmt.Printf("%q\n", s[2])               // 不是「小」，而是 UTF-8 的第一个字节
```

用下标取的是**字节**。要按码点遍历：

```go
for i, r := range s {
	fmt.Printf("%d %c %U\n", i, r, r)
}
```

`i` 是该 rune 起始字节下标，不是「第几个字」。

---

## 2. 字面量与原始字符串

```go
a := "hello\n"
b := `第一行
第二行`           // 反引号：所见即所得，不能含未转义的反引号
c := "字符串里写 `代码` 可以"
```

`+` 拼接字符串会分配新对象。循环里拼大量碎片，请用 `strings.Builder`：

```go
var b strings.Builder
b.Grow(64)
for _, w := range words {
	b.WriteString(w)
	b.WriteByte(' ')
}
out := b.String()
```

`bytes.Buffer` 更偏通用 IO；纯拼字符串用 `strings.Builder`。

---

## 3. 转换成本

```go
b := []byte(s) // 复制
s2 := string(b) // 再复制
```

每次 `string` ↔ `[]byte` 通常都要拷贝（编译器在特定只读场景会优化，但不要依赖）。热路径上选定一种表示并坚持。

`[]rune(s)` 按码点拆开，再 `string(runes)` 拼回去。适合「改第 n 个字」：

```go
rs := []rune("Go小册")
rs[2] = '手'
fmt.Println(string(rs)) // Go手册
```

对非法 UTF-8，`range` 会产出 `0xFFFD`（替换符），`string([]byte{0xff})` 仍允许，因为 string 不强制合法 UTF-8。

---

## 4. strings / bytes / strconv / unicode

这四个包要能在脑子里索引。

```go
strings.HasPrefix(s, "Go")
strings.Split("a,b,c", ",")
strings.Join([]string{"a", "b"}, "-")
strings.ReplaceAll(s, "旧", "新")
strings.Fields("  a  b\tc") // 按空白切
strings.Cut(s, ":")         // 1.18+，比 SplitN 更直
strings.CutPrefix(s, "http://")

bytes.Contains(b, []byte("Go"))
bytes.TrimSpace(b)

n, err := strconv.Atoi("42")
s := strconv.FormatInt(42, 10)
b, err := strconv.ParseBool("true")

unicode.IsSpace(r)
unicode.ToLower(r)
```

按字节处理用 `bytes`，按字符串用 `strings`，API 几乎镜像。数字转换一律 `strconv`，不要 `fmt.Sprintf` 当热路径。

---

## 5. UTF-8 编解码

```go
r, size := utf8.DecodeRuneInString(s) // 第一个码点及字节数
buf := make([]byte, utf8.UTFMax)
n := utf8.EncodeRune(buf, '册')
```

校验：

```go
utf8.ValidString(s)
utf8.Valid(b)
```

网络协议、文件格式经常是「长度 + 原始字节」。先当 `[]byte` 处理，需要给人看时再变成 string。

---

## 6. 比较、哈希、可当 map 键

字符串可比较，按字节字典序。可以当 map 键、switch case。

```go
switch lang {
case "zh", "zh-CN":
default:
}
```

大段文本忽略大小写比较：`strings.EqualFold`。不要先双方 `ToLower` 再比，既分配又对某些 Unicode 不完整。

---

## 7. 和切片共享的坑

```go
s := "abcdef"
// s[0] = 'A'  // 非法，字符串不可变

b := []byte(s)
b[0] = 'A'
fmt.Println(s, string(b)) // s 仍是 abcdef
```

`s[i:j]` 子串与原串共享底层字节（只读）。极长字符串若只留一个短 sub-string，可能拖住整块内存。需要长期保存短片段时：

```go
func clone(s string) string {
	return strings.Clone(s) // 1.18+
}
```

---

## 8. 打印动词（和字符串强相关）

```go
fmt.Printf("%s\n", s)   // 原样
fmt.Printf("%q\n", s)   // 带引号、转义，调试首选
fmt.Printf("%x\n", s)   // 十六进制
fmt.Printf("%T\n", s)   // 类型
```

日志里打印未知字符串，优先 `%q`，避免换行把日志冲乱。

---

## 想一想

1. 为什么 `len("小")` 是 3？`for i := 0; i < len(s); i++` 按「字」处理中文会怎样？
2. 循环里用 `s = s + chunk` 和 `Builder` 的复杂度差在哪？
3. 什么时候该继续用 `string`，什么时候该改用 `[]byte`？

## 练习

1. 写 `func ReverseRunes(s string) string`，按码点反转（注意不要按字节反转 UTF-8）。
2. 写 `func WordCount(s string) map[string]int`，用 `strings.Fields`。
3. 比较 `s + s + s` 循环 1 万次和 `Builder` 的基准（学完测试章再跑 `go test -bench`）。

入门篇结束。下一篇从 [指针](../02-核心/01-指针.md) 开始。
