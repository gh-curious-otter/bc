// Package names provides random name generation for agents.
package names

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// Adjectives for name generation (positive, memorable)
var adjectives = []string{
	"swift", "clever", "bright", "keen", "bold",
	"calm", "eager", "fair", "glad", "kind",
	"lively", "merry", "neat", "prime", "quick",
	"ready", "sharp", "smart", "steady", "sure",
	"true", "vivid", "warm", "wise", "zesty",
	"agile", "brave", "crisp", "deft", "epic",
	"fresh", "grand", "happy", "ideal", "jolly",
	"lucid", "noble", "polar", "rapid", "sleek",
	"tidy", "ultra", "vital", "witty", "young",
	"zen", "azure", "coral", "dusky", "frost",
}

// Animals for name generation (distinctive, easy to type)
var animals = []string{
	"falcon", "otter", "panda", "tiger", "eagle",
	"dolphin", "jaguar", "koala", "lemur", "meerkat",
	"narwhal", "osprey", "panther", "quail", "raven",
	"salmon", "toucan", "urchin", "viper", "walrus",
	"xerus", "yak", "zebra", "badger", "condor",
	"dingo", "ermine", "ferret", "gecko", "heron",
	"impala", "jackal", "kiwi", "lobster", "marten",
	"newt", "ocelot", "puffin", "quokka", "raccoon",
	"serval", "tapir", "urial", "vulture", "wombat",
	"axolotl", "bison", "crane", "duck", "elk",
}

// Generator creates random agent names.
type Generator struct {
	rng *rand.Rand
}

// New creates a new name generator.
func New() *Generator {
	return &Generator{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())), //nolint:gosec // not for crypto
	}
}

// Generate creates a random adjective-animal name.
func (g *Generator) Generate() string {
	adj := adjectives[g.rng.Intn(len(adjectives))]
	animal := animals[g.rng.Intn(len(animals))]
	return fmt.Sprintf("%s-%s", adj, animal)
}

// GenerateUnique creates a random name that doesn't exist in the given set.
// Returns error if unable to generate unique name after maxAttempts.
func (g *Generator) GenerateUnique(existing map[string]bool, maxAttempts int) (string, error) {
	if maxAttempts <= 0 {
		maxAttempts = 100
	}

	for i := 0; i < maxAttempts; i++ {
		name := g.Generate()
		if !existing[name] {
			return name, nil
		}
	}

	// Fallback: add numeric suffix
	base := g.Generate()
	for i := 1; i <= 999; i++ {
		name := fmt.Sprintf("%s-%d", base, i)
		if !existing[name] {
			return name, nil
		}
	}

	return "", fmt.Errorf("unable to generate unique name after %d attempts", maxAttempts)
}

// GenerateUniqueFromList creates a random name not in the given list.
func (g *Generator) GenerateUniqueFromList(existingNames []string, maxAttempts int) (string, error) {
	existing := make(map[string]bool, len(existingNames))
	for _, name := range existingNames {
		existing[strings.ToLower(name)] = true
	}
	return g.GenerateUnique(existing, maxAttempts)
}

// DefaultGenerator is a package-level generator for convenience.
var DefaultGenerator = New()

// Generate creates a random name using the default generator.
func Generate() string {
	return DefaultGenerator.Generate()
}

// GenerateUnique creates a unique random name using the default generator.
func GenerateUnique(existing map[string]bool, maxAttempts int) (string, error) {
	return DefaultGenerator.GenerateUnique(existing, maxAttempts)
}

// GenerateUniqueFromList creates a unique random name using the default generator.
func GenerateUniqueFromList(existingNames []string, maxAttempts int) (string, error) {
	return DefaultGenerator.GenerateUniqueFromList(existingNames, maxAttempts)
}
