package store

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

func TestMigrationPrefixUniqueness(t *testing.T) {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}

	// Match the version prefix including optional letter suffix (e.g. "007", "007b").
	// Two files sharing the same pure numeric prefix AND the same letter suffix would collide.
	prefixRe := regexp.MustCompile(`^(\d{3}[a-z]?)`)
	seen := map[string]string{}

	for _, e := range entries {
		name := e.Name()
		m := prefixRe.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		prefix := m[1]
		if existing, ok := seen[prefix]; ok {
			t.Errorf("duplicate 3-digit prefix %q: %q and %q both share it", prefix, existing, name)
		} else {
			seen[prefix] = name
		}
	}
}

// TestNoNewStrftimeDefaults enforces the policy from migration 020: new migration
// files (019 onward) must not introduce DEFAULT (strftime(...)) columns.
// Existing migrations before 019 are exempt because SQLite cannot drop defaults.
func TestNoNewStrftimeDefaults(t *testing.T) {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}

	policyFromPrefix := "019"
	prefixRe := regexp.MustCompile(`^(\d{3})`)

	for _, e := range entries {
		name := e.Name()
		m := prefixRe.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		if m[1] < policyFromPrefix {
			continue // exempt: predates the policy
		}
		data, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if strings.Contains(string(data), "DEFAULT (strftime") {
			t.Errorf("migration %s violates timestamp policy: contains DEFAULT (strftime(...)); set timestamps in application code instead", name)
		}
	}
}
