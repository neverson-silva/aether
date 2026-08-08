package core

import "testing"

func TestValidateComposeYAML(t *testing.T) {
	good := `services:
  web:
    image: nginx:alpine
    ports: ["80:80"]
    depends_on: [db]
  db:
    image: postgres:16
volumes: {}
networks: {}`
	v := ValidateComposeYAML(good)
	if !v.Valid {
		t.Fatalf("deveria ser válido: %v", v.Errors)
	}
	if len(v.Services) != 2 {
		t.Fatalf("esperado 2 services, obtido %d", len(v.Services))
	}
	bad := "services:\n  web:\n    image: ["
	v2 := ValidateComposeYAML(bad)
	if v2.Valid {
		t.Fatal("compose inválido não detectado")
	}
	if len(v2.Errors) == 0 {
		t.Fatal("sem erros reportados")
	}
	noSvc := "version: '3'"
	v3 := ValidateComposeYAML(noSvc)
	if v3.Valid {
		t.Fatal("compose sem services não detectado")
	}
}
