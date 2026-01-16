package api

import (
	"net/http"

	"github.com/celerix-dev/celerix-store/pkg/sdk"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	Store sdk.CelerixStore
}

func (h *Handler) GetPersonas(c *gin.Context) {
	personas, err := h.Store.GetPersonas()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, personas)
}

func (h *Handler) GetApps(c *gin.Context) {
	personaID := c.Param("persona")
	apps, err := h.Store.GetApps(personaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, apps)
}

func (h *Handler) GetAppStore(c *gin.Context) {
	personaID := c.Param("persona")
	appID := c.Param("app")
	data, err := h.Store.GetAppStore(personaID, appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *Handler) GetGlobal(c *gin.Context) {
	appID := c.Param("app")
	key := c.Param("key")
	val, persona, err := h.Store.GetGlobal(appID, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"persona": persona,
		"value":   val,
	})
}

func (h *Handler) Set(c *gin.Context) {
	personaID := c.Param("persona")
	appID := c.Param("app")
	key := c.Param("key")

	var val any
	if err := c.ShouldBindJSON(&val); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.Store.Set(personaID, appID, key, val); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *Handler) Delete(c *gin.Context) {
	personaID := c.Param("persona")
	appID := c.Param("app")
	key := c.Param("key")

	if err := h.Store.Delete(personaID, appID, key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *Handler) Move(c *gin.Context) {
	var input struct {
		SrcPersona string `json:"src_persona" binding:"required"`
		DstPersona string `json:"dst_persona" binding:"required"`
		AppID      string `json:"app_id" binding:"required"`
		Key        string `json:"key" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.Store.Move(input.SrcPersona, input.DstPersona, input.AppID, input.Key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
