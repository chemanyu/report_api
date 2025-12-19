//go:build windows
// +build windows

package control

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 开启服务 - Windows版本，使用标准http.Server
func startServerImpl(r *gin.Engine, port string) error {
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	if err := srv.ListenAndServe(); err != nil {
		if err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
		log.Println("Server stopped gracefully")
	}
	return nil
}
