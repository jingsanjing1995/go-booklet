package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type Namer interface {
	Name() string
}

type User struct{ name string }

func (u User) Name() string { return u.name }

type Logger struct{ w io.Writer }

func (l *Logger) Write(p []byte) (int, error) {
	return fmt.Fprintf(l.w, "[log] %s", p)
}

var _ io.Writer = (*Logger)(nil)

type pathError struct{ Path string }

func (e *pathError) Error() string { return "path: " + e.Path }

func greet(n Namer) string { return "hello, " + n.Name() }

func typedNil() {
	var p *pathError
	var err error = p
	fmt.Printf("typed-nil: err==nil? %v  类型=%T  值=%#v\n", err == nil, err, err)
}

func main() {
	fmt.Println(greet(User{name: "Ada"}))

	l := &Logger{w: os.Stdout}
	_, _ = l.Write([]byte("接口编译期检查通过\n"))

	r := strings.NewReader("io.Reader 随处可用\n")
	_, _ = io.Copy(os.Stdout, r)

	typedNil()
}
