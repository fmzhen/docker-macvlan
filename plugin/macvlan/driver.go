package macvlan

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/libnetwork/ipallocator"
	"github.com/gorilla/mux"
	"github.com/samalba/dockerclient"
	"github.com/vishvananda/netlink"
)

const (
	MethodReceiver       = "NetworkDriver"
	bridgeMode           = "bridge"
	defaultRoute         = "0.0.0.0/0"
	containerIfacePrefix = "eth"
	defaultMTU           = 1500
	minMTU               = 68
)

type Driver interface {
	Listen(string) error
}

// Struct for binding bridge options CLI flags
type bridgeOpts struct {
	brName   string
	brSubnet net.IPNet
	brIP     net.IPNet
}

type driver struct {
	dockerer
	pluginConfig
	ipAllocator *ipallocator.IPAllocator
	version     string
	network     string
	cidr        *net.IPNet
	nameserver  string
}

// Struct for binding plugin specific configurations (cli.go for details).
type pluginConfig struct {
	mtu             int
	mode            string
	hostIface       string
	containerSubnet *net.IPNet
	gatewayIP       net.IP
}

func New(version string, ctx *cli.Context) (Driver, error) {
	docker, err := dockerclient.NewDockerClient("unix:///var/run/docker.sock", nil)
	if err != nil {
		return nil, fmt.Errorf("could not connect to docker: %s", err)
	}
	if ctx.String("host-interface") == "" {
		log.Fatalf("Required flag [ host-interface ] that is used for off box communication was not defined. Example: --host-interface=eth1")
	}

	// bind CLI opts to the user config struct
	if ok := validateHostIface(ctx.String("host-interface")); !ok {
		log.Fatalf("Requird field [ host-interface ] ethernet interface [ %s ] was not found. Exiting since this is required for both l2 and l3 modes.", ctx.String("host-interface"))
	}
	macvlanEthIface = ctx.String("host-interface")

	// lower bound of v4 MTU is 68-bytes per rfc791
	if ctx.Int("mtu") <= 0 {
		cliMTU = cliMTU
	} else if ctx.Int("mtu") >= minMTU {
		cliMTU = ctx.Int("mtu")
	} else {
		log.Fatalf("The MTU value passed [ %d ] must be greater than [ %d ] bytes per rfc791", ctx.Int("mtu"), minMTU)
	}

	// Parse the container IP subnet and network addr to be used to guess the gateway if necessary
	containerGW, containerNet, err := net.ParseCIDR(ctx.String("macvlan-subnet"))
	if err != nil {
		log.Fatalf("Error parsing cidr from the subnet flag provided [ %s ]: %s", ctx.String("macvlan-subnet"), err)
	}

	// Set the default mode to bridge
	if ctx.String("mode") == "" {
		macvlanMode = bridgeMode
	}
	switch ctx.String("mode") {
	case bridgeMode:
		macvlanMode = bridgeMode
		// todo: in other modes if relevant
	default:
		log.Fatalf("Invalid macvlan mode supplied [ %s ] we currently only support [ %s ] mode. If mode is left empty, bridge is the default mode.", ctx.String("mode"), macvlanMode)
	}

	// if no gateway was passed, use the first valid addr on the container subnet
	if ctx.String("gateway") != "" {
		// bind the container gateway to the IP passed from the CLI
		cliGateway := net.ParseIP(ctx.String("gateway"))
		if err != nil {
			log.Fatalf("The IP passed with the [ gateway ] flag [ %s ] was not a valid address: %s", ctx.String("gateway"), err)
		}
		containerGW = cliGateway
	} else {
		// default network IP plus 1. For example 192.168.1.0 is network IP, 192.168.1.1 is GW.
		containerGW = ipIncrement(containerGW)
	}

	pluginOpts := &pluginConfig{
		mtu:             cliMTU,
		mode:            macvlanMode,
		containerSubnet: containerNet,
		gatewayIP:       containerGW,
		hostIface:       macvlanEthIface,
	}
	// containerGW\ containerNet should pass to defaultSubnet/gatewayIP which will be use in ipallocator
	defaultSubnet = containerNet
	gatewayIP = containerGW
	// Leaving as info for now to stdout the plugin config
	log.Infof("Plugin configuration options are: \n %s", pluginOpts)

	ipAllocator := ipallocator.New()
	d := &driver{
		dockerer: dockerer{
			client: docker,
		},
		ipAllocator:  ipAllocator,
		version:      version,
		pluginConfig: *pluginOpts,
	}
	return d, nil
}

func (driver *driver) Listen(socket string) error {
	router := mux.NewRouter()
	router.NotFoundHandler = http.HandlerFunc(notFound)

	router.Methods("GET").Path("/status").HandlerFunc(driver.status)
	router.Methods("POST").Path("/Plugin.Activate").HandlerFunc(driver.handshake)

	handleMethod := func(method string, h http.HandlerFunc) {
		router.Methods("POST").Path(fmt.Sprintf("/%s.%s", MethodReceiver, method)).HandlerFunc(h)
	}
	handleMethod("CreateNetwork", driver.createNetwork)
	handleMethod("DeleteNetwork", driver.deleteNetwork)
	handleMethod("CreateEndpoint", driver.createEndpoint)
	handleMethod("DeleteEndpoint", driver.deleteEndpoint)
	handleMethod("EndpointOperInfo", driver.infoEndpoint)
	handleMethod("Join", driver.joinEndpoint)
	handleMethod("Leave", driver.leaveEndpoint)
	var (
		listener net.Listener
		err      error
	)

	listener, err = net.Listen("unix", socket)
	if err != nil {
		return err
	}

	return http.Serve(listener, router)
}

func notFound(w http.ResponseWriter, r *http.Request) {
	log.Warnf("[plugin] Not found: %+v", r)
	http.NotFound(w, r)
}

func sendError(w http.ResponseWriter, msg string, code int) {
	log.Errorf("%d %s", code, msg)
	http.Error(w, msg, code)
}

func errorResponsef(w http.ResponseWriter, fmtString string, item ...interface{}) {
	json.NewEncoder(w).Encode(map[string]string{
		"Err": fmt.Sprintf(fmtString, item...),
	})
}

func objectResponse(w http.ResponseWriter, obj interface{}) {
	if err := json.NewEncoder(w).Encode(obj); err != nil {
		sendError(w, "Could not JSON encode response", http.StatusInternalServerError)
		return
	}
}

func emptyResponse(w http.ResponseWriter) {
	json.NewEncoder(w).Encode(map[string]string{})
}

type handshakeResp struct {
	Implements []string
}

func (driver *driver) handshake(w http.ResponseWriter, r *http.Request) {
	err := json.NewEncoder(w).Encode(&handshakeResp{
		[]string{"NetworkDriver"},
	})
	if err != nil {
		log.Fatalf("handshake encode: %s", err)
		sendError(w, "encode error", http.StatusInternalServerError)
		return
	}
	log.Debug("Handshake completed")
}

func (driver *driver) status(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, fmt.Sprintln("macvlan plugin", driver.version))
}

type networkCreate struct {
	NetworkID string
	Options   map[string]interface{}
}

func (driver *driver) createNetwork(w http.ResponseWriter, r *http.Request) {
	var create networkCreate
	err := json.NewDecoder(r.Body).Decode(&create)
	if err != nil {
		sendError(w, "Unable to decode JSON payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	if driver.network != "" {
		errorResponsef(w, "You get just one network, and you already made %s", driver.network)
		return
	}
	driver.network = create.NetworkID

	// Parse the network address from the user supplied or default container network
	_, ipNet, err := net.ParseCIDR(defaultSubnet)
	if err != nil {
		log.Warnf("Error parsing cidr from the default subnet: %s", err)
	}
	driver.cidr = ipNet
	driver.ipAllocator.RequestIP(ipNet, nil)
	emptyResponse(w)
}

type networkDelete struct {
	NetworkID string
}

func (driver *driver) deleteNetwork(w http.ResponseWriter, r *http.Request) {
	var delete networkDelete
	if err := json.NewDecoder(r.Body).Decode(&delete); err != nil {
		sendError(w, "Unable to decode JSON payload: "+err.Error(), http.StatusBadRequest)
		return
	}
	log.Debugf("Delete network request: %+v", &delete)
	if delete.NetworkID != driver.network {
		log.Debugf("network not found: %+v", &delete)
		errorResponsef(w, "Network %s not found", delete.NetworkID)
		return
	}
	driver.network = ""
	emptyResponse(w)
	log.Infof("Destroy network %s", delete.NetworkID)
}

type endpointCreate struct {
	NetworkID  string
	EndpointID string
	Interfaces []*iface
	Options    map[string]interface{}
}

type iface struct {
	ID         int
	SrcName    string
	DstPrefix  string
	Address    string
	MacAddress string
}

type endpointResponse struct {
	Interfaces []*iface
}

func (driver *driver) createEndpoint(w http.ResponseWriter, r *http.Request) {
	var create endpointCreate
	if err := json.NewDecoder(r.Body).Decode(&create); err != nil {
		sendError(w, "Unable to decode JSON payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	netID := create.NetworkID
	endID := create.EndpointID

	if netID != driver.network {
		log.Warnf("Network not found, [ %s ]", netID)
		errorResponsef(w, "No such network %s", netID)
		return
	}

	// Request an IP address from libnetwork based on the cidr scope
	// TODO: Add a user defined static ip addr option
	allocatedIP, err := driver.ipAllocator.RequestIP(driver.cidr, nil)
	if err != nil || allocatedIP == nil {
		log.Errorf("Unable to obtain an IP address from libnetwork ipam: %s", err)
		errorResponsef(w, "%s", err)
		return
	}

	// generate a mac address for the pending container
	mac := makeMac(allocatedIP)

	// Have to convert container IP to a string ip/mask format
	bridgeMask := strings.Split(driver.cidr.String(), "/")
	containerAddress := allocatedIP.String() + "/" + bridgeMask[1]

	log.Infof("The allocated container IP is: [ %s ]", allocatedIP.String())

	respIface := &iface{
		Address:    containerAddress,
		MacAddress: mac,
	}
	resp := &endpointResponse{
		Interfaces: []*iface{respIface},
	}
	objectResponse(w, resp)
	log.Debugf("Create endpoint %s %+v", endID, resp)
}

type endpointDelete struct {
	NetworkID  string
	EndpointID string
}

func (driver *driver) deleteEndpoint(w http.ResponseWriter, r *http.Request) {
	var delete endpointDelete
	if err := json.NewDecoder(r.Body).Decode(&delete); err != nil {
		sendError(w, "Could not decode JSON encode payload", http.StatusBadRequest)
		return
	}
	log.Debugf("Delete endpoint request: %+v", &delete)
	emptyResponse(w)
	// null check cidr in case driver restarted and doesn't know the network to avoid panic
	if driver.cidr == nil {
		return
	}
	// ReleaseIP releases an ip back to a network
	if err := driver.ipAllocator.ReleaseIP(driver.cidr, driver.cidr.IP); err != nil {
		log.Warnf("error releasing IP: %s", err)
	}
	log.Debugf("Delete endpoint %s", delete.EndpointID)

	containerLink := delete.EndpointID[:5]

	log.Infof("Removing unused macvlan link [ %s ] from the removed container", containerLink)
	clink, err := netlink.LinkByName(containerLink)
	if err != nil {
		log.Warnf("Error looking up link [ %s ] object: [ %v ] error: [ %s ]", clink.Attrs().Name, clink, err)
	}
	if err := netlink.LinkDel(clink); err != nil {
		log.Errorf("unable to delete the Macvlan link [ %s ] on leave: %s", clink, err)
	}
}

type endpointInfoReq struct {
	NetworkID  string
	EndpointID string
}

type endpointInfo struct {
	Value map[string]interface{}
}

func (driver *driver) infoEndpoint(w http.ResponseWriter, r *http.Request) {
	var info endpointInfoReq
	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		sendError(w, "Could not decode JSON encode payload", http.StatusBadRequest)
		return
	}
	log.Debugf("Endpoint info request: %+v", &info)
	objectResponse(w, &endpointInfo{Value: map[string]interface{}{}})
	log.Debugf("Endpoint info %s", info.EndpointID)
}

type joinInfo struct {
	InterfaceNames []*iface
	Gateway        string
	GatewayIPv6    string
	HostsPath      string
	ResolvConfPath string
}

type join struct {
	NetworkID  string
	EndpointID string
	SandboxKey string
	Options    map[string]interface{}
}

type staticRoute struct {
	Destination string
	RouteType   int
	NextHop     string
	InterfaceID int
}

type joinResponse struct {
	HostsPath      string
	ResolvConfPath string
	Gateway        string
	InterfaceNames []*iface
	StaticRoutes   []*staticRoute
}

func (driver *driver) joinEndpoint(w http.ResponseWriter, r *http.Request) {
	var j join
	if err := json.NewDecoder(r.Body).Decode(&j); err != nil {
		sendError(w, "Could not decode JSON encode payload", http.StatusBadRequest)
		return
	}
	log.Debugf("Join request: %+v", &j)

	endID := j.EndpointID
	// unique name while still on the common netns
	preMoveName := endID[:5]
	mode, err := setVlanMode(macvlanMode)
	if err != nil {
		log.Errorf("%s", err)
		return
	}
	hostEth, err := netlink.LinkByName(macvlanEthIface)
	if err != nil {
		log.Warnf("Error looking up the parent iface [ %s ] mode: [ %s ] error: [ %s ]", macvlanEthIface, mode, err)
	}
	macvlan := &netlink.Macvlan{
		LinkAttrs: netlink.LinkAttrs{
			Name:        preMoveName,
			ParentIndex: hostEth.Attrs().Index,
		},
		Mode: mode,
	}
	if err := netlink.LinkAdd(macvlan); err != nil {
		log.Warnf("failed to create Macvlan: [ %v ] with the error: %s", macvlan, err)
	}
	log.Infof("Created Macvlan port: [ %s ] using the mode: [ %s ]", macvlan.Name, macvlanMode)

	// SrcName gets renamed to DstPrefix on the container iface
	ifname := &iface{
		SrcName:   macvlan.Name,
		DstPrefix: containerIfacePrefix,
		ID:        0,
	}
	res := &joinResponse{
		InterfaceNames: []*iface{ifname},
		Gateway:        gatewayIP,
	}
	objectResponse(w, res)
	log.Debugf("Join endpoint %s:%s to %s", j.NetworkID, j.EndpointID, j.SandboxKey)
}

type leave struct {
	NetworkID  string
	EndpointID string
}

func (driver *driver) leaveEndpoint(w http.ResponseWriter, r *http.Request) {
	var l leave
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		sendError(w, "Could not decode JSON encode payload", http.StatusBadRequest)
		return
	}
	log.Debugf("Leave request: %+v", &l)
	emptyResponse(w)
	log.Debugf("Leave %s:%s", l.NetworkID, l.EndpointID)
}

// return string representation of user options in pluginConfig for debugging
func (d *pluginConfig) String() string {
	str := fmt.Sprintf(" container subnet: [%s],\n", d.containerSubnet.String())
	str = str + fmt.Sprintf("  container gateway: [%s],\n", d.gatewayIP.String())
	str = str + fmt.Sprintf("  host interface: [%s],\n", d.hostIface)
	str = str + fmt.Sprintf("  mmtu: [%d],\n", d.mtu)
	str = str + fmt.Sprintf("  ipvlan mode: [%s]", d.mode)
	return str
}
