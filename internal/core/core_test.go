package core

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/change-me/internal/modules/math"
)

func TestUnitGreetDefault(t *testing.T) {
	if got := Greet("   "); got != "hello, world" {
		t.Fatalf("expected 'hello, world', got %q", got)
	}
}

func TestUnitGreetNamed(t *testing.T) {
	if got := Greet("Ada"); got != "hello, Ada" {
		t.Fatalf("expected 'hello, Ada', got %q", got)
	}
}

func TestUnitMathModule(t *testing.T) {
	mod := math.NewModule()
	res, err := mod.Controller.HandleAdd(15, 27)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Sum != 42 || res.HistoryCount != 1 {
		t.Fatalf("unexpected math result: %+v", res)
	}
	if _, err := mod.Controller.HandleAdd(2_000_000_000, 1); err == nil {
		t.Fatal("expected out-of-bounds error")
	}
}

func TestIntegrationHealthEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", http.NoBody)
	rec := httptest.NewRecorder()
	NewMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("unexpected status: %v", body)
	}
}

func TestIntegrationMathAddEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/math/add?a=15&b=27", http.NoBody)
	rec := httptest.NewRecorder()
	NewMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body math.AddResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if body.Sum != 42 || body.HistoryCount != 1 {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestArchitectureNoConsolePrintInCore(t *testing.T) {
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		text := string(raw)
		if strings.Contains(text, "fmt.Print") || strings.Contains(text, "panic(") {
			t.Fatalf("forbidden print or panic in %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}
}

func TestArchitectureModularLayerBoundaries(t *testing.T) {
	mathDir := filepath.Join("..", "modules", "math")
	readLayer := func(name string) string {
		raw, err := os.ReadFile(filepath.Join(mathDir, name))
		if err != nil {
			t.Fatalf("failed reading %s: %v", name, err)
		}
		return string(raw)
	}
	entity := readLayer("entity.go")
	dto := readLayer("dto.go")
	repo := readLayer("repository.go")
	service := readLayer("service.go")
	controller := readLayer("controller.go")

	for _, forbidden := range []string{"Service", "Controller", "Repository"} {
		if strings.Contains(entity, forbidden) || strings.Contains(dto, forbidden) {
			t.Fatalf("entity/dto leaked %s", forbidden)
		}
	}
	if strings.Contains(repo, "Service") || strings.Contains(repo, "Controller") {
		t.Fatal("repository leaked service/controller")
	}
	if strings.Contains(service, "Controller") {
		t.Fatal("service leaked controller")
	}
	if strings.Contains(controller, "Repository") {
		t.Fatal("controller leaked repository")
	}
}

func TestArchitectureLocksImmutableLintPolicy(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "cmd", "xtask", "lint.go"))
	if err != nil {
		t.Fatalf("failed reading lint.go: %v", err)
	}
	lintSrc := string(raw)
	requiredRules := []string{
		`"unsafe"`,
		`panic`,
		`fmt`,
		`os`,
		`log`,
		`"init"`,
		`InterfaceType`,
		`"bool"`,
		`context`,
		`Sleep`,
		`IsExported`,
		`//nolint`,
		`mutable package var`,
		`golangci-lint`,
		`.golangci.yml`,
		`TypeAssertExpr`,
		`Mutex`,
		`500`,
	}
	for _, token := range requiredRules {
		if !strings.Contains(lintSrc, token) {
			t.Fatalf("cmd/xtask/lint.go missing immutable policy rule: %s", token)
		}
	}
}

func TestArchitectureRaceDetectorAndCooldownPolicy(t *testing.T) {
	mainRaw, err := os.ReadFile(filepath.Join("..", "..", "cmd", "xtask", "main.go"))
	if err != nil {
		t.Fatalf("failed reading main.go: %v", err)
	}
	if !strings.Contains(string(mainRaw), `"-race"`) {
		t.Fatal("cmd/xtask/main.go must run tests with -race enabled")
	}

	depRaw, err := os.ReadFile(filepath.Join("..", "..", ".github", "dependabot.yml"))
	if err != nil {
		t.Fatalf("failed reading dependabot.yml: %v", err)
	}
	if !strings.Contains(string(depRaw), "default-days: 3") {
		t.Fatal("dependabot.yml must enforce default-days: 3")
	}

	for _, forbidden := range []string{".prettierrc", "package.json"} {
		if _, statErr := os.Stat(filepath.Join("..", "..", forbidden)); statErr == nil {
			t.Fatalf("forbidden formatter or js artifact present: %s", forbidden)
		}
	}
}

func TestArchitectureLocksGolangciPolicy(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", ".golangci.yml"))
	if err != nil {
		t.Fatalf("failed reading .golangci.yml: %v", err)
	}
	cfg := string(raw)
	for _, token := range []string{`version: "2"`, `default: none`, `enable: []`, `formatters:`} {
		if !strings.Contains(cfg, token) {
			t.Fatalf(".golangci.yml missing immutable policy token: %s", token)
		}
	}
	for _, formatter := range []string{"gofmt", "goimports", "gofumpt", "gci", "golines", "swaggo"} {
		if strings.Contains(cfg, formatter) {
			t.Fatalf(".golangci.yml must never enable formatter %s", formatter)
		}
	}
	for _, linter := range []string{"bodyclose", "contextcheck", "errcheck", "errorlint", "exhaustive", "forbidigo", "gocritic", "gosec", "govet", "nilerr", "prealloc", "revive", "staticcheck", "unconvert", "wrapcheck"} {
		if !strings.Contains(cfg, linter) {
			t.Fatalf(".golangci.yml missing semantic linter %s", linter)
		}
	}
	for _, option := range []string{"check-type-assertions: true", "check-blank: true", "default-signifies-exhaustive: true", "enable-all: true"} {
		if !strings.Contains(cfg, option) {
			t.Fatalf(".golangci.yml missing locked option: %s", option)
		}
	}
}

func TestArchitectureForbidsFormattersInHooksAndCI(t *testing.T) {
	for _, rel := range []string{".githooks/pre-commit", ".githooks/pre-push", ".github/workflows/validate.yml"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", rel))
		if err != nil {
			t.Fatalf("failed reading %s: %v", rel, err)
		}
		for _, forbidden := range []string{"gofmt", "goimports", "gofumpt", "golangci-lint fmt", "latest"} {
			if strings.Contains(string(raw), forbidden) {
				t.Fatalf("%s must never invoke %s", rel, forbidden)
			}
		}
	}
	validateRaw, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "validate.yml"))
	if err != nil {
		t.Fatalf("failed reading validate.yml: %v", err)
	}
	if !strings.Contains(string(validateRaw), "v2.15.0") {
		t.Fatal("validate.yml must pin golangci-lint v2.15.0")
	}
}
