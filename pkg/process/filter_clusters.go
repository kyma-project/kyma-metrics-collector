package process

import (
	"os"

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

	err := yaml.Unmarshal(data, &filter)
	if err != nil {
		return nil, err
	}

	for _, account := range filter.SkippedGlobalAccounts {
		if err = uuid.Validate(account); err != nil {
			continue
		}

		skippedAccounts[account] = struct{}{}
	}

	return skippedAccounts, nil
}
