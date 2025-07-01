package util

import (

	"go.uber.org/automaxprocs/maxprocs"


	log "github.com/sirupsen/logrus"

)


// SetAutoMaxProcs sets the GOMAXPROCS value based on the binary name.
// It suppresses logs for CLI binaries and logs the setting for services.
func SetAutoMaxProcs(isCLI bool) {
	if isCLI {
		_, _ = maxprocs.Set() // Intentionally ignore errors for CLI binaries
	} else {
		_, err := maxprocs.Set(maxprocs.Logger(log.Infof))
		if err != nil {
			log.Errorf("Error setting GOMAXPROCS: %v", err)
		}
	}
}

