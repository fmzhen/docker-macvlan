package daemon

import (
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
)

// initSock create the socket file if it does not already exist
func InitSock(socketFile string, socketPath string) {
	if err := os.MkdirAll(socketPath, 0755); err != nil && !os.IsExist(err) {
		log.Warnf("Could not create net plugin path directory: [ %s ]", err)
	}
	// concatenate the absolute path to the spec file handle
	absFile := fmt.Sprint(socketPath, socketFile)
	/* If the plugin socket file already exists, remove it.
	if _, err := os.Stat(absFile); err == nil {
		log.Debugf("socket file [ %s ] already exists, unlinking the old file handle..", absFile)
		RemoveSock(absFile)
	}
	*/
	log.Debugf("The plugin absolute path and handle is [ %s ]", absFile)
}

// removeSock if an old filehandle exists remove it
func RemoveSock(absFile string) {
	err := os.RemoveAll(absFile)
	if err != nil {
		log.Fatalf("Unable to remove the old socket file [ %s ] due to: %s", absFile, err)
	}
}
