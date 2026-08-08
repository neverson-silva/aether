package db

import (
	"fmt"
	"sync"
	"testing"
)

func TestConcurrentMigrateAdvisoryLock(t *testing.T) {
	cfg := ConfigFromEnvPublic(t)
	cfg.DatabaseSchema = "t_conc" + schemaName(t.Name())
	sqldb, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		CleanupTestSchema(cfg)
		sqldb.Close()
	})
	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := Migrate(sqldb); err != nil {
				errs <- fmt.Errorf("migrate concorrente: %w", err)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	v, err := Version(sqldb)
	if err != nil || v != 22 {
		t.Fatalf("versão final: %d %v", v, err)
	}
}
