.PHONY: backend-build backend-test backend-vet frontend-build frontend-typecheck

backend-build:
	go build -o /tmp/aether-api ./api/cmd/api

backend-test:
	cd api && go test ./internal/... -count=1 -p 1 -timeout 25m

backend-vet:
	cd api && go vet ./internal/...

frontend-build:
	npm --prefix frontend/web run build

frontend-typecheck:
	cd frontend/web && npx tsc --noEmit
