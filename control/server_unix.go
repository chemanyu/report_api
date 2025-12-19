//go:build !windows
// +build !windows

package control

import (
	"log"
	"net/http"

	"github.com/fvbock/endless"
	"github.com/gin-gonic/gin"
)

// 开启服务 - Unix/Linux版本，支持优雅重启
func startServerImpl(r *gin.Engine, port string) error {
	srv := endless.NewServer(":"+port, r)
	if err := srv.ListenAndServe(); err != nil {
		if err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
		log.Println("Server stopped gracefully")
	}
	return nil
}
