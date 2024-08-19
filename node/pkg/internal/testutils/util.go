package testutils

import (
	"errors"
	"fmt"
	"path"
	"runtime"
)

// MustGetMockGuardianTssStorage returns the path to a mock guardian storage file.
func MustGetMockGuardianTssStorage() string {
	str, err := GetMockGuardianTssStorage(0)
	if err != nil {
		panic(err)
	}
	return str
}

func GetMockGuardianTssStorage(guardianIndex int) (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("could not get runtime.Caller(0)")
	}

	guardianStorageFname := path.Join(path.Dir(file), "testdata", fmt.Sprintf("guardian%d.json", guardianIndex))
	return guardianStorageFname, nil
}
