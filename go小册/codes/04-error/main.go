package main

import (
	"errors"
	"fmt"
	"os"
)

var ErrNotFound = errors.New("not found")

type PathError struct {
	Path string
	Err  error
}

func (e *PathError) Error() string {
	return fmt.Sprintf("open %s: %v", e.Path, e.Err)
}

func (e *PathError) Unwrap() error { return e.Err }

func find(id string) (string, error) {
	if id == "" {
		return "", errors.New("empty id")
	}
	if id != "42" {
		return "", fmt.Errorf("user %s: %w", id, ErrNotFound)
	}
	return "Ada", nil
}

func slurp(path string) (data []byte, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, &PathError{Path: path, Err: err}
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			err = errors.Join(err, cerr)
		}
	}()
	return []byte("ok"), nil
}

func main() {
	_, err := find("7")
	if errors.Is(err, ErrNotFound) {
		fmt.Println("Is NotFound:", err)
	}

	_, err = slurp("definitely-missing-file-xiaoce")
	var pe *PathError
	if errors.As(err, &pe) {
		fmt.Println("As PathError:", pe.Path)
	}
}
