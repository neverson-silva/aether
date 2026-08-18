package application

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func (m *Monitoring) storageScan(ctx context.Context) (map[string]uint64, error) {
	dfOut, err := exec.CommandContext(ctx, "podman", "system", "df", "-v").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("system df: %w", err)
	}
	containerSize, volumeSize, volumeLinks := parseSystemDF(string(dfOut))

	mounts, err := m.mountsPerContainer(ctx)
	if err != nil {
		return nil, fmt.Errorf("mounts: %w", err)
	}

	sizeByVol := map[string]uint64{}
	linksByVol := map[string]int{}
	for _, v := range volumeSize {
		if _, ok := sizeByVol[v.name]; !ok {
			sizeByVol[v.name] = v.size
		}
	}
	for _, v := range volumeLinks {
		linksByVol[v.name] = v.links
	}

	out := map[string]uint64{}
	for _, c := range containerSize {
		total := c.size
		for _, vol := range mounts[c.name] {
			sz, ok := sizeByVol[vol]
			if !ok || sz == 0 {
				continue
			}
			links := linksByVol[vol]
			if links <= 0 {
				links = 1
			}
			total += sz / uint64(links)
		}
		out[c.name] = total
	}
	return out, nil
}

func (m *Monitoring) mountsPerContainer(ctx context.Context) (map[string][]string, error) {
	idsOut, err := exec.CommandContext(ctx, "podman", "ps", "-aq").CombinedOutput()
	if err != nil {
		return map[string][]string{}, nil
	}
	ids := strings.Fields(string(idsOut))
	if len(ids) == 0 {
		return map[string][]string{}, nil
	}
	args := append([]string{"inspect", "--format", "{{.Name}}|{{json .Mounts}}"}, ids...)
	raw, err := exec.CommandContext(ctx, "podman", args...).CombinedOutput()
	if err != nil {
		return map[string][]string{}, nil
	}
	result := map[string][]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, "|")
		if idx < 0 {
			continue
		}
		name := strings.TrimSpace(line[:idx])
		var mounts []struct {
			Type string `json:"Type"`
			Name string `json:"Name"`
		}
		if err := json.Unmarshal([]byte(line[idx+1:]), &mounts); err != nil {
			continue
		}
		for _, mnt := range mounts {
			if mnt.Type == "volume" && mnt.Name != "" {
				result[name] = append(result[name], mnt.Name)
			}
		}
	}
	return result, nil
}

type dfRow struct {
	id   string
	name string
	size uint64
}

type volRow struct {
	name  string
	size  uint64
	links int
}

// parseSystemDF parses the "Containers space usage:" and
// "Local Volumes space usage:" sections of `podman system df -v`. Row parsing
// is positional from the right because COMMAND/CREATED fields contain spaces.
func parseSystemDF(out string) (containers []dfRow, volumes []volRow, volumeLinks []volRow) {
	lines := strings.Split(out, "\n")
	section := ""
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if strings.HasPrefix(line, "Containers space usage:") {
			section = "containers"
			continue
		}
		if strings.HasPrefix(line, "Local Volumes space usage:") {
			section = "volumes"
			continue
		}
		if strings.HasPrefix(line, "Images space usage:") {
			section = "images"
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "CONTAINER ID") || strings.HasPrefix(trimmed, "VOLUME NAME") || strings.HasPrefix(trimmed, "IMAGE ID") || strings.HasPrefix(trimmed, "REPOSITORY") {
			continue
		}
		switch section {
		case "containers":
			toks := strings.Fields(line)
			if len(toks) < 6 {
				continue
			}
			// Rightmost: NAME STATUS CREATED(2 tokens) SIZE LOCAL_VOLUMES ID
			sz, ok := parseDfBytes(toks[len(toks)-5])
			if !ok {
				continue
			}
			id := strings.TrimPrefix(toks[0], "sha256:")
			containers = append(containers, dfRow{id: id, name: toks[len(toks)-1], size: sz})
		case "volumes":
			toks := strings.Fields(line)
			if len(toks) < 3 {
				continue
			}
			sz, ok := parseDfBytes(toks[len(toks)-1])
			if !ok {
				continue
			}
			links, _ := strconv.Atoi(toks[len(toks)-2])
			name := strings.Join(toks[:len(toks)-2], " ")
			volumes = append(volumes, volRow{name: name, size: sz})
			volumeLinks = append(volumeLinks, volRow{name: name, links: links})
		}
	}
	return containers, volumes, volumeLinks
}

func parseDfBytes(s string) (uint64, bool) {
	s = strings.TrimSpace(s)
	mult := uint64(1)
	switch {
	case strings.HasSuffix(s, "kB"):
		mult, s = 1<<10, strings.TrimSuffix(s, "kB")
	case strings.HasSuffix(s, "KB"):
		mult, s = 1<<10, strings.TrimSuffix(s, "KB")
	case strings.HasSuffix(s, "MB"):
		mult, s = 1<<20, strings.TrimSuffix(s, "MB")
	case strings.HasSuffix(s, "GB"):
		mult, s = 1<<30, strings.TrimSuffix(s, "GB")
	case strings.HasSuffix(s, "TB"):
		mult, s = 1<<40, strings.TrimSuffix(s, "TB")
	case strings.HasSuffix(s, "B"):
		mult, s = 1, strings.TrimSuffix(s, "B")
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, false
	}
	return uint64(v * float64(mult)), true
}
