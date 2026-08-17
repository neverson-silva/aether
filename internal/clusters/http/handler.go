package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authhttp "aether/internal/auth/http"
	"aether/internal/clusters/application"
	"aether/internal/clusters/domain"
)

type Handler struct {
	clusters *application.Clusters
}

func New(clusters *application.Clusters) *Handler {
	return &Handler{clusters: clusters}
}

type clusterReq struct {
	Name   string   `json:"name"`
	Labels []string `json:"labels"`
}

type addServerReq struct {
	ServerID string `json:"server_id"`
}

type registryReq struct {
	Enabled bool `json:"enabled"`
}

func (h *Handler) CreateCluster(c *gin.Context) {
	var req clusterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	cluster, err := h.clusters.CreateCluster(c.Request.Context(), orgID(c), req.Name, req.Labels)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, clusterDTO(cluster))
}

func (h *Handler) ListClusters(c *gin.Context) {
	clusters, err := h.clusters.ListClusters(c.Request.Context(), orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(clusters))
	for i := range clusters {
		out = append(out, clusterDTO(&clusters[i]))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) DeleteCluster(c *gin.Context) {
	id, err := uuid.Parse(c.Param("clusterID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.clusters.DeleteCluster(c.Request.Context(), id, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) AddServer(c *gin.Context) {
	clusterID, err := uuid.Parse(c.Param("clusterID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req addServerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	serverID, err := uuid.Parse(req.ServerID)
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.clusters.AddServer(c.Request.Context(), clusterID, orgID(c), serverID); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "added"})
}

func (h *Handler) RemoveServer(c *gin.Context) {
	clusterID, err := uuid.Parse(c.Param("clusterID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	serverID, err := uuid.Parse(c.Param("serverID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.clusters.RemoveServer(c.Request.Context(), clusterID, orgID(c), serverID); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "removed"})
}

func (h *Handler) ListServers(c *gin.Context) {
	servers, err := h.clusters.ListServers(c.Request.Context())
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(servers))
	for i := range servers {
		out = append(out, serverDTO(&servers[i]))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) ServerToken(c *gin.Context) {
	token, err := h.clusters.AgentToken(c.Request.Context())
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}

func (h *Handler) DeleteServer(c *gin.Context) {
	id, err := uuid.Parse(c.Param("serverID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.clusters.DeleteServer(c.Request.Context(), id); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) GetRegistry(c *gin.Context) {
	registry, err := h.clusters.GetRegistry(c.Request.Context())
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, registryDTO(registry))
}

func (h *Handler) SetRegistry(c *gin.Context) {
	var req registryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	registry, err := h.clusters.SetRegistryEnabled(c.Request.Context(), req.Enabled)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, registryDTO(registry))
}

func (h *Handler) RegistryImages(c *gin.Context) {
	images, err := h.clusters.RegistryImages(c.Request.Context())
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(images))
	for _, img := range images {
		out = append(out, gin.H{
			"repo": img.Repo, "tag": img.Tag, "id": img.ID, "size": img.Size, "created": img.Created,
		})
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) RegistryImageDelete(c *gin.Context) {
	if err := h.clusters.RegistryDelete(c.Request.Context(), c.Param("repo"), c.Param("tag")); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func clusterDTO(cl *domain.Cluster) gin.H {
	return gin.H{
		"id": cl.ID, "org_id": cl.OrgID, "name": cl.Name, "labels": cl.Labels, "created_at": cl.CreatedAt,
	}
}

func serverDTO(s *domain.Server) gin.H {
	return gin.H{
		"id": s.ID, "name": s.Name, "host": s.Host, "role": s.Role, "status": s.Status,
		"version": s.Version, "labels": s.Labels, "cpu_cores": s.CPUCores,
		"mem_total_bytes": s.MemTotalBytes, "load": s.Load, "last_heartbeat": s.LastHeartbeat,
		"cluster_id": s.ClusterID, "created_at": s.CreatedAt,
	}
}

func registryDTO(r *domain.Registry) gin.H {
	return gin.H{
		"enabled": r.Enabled, "host": r.Host, "port": r.Port, "container_id": r.ContainerID, "status": r.Status,
	}
}

func orgID(c *gin.Context) uuid.UUID {
	return c.MustGet(authhttp.ContextOrgID).(uuid.UUID)
}

func abort(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, domain.ErrConflict):
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "conflict"})
	case errors.Is(err, domain.ErrValidation):
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
	case errors.Is(err, domain.ErrForbidden):
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied"})
	default:
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}
