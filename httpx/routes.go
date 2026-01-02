package httpx

import "github.com/gin-gonic/gin"

type Routes func(r *gin.Engine)

func PublicRoutes(r *gin.Engine) {
	r.GET("/healthz", func(c *gin.Context) { c.String(200, "ok") })
	r.GET("/readyz", func(c *gin.Context) { c.String(200, "ready") })
}
