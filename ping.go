package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type PingResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	response := PingResponse{
		Code:    200,
		Message: "pong",
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func main() {
	http.HandleFunc("/ping", pingHandler)
	
	log.Println("HTTP server starting on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}