package runtime

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type K8sConfig struct {
	API       string
	Token     string
	CACert    string
	Namespace string
}

func LoadK8sConfig() *K8sConfig {
	return &K8sConfig{
		API:       os.Getenv("AETHER_K8S_API"),
		Token:     os.Getenv("AETHER_K8S_TOKEN"),
		CACert:    os.Getenv("AETHER_K8S_CACERT"),
		Namespace: os.Getenv("AETHER_K8S_NAMESPACE"),
	}
}

func (k *K8sConfig) Valid() bool {
	return k.API != "" && k.Token != ""
}

type k8sDriver struct {
	cfg K8sConfig
	hc  *http.Client
}

func NewK8sDriver(cfg K8sConfig) *k8sDriver {
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if cfg.CACert != "" {
		if data, err := os.ReadFile(cfg.CACert); err == nil {
			pool := x509.NewCertPool()
			pool.AppendCertsFromPEM(data)
			tlsCfg.RootCAs = pool
		}
	}
	return &k8sDriver{cfg: cfg, hc: &http.Client{
		Timeout:   30 * time.Second,
		Transport: &http.Transport{TLSClientConfig: tlsCfg},
	}}
}

func (d *k8sDriver) do(ctx context.Context, method, path string, body any) ([]byte, error) {
	var rd io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rd = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, d.cfg.API+path, rd)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+d.cfg.Token)
	req.Header.Set("Content-Type", "application/json")
	if d.cfg.Namespace == "" {
		d.cfg.Namespace = "default"
	}
	resp, err := d.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("k8s %s %s: %d %s", method, path, resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return data, nil
}

func (d *k8sDriver) ns() string {
	if d.cfg.Namespace == "" {
		return "default"
	}
	return d.cfg.Namespace
}

func (d *k8sDriver) Name() string { return "kubernetes" }

func (d *k8sDriver) Info(ctx context.Context) (Info, error) {
	data, err := d.do(ctx, "GET", "/api/v1/namespaces", nil)
	if err != nil {
		return Info{}, err
	}
	var v struct {
		APIVersion string `json:"apiVersion"`
	}
	json.Unmarshal(data, &v)
	return Info{Driver: "kubernetes", Version: v.APIVersion}, nil
}

func (d *k8sDriver) Pull(ctx context.Context, image string) error { return nil }

func (d *k8sDriver) Exists(ctx context.Context, image string) (bool, error) { return true, nil }

func (d *k8sDriver) Run(ctx context.Context, spec RunSpec) (string, error) {
	port := 80
	for _, p := range spec.Ports {
		if p.ContainerPort != "" {
			port, _ = strconv.Atoi(p.ContainerPort)
			break
		}
	}
	if port == 0 {
		port = 80
	}
	name := sanitizeK8s(spec.Name)
	labels := map[string]string{"aether.app": spec.Labels["aether.app"], "app": name}
	replicas := int32(1)
	dep := map[string]any{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata":   map[string]any{"name": name, "namespace": d.ns(), "labels": labels},
		"spec": map[string]any{
			"replicas": replicas,
			"selector": map[string]any{"matchLabels": map[string]any{"app": name}},
			"template": map[string]any{
				"metadata": map[string]any{"labels": labels},
				"spec": map[string]any{
					"containers": []map[string]any{{
						"name":  name,
						"image": spec.Image,
						"env":   k8sEnv(spec.Env),
						"ports": []map[string]any{{"containerPort": port}},
					}},
				},
			},
		},
	}
	if spec.MemMB > 0 || spec.CPUs != "" {
		res := map[string]any{}
		limits := map[string]any{}
		reqs := map[string]any{}
		if spec.MemMB > 0 {
			limits["memory"] = strconv.FormatInt(spec.MemMB, 10) + "Mi"
			reqs["memory"] = strconv.FormatInt(spec.MemMB/2, 10) + "Mi"
		}
		if spec.CPUs != "" {
			limits["cpu"] = spec.CPUs
		}
		res["limits"] = limits
		res["requests"] = reqs
		dep["spec"].(map[string]any)["template"].(map[string]any)["spec"].(map[string]any)["containers"].([]map[string]any)[0]["resources"] = res
	}
	if _, err := d.do(ctx, "POST", "/apis/apps/v1/namespaces/"+d.ns()+"/deployments", dep); err != nil {
		return "", err
	}
	svc := map[string]any{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata":   map[string]any{"name": name, "namespace": d.ns(), "labels": labels},
		"spec": map[string]any{
			"selector": map[string]any{"app": name},
			"ports":    []map[string]any{{"port": port, "targetPort": port}},
		},
	}
	if _, err := d.do(ctx, "POST", "/api/v1/namespaces/"+d.ns()+"/services", svc); err != nil {
		return "", err
	}
	return name, nil
}

func (d *k8sDriver) Start(ctx context.Context, id string) error { return d.scale(ctx, id, 1) }

func (d *k8sDriver) Stop(ctx context.Context, id string) error { return d.scale(ctx, id, 0) }

func (d *k8sDriver) scale(ctx context.Context, id string, replicas int) error {
	body := map[string]any{"spec": map[string]any{"replicas": replicas}}
	_, err := d.do(ctx, "PATCH", "/apis/apps/v1/namespaces/"+d.ns()+"/deployments/"+id+"/scale", body)
	return err
}

func (d *k8sDriver) Restart(ctx context.Context, id string) error {
	_, err := d.do(ctx, "POST", "/apis/apps/v1/namespaces/"+d.ns()+"/deployments/"+id+"/rollout/restart", map[string]any{})
	return err
}

func (d *k8sDriver) Remove(ctx context.Context, id string, force bool) error {
	if _, err := d.do(ctx, "DELETE", "/apis/apps/v1/namespaces/"+d.ns()+"/deployments/"+id, nil); err != nil {
		return err
	}
	_, err := d.do(ctx, "DELETE", "/api/v1/namespaces/"+d.ns()+"/services/"+id, nil)
	return err
}

func (d *k8sDriver) Inspect(ctx context.Context, id string) (ContainerInfo, error) {
	data, err := d.do(ctx, "GET", "/apis/apps/v1/namespaces/"+d.ns()+"/deployments/"+id, nil)
	if err != nil {
		return ContainerInfo{}, err
	}
	var dep struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
		Status struct {
			Replicas          int `json:"replicas"`
			AvailableReplicas int `json:"availableReplicas"`
			Conditions        []struct {
				Type   string `json:"type"`
				Status string `json:"status"`
			} `json:"conditions"`
		} `json:"status"`
	}
	json.Unmarshal(data, &dep)
	state := "running"
	for _, c := range dep.Status.Conditions {
		if c.Type == "Available" && c.Status == "True" {
			state = "running"
		}
	}
	if dep.Status.AvailableReplicas == 0 {
		state = "creating"
	}
	return ContainerInfo{ID: dep.Metadata.Name, Name: dep.Metadata.Name, State: state, StartedAt: time.Now()}, nil
}

func (d *k8sDriver) Ports(ctx context.Context, id string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (d *k8sDriver) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	params := "?container=" + id
	if follow {
		params += "&follow=true"
	}
	data, err := d.do(ctx, "GET", "/api/v1/namespaces/"+d.ns()+"/pods/"+id+"/log"+params, nil)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (d *k8sDriver) Stats(ctx context.Context, id string) (Stats, error) {
	return Stats{}, nil
}

func (d *k8sDriver) Build(ctx context.Context, dir, dockerfile, tag string) (string, error) {
	return "", fmt.Errorf("build no driver kubernetes não suportado (use o driver local)")
}

func (d *k8sDriver) BuildWithWriter(ctx context.Context, dir, dockerfile, tag string, w io.Writer) (string, error) {
	return "", fmt.Errorf("build no driver kubernetes não suportado (use o driver local)")
}

func (d *k8sDriver) UpdateResources(ctx context.Context, id string, memMB int64, cpus string) error {
	limits := map[string]any{}
	if memMB > 0 {
		limits["memory"] = strconv.FormatInt(memMB, 10) + "Mi"
	}
	if cpus != "" {
		limits["cpu"] = cpus
	}
	if len(limits) == 0 {
		return nil
	}
	body := map[string]any{"spec": map[string]any{"template": map[string]any{"spec": map[string]any{
		"containers": []map[string]any{{"name": id, "resources": map[string]any{"limits": limits}}},
	}}}}
	_, err := d.do(ctx, "PATCH", "/apis/apps/v1/namespaces/"+d.ns()+"/deployments/"+id, body)
	return err
}

func (d *k8sDriver) NetworkCreate(ctx context.Context, name string) error { return nil }

func (d *k8sDriver) NetworkRemove(ctx context.Context, name string) error { return nil }

func (d *k8sDriver) VolumeCreate(ctx context.Context, name string, sizeMB int64) error { return nil }

func (d *k8sDriver) VolumeRemove(ctx context.Context, name string) error { return nil }

func (d *k8sDriver) Exec(ctx context.Context, id string, req ExecRequest) (*ExecResult, error) {
	return nil, fmt.Errorf("exec no driver kubernetes não suportado")
}

func (d *k8sDriver) ExecStream(ctx context.Context, id string, req ExecRequest) (ExecStream, error) {
	return nil, fmt.Errorf("exec-stream no driver kubernetes não suportado")
}

func sanitizeK8s(name string) string {
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	out := strings.ToLower(b.String())
	out = strings.Trim(out, "-")
	if out == "" {
		out = "app"
	}
	if len(out) > 53 {
		out = out[:53]
	}
	return out
}

func k8sEnv(env []string) []map[string]any {
	var out []map[string]any
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			continue
		}
		out = append(out, map[string]any{"name": parts[0], "value": parts[1]})
	}
	return out
}
