# 02 · 反射与 unsafe

反射让程序在运行期观察和修改类型与值；`unsafe` 让你绕过类型系统看内存。两者都强大，也都容易写出「能跑但对版本和架构脆弱」的代码。原则：**先想标准库和代码生成能不能解决，再考虑反射，最后才是 unsafe。**

---

## 1. 反射解决什么

`encoding/json`、ORM、校验库、配置库必须在编译期不知道你的结构体长什么样。它们靠：

```go
t := reflect.TypeOf(v)
val := reflect.ValueOf(v)
```

你自己的业务代码很少需要直接碰 `reflect`。

---

## 2. Type 与 Value

```go
var x int = 3
t := reflect.TypeOf(x)   // int
v := reflect.ValueOf(x)  // 3 的拷贝

fmt.Println(t.Kind())    // Int
fmt.Println(v.Int())
```

`Kind` 是底层种类（`Int` `Struct` `Slice` `Ptr`…），`Type` 还包含包路径、字段名。`type Age int` 的 Kind 仍是 `Int`，Type 不是 `int`。

要修改原值，必须传入指针，并且 `Elem()` 后 `CanSet()`：

```go
func setToZero(p any) {
	v := reflect.ValueOf(p)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return
	}
	e := v.Elem()
	if e.CanSet() {
		e.Set(reflect.Zero(e.Type()))
	}
}
```

`reflect.ValueOf(x)` 得到的是 **x 的副本**，改它不影响外面。

---

## 3. 走过结构体

```go
func fields(v any) {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fmt.Println(f.Name, f.Type, f.Tag.Get("json"))
	}
}
```

未导出字段：能看见名字，但不能 `Set`。嵌入字段会出现在列表里，需要自己决定是否展开。

调用方法：

```go
m := reflect.ValueOf(u).MethodByName("Name")
if m.IsValid() {
	out := m.Call(nil)
	fmt.Println(out[0].String())
}
```

参数和返回值都是 `[]reflect.Value`，写起来冗长，错了运行时才炸。

---

## 4. 性能与可维护性

反射的开销在于：动态决策、装箱、不能内联。热路径上对每个字段 `Field(i)` 会很疼；成熟库会缓存 `Type → 字段偏移`。

比性能更麻烦的是：**重构改字段名，编译器不帮你**。必须靠测试和标签约定。

替代：

- `go generate` + 模板生成编解码
- 接口
- 泛型
- 显式手写 `MarshalJSON`

---

## 5. unsafe：你在和内存布局签约

```go
import "unsafe"
```

常见合法用途（仍要谨慎）：

- `unsafe.Sizeof` / `Alignof` / `Offsetof` 看布局
- 和 cgo、系统调用交换指针
- 标准库内部的优化（你去抄前先读发布说明）

```go
type user struct {
	age  int32
	flag bool
}

fmt.Println(unsafe.Sizeof(user{}))
fmt.Println(unsafe.Offsetof(user{}.flag))
```

对齐会插入填充。字段顺序影响结构体大小，这是性能调优里「把大字段放一起、减少 padding」的依据。

`unsafe.Pointer` 可以在类型之间转换，规则写在 [unsafe 包文档](https://pkg.go.dev/unsafe) 的「合法使用」六条里。违反它们，下次编译器升级就可能静默坏掉。

**不要**用 unsafe 做：

- 把 `[]byte` 转 `string` 省一次拷贝还继续改底层字节（会破坏字符串不可变，极难查）
- 依赖 slice header 的私有结构（已有 `unsafe.Slice` 等官方入口时用官方的）
- 绕过未导出字段去改别人的包

Go 1.17+ 提供 `unsafe.Slice`、`unsafe.String`（1.20，且有只读约定）。即使用这些，也要保证生命周期和只读承诺。

---

## 6. 和 GC 的关系

GC 靠类型信息扫指针。把整数当指针、把指针藏进 `uintptr` 再跨调用保存，都可能让 GC 收掉对象，留下悬空。`uintptr` 是数字，不是指针。文档写得很狠，请当真。

---

## 7. 实用建议

1. 写 JSON 标签和 `encoding/json` 就够的，不要自己反射一遍。
2. 必须反射时，先 `TypeOf` 做缓存，再对 `Value` 操作。
3. 用 `go test` 锁住字段名和标签。
4. `unsafe` 出现在业务 PR 里，默认应该被问「为什么不能不用」。
5. 读标准库里的用法当参考，不要只读博客里的「黑科技一行」。

---

## 想一想

1. 为什么 `ValueOf(x).Set(...)` 通常改不了 x？
2. 反射为什么让「改名重构」变危险？
3. `uintptr` 和 `unsafe.Pointer` 哪一个能被 GC 识别为指针？

## 练习

1. 写一个函数，打印任意结构体的 json 标签列表。
2. 用反射把结构体里所有 `string` 字段 `TrimSpace`（只改导出字段）。
3. 用 `Sizeof`/`Offsetof` 打印一个带 padding 的结构体，再调整字段顺序看大小变化。

下一章：[测试与基准](./03-测试与基准.md)
