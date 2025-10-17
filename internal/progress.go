package internal

import (
	"fmt"
	"strconv"

	"github.com/zondax/fil-trace-check/api"
	"github.com/zondax/fil-trace-check/internal/types"
)

const (
	ProgressOK = "ok"
)

func UpdateProgressHeight(height int64, success bool, message string, db *api.DB) {
	progress := types.Progress{
		Success: success,
		Message: message,
	}
	if err := db.Insert(strconv.FormatInt(height, 10), progress); err != nil {
		panic(fmt.Errorf("failed to update progress: %s", err))
	}
}

func UpdateProgressAddress(address string, height int64, success bool, message string, db *api.DB) {
	progress := types.Progress{
		Success: success,
		Message: message,
	}
	if err := db.Insert(address+api.AddressHeightSeparator+strconv.FormatInt(height, 10), progress); err != nil {
		panic(fmt.Errorf("failed to update progress: %s", err))
	}
}

func GetProgressAddressState(address string, state any, stateDB *api.DB) error {
	if err := stateDB.Get(address, state); err != nil {
		return fmt.Errorf("failed to get state: %s", err)
	}
	return nil
}

func UpdateProgressAddressState(address string, state any, stateDB *api.DB) error {
	if err := stateDB.Insert(address, state); err != nil {
		return fmt.Errorf("failed to update state: %s", err)
	}
	return nil
}
