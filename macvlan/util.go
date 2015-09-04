package macvlan

import (
	log "github.com/Sirupsen/logrus"
	"net"
)

func validateHostIface(ifaceStr string) bool {
	_, err := net.InterfaceByName(ifaceStr)
	if err != nil {
		log.Warnf("interface [ %s ] was not found on the host. Please verify that the interface is valid: %s", ifaceStr, err)
		return false
	}
	return true
}
