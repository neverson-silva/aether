package http

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authhttp "aether/internal/auth/http"
	authdomain "aether/internal/auth/domain"
	"aether/internal/database/adapter"
	"aether/internal/databases/domain"
)

func (h *Handler) studioID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("dbID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) StudioMeta(c *gin.Context) {
	id, ok := h.studioID(c)
	if !ok {
		return
	}
	meta, err := h.studio.IntrospectMeta(c.Request.Context(), id, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, meta)
}

func (h *Handler) StudioSchemas(c *gin.Context) {
	id, ok := h.studioID(c)
	if !ok {
		return
	}
	schemas, err := h.studio.IntrospectSchemas(c.Request.Context(), id, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, schemas)
}

func (h *Handler) StudioObjects(c *gin.Context) {
	id, ok := h.studioID(c)
	if !ok {
		return
	}
	objs, err := h.studio.IntrospectObjects(c.Request.Context(), id, orgID(c), c.Param("schema"))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, objs)
}

func (h *Handler) StudioObjectsList(c *gin.Context) {
	id, ok := h.studioID(c)
	if !ok {
		return
	}
	objs, err := h.studio.ListObjects(c.Request.Context(), id, orgID(c), c.Param("schema"))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, objs)
}

func (h *Handler) StudioTable(c *gin.Context) {
	id, ok := h.studioID(c)
	if !ok {
		return
	}
	detail, err := h.studio.IntrospectTable(c.Request.Context(), id, orgID(c), c.Param("schema"), c.Param("table"))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, detail)
}

func (h *Handler) StudioTableData(c *gin.Context) {
	id, ok := h.studioID(c)
	if !ok {
		return
	}
	opts := adapter.QueryOptions{Limit: queryInt(c, "limit", 100), Offset: queryInt(c, "offset", 0), Sort: c.Query("sort"), Order: c.Query("order")}
	if fs := c.Query("filters"); fs != "" {
		for _, f := range strings.Split(fs, ",") {
			parts := strings.SplitN(f, ":", 3)
			if len(parts) < 2 {
				continue
			}
			op := "="
			val := ""
			if len(parts) == 3 {
				op, val = parts[1], parts[2]
			} else {
				val = parts[1]
			}
			opts.Filters = append(opts.Filters, adapter.Filter{Column: parts[0], Op: op, Value: val})
		}
	}
	res, err := h.studio.TableData(c.Request.Context(), id, orgID(c), c.Param("schema"), c.Param("table"), opts)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

type studioQueryReq struct {
	SQL     string `json:"sql"`
	Timeout int    `json:"timeout"`
}

func (h *Handler) StudioQuery(c *gin.Context) {
	id, ok := h.studioID(c)
	if !ok {
		return
	}
	var req studioQueryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if strings.TrimSpace(req.SQL) == "" {
		abort(c, domain.ErrValidation)
		return
	}
	res, err := h.studio.Query(c.Request.Context(), id, orgID(c), req.SQL, adapter.QueryOptions{})
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) StudioExec(c *gin.Context) {
	id, ok := h.studioID(c)
	if !ok {
		return
	}
	role, _ := c.MustGet(authhttp.ContextRole).(string)
	if !authdomain.Role(role).CanManage() {
		abort(c, domain.ErrForbidden)
		return
	}
	var req studioQueryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	res, err := h.studio.Exec(c.Request.Context(), id, orgID(c), req.SQL)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) StudioRefresh(c *gin.Context) {
	id, ok := h.studioID(c)
	if !ok {
		return
	}
	role, _ := c.MustGet(authhttp.ContextRole).(string)
	if !authdomain.Role(role).CanManage() {
		abort(c, domain.ErrForbidden)
		return
	}
	if err := h.studio.Refresh(c.Request.Context(), id, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func queryInt(c *gin.Context, key string, def int) int {
	if v := c.Query(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}