package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

func hello(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("名字不能为空")
	}
	return "你好，" + name, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "用法: go run ./01-hello <名字>")
		os.Exit(2)
	}
	msg, err := hello(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
	fmt.Println(msg)
}
