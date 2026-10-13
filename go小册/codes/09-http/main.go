package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"
)

var errNotFound = errors.New("not found")

type user struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "ok")
}

func getUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "empty id", http.StatusBadRequest)
		return
	}
	u, err := findUser(id)
	if err != nil {
		if errors.Is(err, errNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(u)
}

func findUser(id string) (user, error) {
	if id == "1" {
		return user{ID: "1", Name: "Ada"}, nil
	}
	return user{}, errNotFound
}

func withLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		fmt.Printf("%s %s %s\n", r.Method, r.URL.Path, time.Since(start))
	})
}

func mux() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /health", health)
	m.HandleFunc("GET /users/{id}", getUser)
	return withLog(m)
}

func main() {
	ts := httptest.NewServer(mux())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		panic(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	fmt.Println("health:", resp.StatusCode, string(body))

	resp, err = http.Get(ts.URL + "/users/1")
	if err != nil {
		panic(err)
	}
	body, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	fmt.Println("user:", resp.StatusCode, string(body))
}
