package main

import (
	"e/random"
	"log"
	"net/http"
"math"
	"github.com/gin-gonic/gin"
)

func main() {

	router := gin.New()
	health := new(H)

	path := "/health"
	// Health check API required for the Kubernetes pod health
	// check and take action.
	router.GET(path, health.Handler)
	router.Run(":8000")
}

type H struct{}

func (h H) Handler(c *gin.Context) {
	ip := c.ClientIP()
	log.Println("ip----------",ip)
	random.Mapmap()
    compoundprincipal := math.Pow((30000/20000), (1/3)) - 1
	log.Println("compound",compoundprincipal)


	c.JSON(http.StatusOK, gin.H{"Working!": "gghgfgfdgf"})
}
