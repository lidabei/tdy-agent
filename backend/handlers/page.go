package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

type PageResult struct {
	List  interface{} `json:"list"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
}

func parsePage(c *gin.Context) (page, size int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ = strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return
}

func yuanToFen(y float64) int64 {
	if y >= 0 {
		return int64(y*100 + 0.5)
	}
	return int64(y*100 - 0.5)
}
