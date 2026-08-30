package oracle

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"aether/internal/modules/backups/adapters/container"
	"aether/internal/modules/backups/application"
)

const queryFile = "/tmp/aether-dpdir.sql"

var errDataPumpDirNotFound = errors.New("DATA_PUMP_DIR directory object not found in the database")

type Adapter struct {
	exec container.Executor
}

func New(exec container.Executor) *Adapter {
	return &Adapter{exec: exec}
}

func (a *Adapter) Engine() application.BackupEngine { return application.EngineOracle }
func (a *Adapter) Format() string                   { return "dmp" }
func (a *Adapter) ContentType() string              { return "application/octet-stream" }

func connectString(db application.DBDescriptor, service string) string {
	return fmt.Sprintf("%s/%s@//localhost:1521/%s", db.User, db.Password, service)
}

func failureMessage(stdout, stderr string) string {
	msg := strings.TrimSpace(stdout + " " + stderr)
	msg = strings.ReplaceAll(msg, "\n", " ")
	if len(msg) > 400 {
		msg = msg[len(msg)-400:]
	}
	return msg
}

func (a *Adapter) Validate(ctx context.Context, db application.DBDescriptor) error {
	_, stderr, err := a.exec.ExecAs(ctx, db.ContainerID, "oracle", nil, nil, "expdp", "help=y")
	if err != nil {
		return fmt.Errorf("Oracle Data Pump unavailable: %s", strings.TrimSpace(stderr))
	}
	return nil
}

func (a *Adapter) service(ctx context.Context, db application.DBDescriptor) (string, error) {
	for _, env := range []string{"ORACLE_PDB", "ORACLE_SID"} {
		out, _, err := a.exec.ExecAs(ctx, db.ContainerID, "oracle", nil, nil, "printenv", env)
		if err == nil {
			if svc := strings.TrimSpace(out); svc != "" {
				return svc, nil
			}
		}
	}
	return "", errors.New("ORACLE_PDB/ORACLE_SID not set in the Oracle container")
}

func (a *Adapter) dataPumpDir(ctx context.Context, db application.DBDescriptor) (string, error) {
	script := "SET HEADING OFF\nSET PAGESIZE 0\nSELECT directory_path FROM dba_directories WHERE directory_name = 'DATA_PUMP_DIR';\nEXIT\n"
	if err := a.exec.ExecIn(ctx, db.ContainerID, nil, strings.NewReader(script), io.Discard, io.Discard, "tee", queryFile); err != nil {
		return "", fmt.Errorf("write query script into container: %w", err)
	}
	stdout, stderr, err := a.exec.ExecAs(ctx, db.ContainerID, "oracle", nil, nil, "sqlplus", "-s", "/ as sysdba", "@"+queryFile)
	_, _, _ = a.exec.Exec(ctx, db.ContainerID, nil, "rm", "-f", queryFile)
	if err != nil {
		return "", fmt.Errorf("query DATA_PUMP_DIR failed: %s", failureMessage(stdout, stderr))
	}
	dir := ""
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "/") {
			dir = line
		}
	}
	if dir == "" {
		return "", errDataPumpDirNotFound
	}
	return dir, nil
}

func (a *Adapter) CreateBackup(ctx context.Context, db application.DBDescriptor, dest io.Writer) error {
	svc, err := a.service(ctx, db)
	if err != nil {
		return err
	}
	dir, err := a.dataPumpDir(ctx, db)
	if err != nil {
		return err
	}
	name := fmt.Sprintf("%s-%s.dmp", application.SafeToken(db.DBName), time.Now().UTC().Format("20060102T150405Z"))
	args := []string{
		"expdp", connectString(db, svc),
		"SCHEMAS=" + application.SafeToken(db.DBName),
		"DUMPFILE=" + name,
		"LOGFILE=" + name + ".log",
		"COMPRESSION=ALL",
	}
	stdout, stderr, err := a.exec.ExecAs(ctx, db.ContainerID, "oracle", nil, nil, args...)
	if err != nil {
		return fmt.Errorf("expdp failed: %s", failureMessage(stdout, stderr))
	}
	if err := a.exec.ExecStream(ctx, db.ContainerID, nil, dest, io.Discard, "cat", dir+"/"+name); err != nil {
		return fmt.Errorf("read dump file from container: %w", err)
	}
	_, _, _ = a.exec.Exec(ctx, db.ContainerID, nil, "rm", "-f", dir+"/"+name, dir+"/"+name+".log")
	return nil
}

func (a *Adapter) Restore(ctx context.Context, db application.DBDescriptor, src io.Reader) error {
	svc, err := a.service(ctx, db)
	if err != nil {
		return err
	}
	dir, err := a.dataPumpDir(ctx, db)
	if err != nil {
		return err
	}
	name := fmt.Sprintf("%s-restore-%s.dmp", application.SafeToken(db.DBName), time.Now().UTC().Format("20060102T150405Z"))
	path := dir + "/" + name
	if err := a.exec.ExecIn(ctx, db.ContainerID, nil, src, io.Discard, io.Discard, "tee", path); err != nil {
		return fmt.Errorf("write dump file into container: %w", err)
	}
	args := []string{
		"impdp", connectString(db, svc),
		"DUMPFILE=" + name,
		"LOGFILE=imp-" + name + ".log",
		"TABLE_EXISTS_ACTION=REPLACE",
	}
	stdout, stderr, impErr := a.exec.ExecAs(ctx, db.ContainerID, "oracle", nil, nil, args...)
	if impErr != nil {
		_, _, _ = a.exec.Exec(ctx, db.ContainerID, nil, "rm", "-f", path)
		return fmt.Errorf("impdp failed: %s", failureMessage(stdout, stderr))
	}
	_, _, _ = a.exec.Exec(ctx, db.ContainerID, nil, "rm", "-f", path, path+".log")
	return nil
}
