package main

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"time"
)

func main() {
	g := gin.Default()
	g.GET("/", func(c *gin.Context) {
		log.Println(c.Request.RemoteAddr)
		c.JSON(http.StatusOK, map[string]string{
			"Msg":    "OK",
			"Time":   time.Now().Format(time.StampMilli),
			"Method": "GET",
		})
	})
	g.POST("/", func(c *gin.Context) {
		log.Println(c.Request.RemoteAddr)
		c.JSON(http.StatusOK, map[string]string{
			"Msg":    "OK",
			"Time":   time.Now().Format(time.StampMilli),
			"Method": "POST",
		})
	})
	g.Any("/hello", func(c *gin.Context) {
		log.Println(c.Request.RemoteAddr)
		c.JSON(http.StatusOK, map[string]string{
			"Msg":    "Hello world!",
			"Time":   time.Now().Format(time.StampMilli),
			"Method": c.Request.Method,
		})
	})
	if err := g.Run("30.0.0.1:8889"); err != nil {
		panic(err)
	}
}
