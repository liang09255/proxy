package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

func main() {
	g := gin.New()
	g.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]string{
			"Msg":    "OK",
			"Time":   time.Now().Format(time.StampMilli),
			"Method": "GET",
		})
	})
	g.POST("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]string{
			"Msg":    "OK",
			"Time":   time.Now().Format(time.StampMilli),
			"Method": "POST",
		})
	})
	if err := g.Run("127.0.0.1:8080"); err != nil {
		panic(err)
	}
}
