package contract

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func Save(path string, contract *Contract) error {
	if contract == nil {
		return fmt.Errorf("contract is nil")
	}

	data, err := json.MarshalIndent(contract, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

func Load(path string) (*Contract, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var c Contract
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}
