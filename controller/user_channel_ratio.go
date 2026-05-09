package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type UserChannelRatioRequest struct {
	UserId    int     `json:"user_id"`
	ChannelId int     `json:"channel_id"`
	Ratio     float64 `json:"ratio"`
}

func CreateUserChannelRatio(c *gin.Context) {
	var req UserChannelRatioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	ratio := model.UserChannelRatio{
		UserId:    req.UserId,
		ChannelId: req.ChannelId,
		Ratio:     req.Ratio,
	}
	if err := ratio.Create(); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    ratio,
	})
}

func UpdateUserChannelRatio(c *gin.Context) {
	var req model.UserChannelRatio
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if err := req.Update(); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    req,
	})
}

func GetUserChannelRatios(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Query("user_id"))
	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	var ratios []model.UserChannelRatio
	var total int64
	var err error

	if userId > 0 {
		ratios, err = model.GetUserChannelRatios(userId)
		total = int64(len(ratios))
	} else {
		ratios, total, err = model.GetAllUserChannelRatios(page, pageSize)
	}

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"items":     ratios,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

func DeleteUserChannelRatio(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := model.DeleteUserChannelRatio(id); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}
