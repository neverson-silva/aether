package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"aether/internal/domain"
	"aether/internal/runtime/compose"
)

// GET /api/v1/apps/{id}/compose — docker-compose.yml atual (ao vivo), gerado
// a partir do Deployment Spec. ?download=1 baixa o arquivo.
func (s *Server) handleAppCompose(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	yaml, err := s.core.GenerateCompose(app)
	if err != nil {
		writeErr(w, 500, "erro ao gerar compose: "+err.Error())
		return
	}
	if r.URL.Query().Get("download") == "1" {
		w.Header().Set("Content-Type", "application/x-yaml")
		w.Header().Set("Content-Disposition", "attachment; filename=docker-compose.yml")
		w.Write([]byte(yaml))
		return
	}
	writeJSON(w, 200, map[string]string{"compose": yaml})
}

// GET /api/v1/apps/{id}/export?runtime=compose|kubernetes|nomad
// Exporta o serviço a partir do Deployment Spec para o runtime escolhido.
func (s *Server) handleExportRuntime(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	spec, err := s.core.AppToSpec(app, 0)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	runtime := r.URL.Query().Get("runtime")
	switch runtime {
	case "", "compose":
		yml, _ := compose.Generate(spec)
		serveExport(w, "docker-compose.yml", "application/x-yaml", yml)
	case "kubernetes", "k8s":
		manifest, _ := compose.ExportKubernetes(spec)
		serveExport(w, app.Name+".deployment.yaml", "application/x-yaml", manifest)
	case "nomad":
		job, _ := compose.ExportNomad(spec)
		serveExport(w, app.Name+".nomad.hcl", "text/plain", job)
	default:
		writeErr(w, 400, "runtime inválido (use compose, kubernetes ou nomad)")
	}
}

func serveExport(w http.ResponseWriter, filename, ctype, content string) {
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Write([]byte(content))
}

// GET /api/v1/apps/{id}/spec — Deployment Spec tipado (fonte de verdade).
func (s *Server) handleAppSpec(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	spec, err := s.core.AppToSpec(app, 0)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, spec)
}

// GET /api/v1/apps/{id}/deployments/{depID}/compose — compose histórico usado
// naquele deployment (para diff/rollback/auditoria).
func (s *Server) handleDeploymentCompose(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	dep, err := s.core.Store.GetDeployment(r.PathValue("depID"))
	if err != nil || dep.AppID != app.ID {
		writeErr(w, 404, "deployment não encontrado")
		return
	}
	if dep.ComposeYAML == "" {
		writeErr(w, 404, "compose não capturado neste deployment")
		return
	}
	if r.URL.Query().Get("download") == "1" {
		w.Header().Set("Content-Type", "application/x-yaml")
		w.Header().Set("Content-Disposition", "attachment; filename=deployment-"+strconv.FormatInt(dep.Number, 10)+"-docker-compose.yml")
		w.Write([]byte(dep.ComposeYAML))
		return
	}
	writeJSON(w, 200, map[string]any{
		"number":  dep.Number,
		"hash":    dep.ComposeHash,
		"compose": dep.ComposeYAML,
	})
}

// POST /api/v1/apps/{id}/compose/import — importa um docker-compose.yml e
// atualiza o Deployment Spec do app (migração de fonte de verdade).
func (s *Server) handleImportCompose(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	body, err := readBody(r, 1<<20)
	if err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	var req struct {
		Compose string `json:"compose"`
	}
	if json.Unmarshal(body, &req) != nil || strings.TrimSpace(req.Compose) == "" {
		writeErr(w, 400, "campo compose obrigatório")
		return
	}
	spec, err := compose.Parse([]byte(req.Compose))
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	// aplica portas/volumes/envs detectados ao app
	if len(spec.Ports) > 0 {
		if p := spec.Ports[0]; p.Container != "" {
			if n, err := strconv.Atoi(p.Container); err == nil {
				app.Port = n
			}
			if h, err := strconv.Atoi(p.Host); err == nil {
				app.Port = h
			}
		}
	}
	var vols []domain.Volume
	for _, v := range spec.Volumes {
		if v.Target != "" {
			vols = append(vols, domain.Volume{Name: safeVolName(v.Source), MountPath: v.Target})
		}
	}
	app.Volumes = vols
	if spec.Image != "" {
		app.Image = spec.Image
		app.SourceType = domain.SourceImage
	}
	if err := s.core.Store.UpdateApp(app); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"spec": spec, "port": app.Port})
}

func safeVolName(s string) string {
	if s == "" {
		return "data"
	}
	out := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, s)
	if out == "" {
		return "data"
	}
	return out
}

func readBody(r *http.Request, limit int64) ([]byte, error) {
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 4096)
	for {
		n, err := r.Body.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if int64(len(buf)) > limit {
			return nil, errTooLarge
		}
		if err != nil {
			break
		}
	}
	return buf, nil
}

var errTooLarge = errBodyTooLarge

type errBodyTooLargeType struct{}

func (errBodyTooLargeType) Error() string { return "corpo muito grande" }

var errBodyTooLarge = errBodyTooLargeType{}
