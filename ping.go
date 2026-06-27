package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func pingHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"code":    200,
		"message": "pong",
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func main() {
	http.HandleFunc("/ping", pingHandler)
	
	fmt.Println("HTTP ping 服务器启动在 :8080 端口")
	fmt.Println("访问 http://localhost:8080/ping 进行测试")
	
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("服务器启动失败: %v\n", err)
	}
}