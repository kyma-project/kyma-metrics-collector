package process

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type accounts struct {
	SkippedGlobalAccounts []string `yaml:"globalAccounts"`
}

func readFilterFile(file string) ([]byte, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func parseRuntimesToBeFiltered(data []byte) (map[string]struct{}, error) {
	var filter accounts

	skippedAccounts := make(map[string]struct{})
	invalidGlobalAccount := []string{}

	err := yaml.Unmarshal(data, &filter)
	if err != nil {
		return nil, err
	}

	for _, account := range filter.SkippedGlobalAccounts {
		if err = uuid.Validate(account); err != nil {
			invalidGlobalAccount = append(invalidGlobalAccount, account)
			continue
		}

		skippedAccounts[account] = struct{}{}
	}

	if len(invalidGlobalAccount) > 0 {
		return nil, fmt.Errorf("invalid global account IDs: %s", strings.Join(invalidGlobalAccount, ", "))
	}
	return skippedAccounts, nil
}
