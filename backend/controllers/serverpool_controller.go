package controllers

import (
	"PoolManagerVM/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetServerpool(c *gin.Context) {

	allServers, err := utils.GetAllServers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	var activeServs []gin.H
	for _, s := range allServers {
		activeServs = append(activeServs, gin.H{
			"id":       s.ID,
			"name":     s.Name,
			"HostID":   s.HostID,
			"status":   s.Status,
			"Progress": s.Progress,
		})
	}
	c.JSON(http.StatusOK, gin.H{"servers": activeServs})
}
