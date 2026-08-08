// SPIKE-EE: exercita a porta RuntimeDriver com um DockerDriver real.
// Valida H4: a interface expressa o ciclo de vida completo (pull→run→inspect→
// stats→logs→exec→stop→remove) + rede + volume.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"aether/spike/runtimedriver/driver"
)

var step = 0

func ok(format string, args ...interface{}) {
	step++
	fmt.Printf("[%02d] ✓ %s\n", step, fmt.Sprintf(format, args...))
}

func fail(format string, args ...interface{}) {
	fmt.Printf("[%02d] ✗ %s\n", step, fmt.Sprintf(format, args...))
	os.Exit(1)
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	d := driver.NewDockerDriver()

	info, err := d.Info(ctx)
	if err != nil {
		fail("Info: %v", err)
	}
	ok("Info: driver=%s version=%s storage=%s rootless=%v caps=%v",
		info.Driver, info.Version, info.StorageDriver, info.Rootless, info.Capabilities)

	// 1. Pull imagem
	ref := driver.ImageRef{Registry: "docker.io", Repo: "alpine", Tag: "3.20"}
	if _, err := d.Pull(ctx, ref, nil); err != nil {
		fail("Pull: %v", err)
	}
	ok("Pull %s", ref.String())

	// 2. InspectImage
	img, err := d.InspectImage(ctx, ref)
	if err != nil {
		fail("InspectImage: %v", err)
	}
	ok("InspectImage: size=%d layers=%d", img.Size, img.Layers)

	// 3. Network
	netID, err := d.NetworkCreate(ctx, driver.NetworkSpec{Name: "spike-net", Driver: "bridge"})
	if err != nil {
		fail("NetworkCreate: %v", err)
	}
	ok("NetworkCreate %s", netID)
	if n, err := d.NetworkInspect(ctx, netID); err == nil {
		ok("NetworkInspect: driver=%s internal=%v subnet=%q", n.Driver, n.Internal, n.Subnet)
	}

	// 4. Volume
	volID, err := d.VolumeCreate(ctx, driver.VolumeSpec{Name: "spike-vol", Driver: "local"})
	if err != nil {
		fail("VolumeCreate: %v", err)
	}
	ok("VolumeCreate %s", volID)

	// 5. Run container
	handle, err := d.Run(ctx, driver.ContainerSpec{
		Name:      "spike-alpine",
		Image:     ref,
		Command:   []string{"sh", "-c", "echo hello-from-spike; sleep 600"},
		Env:       []string{"SPIKE=1"},
		Ports:     []driver.PortBinding{{ContainerPort: "80", HostPort: "18081"}},
		Networks:  []string{"spike-net"},
		Volumes:   []driver.VolumeMount{{Source: "spike-vol", Target: "/data"}},
		Resources: driver.ResourceSpec{CPUs: 0.25, MemMB: 128},
	})
	if err != nil {
		fail("Run: %v", err)
	}
	ok("Run: id=%s name=%s", handle.ID, handle.Name)

	// 6. Inspect
	ci, err := d.Inspect(ctx, handle.ID)
	if err != nil {
		fail("Inspect: %v", err)
	}
	ok("Inspect: state=%s networks=%v image=%s", ci.State, ci.Networks, ci.Image)

	// 7. Stats
	time.Sleep(1 * time.Second)
	st, err := d.Stats(ctx, handle.ID)
	if err != nil {
		fail("Stats: %v", err)
	}
	ok("Stats: cpu=%2.2f%% mem=%dKiB pids=%d", st.CpuPercent, st.MemBytes/1024, st.Pids)

	// 8. Logs (follow mode não, apenas ler até EOF)
	ls, err := d.Logs(ctx, handle.ID, driver.LogRequest{})
	if err != nil {
		fail("Logs: %v", err)
	}
	data, _ := io.ReadAll(ls.Reader)
	ls.Close()
	ok("Logs: %q", strings.TrimSpace(string(data)))

	// 9. Exec
	ex, err := d.Exec(ctx, handle.ID, driver.ExecRequest{Command: []string{"echo", "exec-ok"}})
	if err != nil {
		fail("Exec: %v", err)
	}
	ok("Exec: exit=%d out=%q", ex.ExitCode, strings.TrimSpace(string(ex.Stdout)))

	// 10. NetworkConnect extra / Disconnect
	if err := d.NetworkDisconnect(ctx, "spike-net", handle.ID); err != nil {
		fail("NetworkDisconnect: %v", err)
	}
	ok("NetworkDisconnect")

	// 11. Restart
	if err := d.Restart(ctx, handle.ID, 2*time.Second); err != nil {
		fail("Restart: %v", err)
	}
	ok("Restart")

	// 12. Stop + Remove
	if err := d.Stop(ctx, handle.ID, 3*time.Second); err != nil {
		fail("Stop: %v", err)
	}
	ok("Stop")
	if err := d.Remove(ctx, handle.ID, false); err != nil {
		fail("Remove: %v", err)
	}
	ok("Remove")

	// 13. Erros normalizados
	_, err = d.Inspect(ctx, "nao-existe")
	if de, ok2 := err.(*driver.DriverError); ok2 {
		ok("Erro normalizado: code=%s", de.Code)
	} else {
		fail("Inspect inexistente deveria retornar DriverError, veio: %v", err)
	}

	// 14. Cleanup rede/volume
	if err := d.NetworkRemove(ctx, netID); err != nil {
		fail("NetworkRemove: %v", err)
	}
	ok("NetworkRemove")
	if err := d.VolumeRemove(ctx, volID, true); err != nil {
		fail("VolumeRemove: %v", err)
	}
	ok("VolumeRemove")

	// 15. Info final
	li, _ := d.ListImages(ctx)
	fmt.Printf("\n=== FIM: interface RuntimeDriver validada contra Docker ===\n")
	fmt.Printf("    imagens locais: %d\n", len(li))
}
