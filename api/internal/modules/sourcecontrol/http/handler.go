package http

import (
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authhttp "aether/internal/modules/auth/http"
	"aether/internal/modules/sourcecontrol/application"
	"aether/internal/modules/sourcecontrol/domain"
	"aether/internal/platform/scm/github"
)

type Handler struct {
	Service     *application.Service
	Connections *application.Connections
	GitHub      *github.Provider
	AppSlug     string
}

func New(service *application.Service, connections *application.Connections, provider *github.Provider, appSlug string) *Handler {
	return &Handler{Service: service, Connections: connections, GitHub: provider, AppSlug: appSlug}
}

func (h *Handler) GitHubInstallURL(c *gin.Context) {
	if h.AppSlug == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "github app is not configured"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": "https://github.com/apps/" + h.AppSlug + "/installations/new"})
}

func (h *Handler) ListConnections(c *gin.Context) {
	if h.Connections == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "github source control is not configured"})
		return
	}
	connections, err := h.Connections.List(c.Request.Context(), orgID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	result := make([]gin.H, 0, len(connections))
	for _, connection := range connections {
		result = append(result, gin.H{
			"id": connection.ID, "provider": connection.Provider, "external_account_id": connection.ExternalAccountID,
			"external_account_name": connection.ExternalAccountName, "installation_id": connection.InstallationID,
			"status": connection.Status, "created_at": connection.CreatedAt, "updated_at": connection.UpdatedAt,
		})
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) ConnectGitHub(c *gin.Context) {
	if h.Connections == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "github source control is not configured"})
		return
	}
	var request struct {
		InstallationID string `json:"installation_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "installation_id is required"})
		return
	}
	connection, err := h.Connections.ConnectGitHub(c.Request.Context(), orgID(c), request.InstallationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": connection.ID, "provider": connection.Provider, "external_account_id": connection.ExternalAccountID, "external_account_name": connection.ExternalAccountName, "installation_id": connection.InstallationID, "status": connection.Status})
}

func (h *Handler) DisconnectGitHub(c *gin.Context) {
	if h.Connections == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "github source control is not configured"})
		return
	}
	connectionID, err := uuid.Parse(c.Param("connectionID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid connection id"})
		return
	}
	if err := h.Connections.Disconnect(c.Request.Context(), orgID(c), connectionID); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "could not disconnect GitHub"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListRepositories(c *gin.Context) {
	if h.Connections == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "github source control is not configured"})
		return
	}
	installationID := c.Query("installation_id")
	if installationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "installation_id is required"})
		return
	}
	repositories, err := h.Connections.ListRepositories(c.Request.Context(), installationID)
	if err != nil {
		if writeGitHubUnavailable(c, err) {
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	result := make([]gin.H, 0, len(repositories))
	for _, repository := range repositories {
		result = append(result, gin.H{"id": repository.ID, "owner": repository.Owner, "name": repository.Name, "full_name": repository.FullName, "default_branch": repository.DefaultBranch})
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) ListBranches(c *gin.Context) {
	if h.Connections == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "github source control is not configured"})
		return
	}
	branches, err := h.Connections.ListBranches(c.Request.Context(), c.Query("installation_id"), c.Param("repositoryID"))
	if err != nil {
		if writeGitHubUnavailable(c, err) {
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	result := make([]gin.H, 0, len(branches))
	for _, branch := range branches {
		result = append(result, gin.H{"name": branch.Name})
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetRepositoryFile(c *gin.Context) {
	if h.Connections == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "github source control is not configured"})
		return
	}
	path := strings.TrimSpace(c.Query("path"))
	ref := strings.TrimSpace(c.Query("ref"))
	if c.Query("installation_id") == "" || path == "" || ref == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "installation_id, path and ref are required"})
		return
	}
	content, err := h.Connections.GetFile(c.Request.Context(), c.Query("installation_id"), c.Param("repositoryID"), path, ref)
	if err != nil {
		if writeGitHubUnavailable(c, err) {
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"path": path, "ref": ref, "content": content})
}

func (h *Handler) StartGitHubManifest(c *gin.Context) {
	if h.Connections == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "github source control is not configured"})
		return
	}
	publicURL := h.Connections.PublicURL
	if publicURL == "" {
		publicURL = "http://" + c.Request.Host
	}
	var request struct {
		ReturnURL string `json:"return_url"`
	}
	if err := c.ShouldBindJSON(&request); err != nil && err.Error() != "EOF" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid return_url"})
		return
	}
	returnURL := request.ReturnURL
	if returnURL == "" || returnURL[0] != '/' || strings.Contains(returnURL, "#") || len(returnURL) > 2048 {
		returnURL = "/?github=connected"
	}
	manifest, state, err := h.Connections.StartManifest(c.Request.Context(), orgID(c), userID(c), publicURL, returnURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": "https://github.com/settings/apps/new", "manifest": manifest, "state": state})
}

func (h *Handler) CompleteGitHubManifest(c *gin.Context) {
	if h.Connections == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "github source control is not configured"})
		return
	}
	_, installURL, err := h.Connections.CompleteManifest(c.Request.Context(), c.Query("state"), c.Query("code"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Redirect(http.StatusFound, installURL)
}

func (h *Handler) CompleteGitHubInstallation(c *gin.Context) {
	if h.Connections == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "github source control is not configured"})
		return
	}
	connectionID, err := uuid.Parse(c.Query("state"))
	if err != nil || c.Query("installation_id") == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid github installation callback"})
		return
	}
	if _, err := h.Connections.CompleteInstallation(c.Request.Context(), connectionID, c.Query("installation_id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	publicURL := h.Connections.PublicURL
	if publicURL == "" {
		publicURL = "http://" + c.Request.Host
	}
	returnURL := c.Query("return_url")
	if returnURL == "" || returnURL[0] != '/' || len(returnURL) > 2048 {
		returnURL = "/?github=connected"
	} else if strings.Contains(returnURL, "#") {
		returnURL = "/?github=connected"
	}
	c.Redirect(http.StatusFound, publicURL+returnURL)
}

func (h *Handler) GetServiceSource(c *gin.Context) {
	serviceID, err := parseServiceID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	source, err := h.Service.GetSource(c.Request.Context(), serviceID, orgID(c))
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "service source not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, source)
}

func (h *Handler) SaveServiceSource(c *gin.Context) {
	serviceID, err := parseServiceID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	var request struct {
		ConnectionID            uuid.UUID `json:"connection_id" binding:"required"`
		RepositoryID            string    `json:"repository_id" binding:"required"`
		RepositoryOwner         string    `json:"repository_owner"`
		RepositoryName          string    `json:"repository_name"`
		RepositoryFullName      string    `json:"repository_full_name"`
		DefaultBranch           string    `json:"default_branch"`
		Branch                  string    `json:"branch"`
		AutoDeploy              bool      `json:"auto_deploy"`
		RootDirectory           string    `json:"root_directory"`
		EnvironmentTemplatePath string    `json:"environment_template_path"`
		WatchPaths              []string  `json:"watch_paths"`
		IgnorePaths             []string  `json:"ignore_paths"`
		WatchRootFiles          bool      `json:"watch_root_files"`
		ComposeFile             string    `json:"compose_file"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	source, err := h.Service.SaveSource(c.Request.Context(), &domain.ServiceSource{
		ServiceID: serviceID, ConnectionID: request.ConnectionID, OrganizationID: orgID(c),
		RepositoryID: request.RepositoryID, RepositoryOwner: request.RepositoryOwner, RepositoryName: request.RepositoryName,
		RepositoryFullName: request.RepositoryFullName, DefaultBranch: request.DefaultBranch, Branch: request.Branch,
		AutoDeploy: request.AutoDeploy, RootDirectory: request.RootDirectory, EnvironmentTemplatePath: request.EnvironmentTemplatePath, WatchPaths: request.WatchPaths,
		IgnorePaths: request.IgnorePaths, WatchRootFiles: request.WatchRootFiles,
		ComposeFile: request.ComposeFile,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "service or source connection not found"})
			return
		}
		if strings.Contains(err.Error(), "repository path escapes checkout") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, source)
}

func (h *Handler) ImportServiceTemplate(c *gin.Context) {
	serviceID, err := parseServiceID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	source, err := h.Service.GetSource(c.Request.Context(), serviceID, orgID(c))
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "service source not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	imported, found, err := h.Service.ImportTemplate(c.Request.Context(), source)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"imported": imported, "found": found})
}

func (h *Handler) DeleteServiceSource(c *gin.Context) {
	serviceID, err := parseServiceID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	if err := h.Service.DeleteSource(c.Request.Context(), serviceID, orgID(c)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func parseServiceID(c *gin.Context) (uuid.UUID, error) {
	value := c.Param("serviceID")
	if value == "" {
		value = c.Param("appID")
	}
	return uuid.Parse(value)
}

func (h *Handler) GitHubPush(c *gin.Context) {
	slog.Default().Info("github webhook received", "event", c.GetHeader("X-GitHub-Event"), "delivery_id", c.GetHeader("X-GitHub-Delivery"))
	if h.Service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "github source control is not configured"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 5<<20))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook body"})
		return
	}
	var event domain.PushEvent
	event, err = github.ParsePushWebhook(c.Request.Header, body)
	if err != nil {
		if c.GetHeader("X-GitHub-Event") != "push" {
			c.Status(http.StatusNoContent)
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	provider := h.GitHub
	if h.Connections != nil {
		provider, err = h.Connections.ProviderForInstallation(c.Request.Context(), event.InstallationID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
	}
	if err := provider.VerifyWebhook(c.Request.Header, body); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if err := h.Service.HandlePushWithFiles(c.Request.Context(), event, provider); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "accepted"})
}

func orgID(c *gin.Context) uuid.UUID {
	return c.MustGet(authhttp.ContextOrgID).(uuid.UUID)
}

func userID(c *gin.Context) uuid.UUID {
	return c.MustGet(authhttp.ContextUserID).(uuid.UUID)
}

func writeGitHubUnavailable(c *gin.Context, err error) bool {
	var apiErr *github.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "github connection is no longer available"})
		return true
	}
	return false
}
