package engine_test

import (
	"testing"

	_ "aether/internal/backups/adapters/engine"
	"aether/internal/backups/application"
)

func TestAllEnginesRegistered(t *testing.T) {
	engines := []application.BackupEngine{
		application.EnginePostgres,
		application.EngineMySQL,
		application.EngineMariaDB,
		application.EngineMSSQL,
		application.EngineOracle,
		application.EngineMongoDB,
	}
	for _, e := range engines {
		a, err := application.BackupAdapterFor(e)
		if err != nil {
			t.Fatalf("engine %s not registered: %v", e, err)
		}
		if a.Engine() != e {
			t.Fatalf("registered adapter engine mismatch: got %s want %s", a.Engine(), e)
		}
	}
}

func TestUnknownEngineFails(t *testing.T) {
	if _, err := application.BackupAdapterFor("redis"); err == nil {
		t.Fatal("expected error for unregistered engine")
	}
}
