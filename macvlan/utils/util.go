package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/samalba/dockerclient"
	"github.com/vishvananda/netlink"
)

func ValidateHostIface(ifaceStr string) bool {
	_, err := net.InterfaceByName(ifaceStr)
	if err != nil {
		log.Warnf("interface [ %s ] was not found on the host. Please verify that the interface is valid: %s", ifaceStr, err)
		return false
	}
	return true
}

// Generate a mac addr
func MakeMac(ip net.IP) string {
	hw := make(net.HardwareAddr, 6)
	hw[0] = 0x7a
	hw[1] = 0x42
	copy(hw[2:], ip.To4())
	return hw.String()
}

// Return the IPv4 address of a network interface
func GetIfaceAddr(name string) (*net.IPNet, error) {
	iface, err := netlink.LinkByName(name)
	if err != nil {
		return nil, err
	}
	addrs, err := netlink.AddrList(iface, netlink.FAMILY_V4)
	if err != nil {
		return nil, err
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("Interface %s has no IP addresses", name)
	}
	if len(addrs) > 1 {
		log.Infof("Interface [ %v ] has more than 1 IPv4 address. Defaulting to using [ %v ]\n", name, addrs[0].IP)
	}
	return addrs[0].IPNet, nil
}

// GenerateRandomName returns a new name joined with a prefix.  This size
// specified is used to truncate the randomly generated value
func GenerateRandomName(prefix string, size int) (string, error) {
	id := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, id); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(id)[:size], nil
}

func DockerPid(containername string) int {
	docker, err := dockerclient.NewDockerClient("unix:///var/run/docker.sock", nil)
	if err != nil {
		log.Fatal("Fail to connect to Docker daemon")
	}
	dockerInfo, err := docker.InspectContainer(containername)
	if err != nil {
		log.Fatalf("Fail to inspcet the containername: %s", containername)
	}

	return dockerInfo.State.Pid

	//"github.com/fsouza/go-dockerclient"
	//endpoint := "unix:///var/run/docker.sock"
	//client, _ := docker.NewClient(endpoint)
	//client.InspectContainer(containername)

}

// initSock create the socket file if it does not already exist
func initSock(socketFile string, socketPath string) {
	if err := os.MkdirAll(socketPath, 0755); err != nil && !os.IsExist(err) {
		log.Warnf("Could not create net plugin path directory: [ %s ]", err)
	}
	// concatenate the absolute path to the spec file handle
	absFile := fmt.Sprint(socketPath, socketFile)
	// If the plugin socket file already exists, remove it.
	if _, err := os.Stat(absFile); err == nil {
		log.Debugf("socket file [ %s ] already exists, unlinking the old file handle..", absFile)
		removeSock(absFile)
	}
	log.Debugf("The plugin absolute path and handle is [ %s ]", absFile)
}

// removeSock if an old filehandle exists remove it
func removeSock(absFile string) {
	err := os.RemoveAll(absFile)
	if err != nil {
		log.Fatalf("Unable to remove the old socket file [ %s ] due to: %s", absFile, err)
	}
}
