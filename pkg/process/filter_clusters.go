package process

import (
	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"gopkg.in/yaml.v3"
	"os"
)

type accounts struct {
	MeteringAccounts []string `yaml:"meteringAccounts"`
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
	var meteringAccounts = make(map[string]struct{})

	err := yaml.Unmarshal(data, &filter)
	if err != nil {
		return nil, err
	}

	for _, account := range filter.MeteringAccounts {
		meteringAccounts[account] = struct{}{}
	}

	return meteringAccounts, nil
}

func skipRuntime(meteringAccount runtime.RuntimeDTO, filterMeteringAccounts map[string]struct{}) bool {
	_, ok := filterMeteringAccounts[meteringAccount.GlobalAccountID]
	return ok
}
