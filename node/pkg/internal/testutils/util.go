package testutils

import (
	"errors"
	"path"
	"runtime"
)

func MustGetMockGuardianTssStorage() string {
	str, err := GetMockGuardianTssStorage()
	if err != nil {
		panic(err)
	}
	return str
}

func GetMockGuardianTssStorage() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("could not get runtime.Caller(0)")
	}

	mockGuardianAddress := path.Join(path.Dir(file), "testdata", "mock_guardian_storage.json")
	return mockGuardianAddress, nil
}
