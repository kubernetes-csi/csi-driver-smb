package testutil

import "os"

func IsRunningInProw() bool {
	_, ok := os.LookupEnv("AZURE_CREDENTIALS")
	return ok
}
