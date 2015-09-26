package utils

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/bitly/go-simplejson"
	"github.com/coreos/etcd/client"
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

func NewHTTPClient(u *url.URL, tlsConfig *tls.Config) *http.Client {
	httpTransport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	switch u.Scheme {
	case "unix":
		socketPath := u.Path
		unixDial := func(proto, addr string) (net.Conn, error) {
			return net.Dial("unix", socketPath)
		}
		httpTransport.Dial = unixDial
		// Override the main URL object so the HTTP lib won't complain
		u.Scheme = "http"
		u.Host = "unix.sock"
		u.Path = ""
	default:
		httpTransport.Dial = func(proto, addr string) (net.Conn, error) {
			return net.Dial(proto, addr)
		}
	}
	return &http.Client{Transport: httpTransport}
}

func DoRequest(client *http.Client, u *url.URL, method string, path string, body []byte, headers map[string]string) (int, []byte, error) {
	b := bytes.NewBuffer(body)

	code, reader, err := DoStreamRequest(client, u, method, path, b, headers)
	if err != nil {
		return 0, nil, err
	}

	defer reader.Close()
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return 0, nil, err
	}
	return code, data, nil
}

func DoStreamRequest(client *http.Client, u *url.URL, method string, path string, in io.Reader, headers map[string]string) (int, io.ReadCloser, error) {
	if (method == "POST" || method == "PUT") && in == nil {
		in = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, u.String()+path, in)
	if err != nil {
		return 0, nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	if headers != nil {
		for header, value := range headers {
			req.Header.Add(header, value)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	if resp.StatusCode == 404 {
		return 0, nil, errors.New("Not found")
	}
	if resp.StatusCode >= 400 {
		return 0, nil, errors.New("error occur: code >= 400")
	}

	return resp.StatusCode, resp.Body, nil
}

//TODO: add the container name validate.

func EtcdClientNew(endpoints []string) client.KeysAPI {
	cfg := client.Config{
		Endpoints:               endpoints,
		Transport:               client.DefaultTransport,
		HeaderTimeoutPerRequest: time.Second,
	}

	c, err := client.New(cfg)
	if err != nil {
		log.Fatalf("etcd client new fail", err)
	}
	kapi := client.NewKeysAPI(c)
	return kapi
	/*
		resp, err = kapi.Set(context.Background(), "foo", "bar", nil)
		if err != nil {
			log.Fatal(err)
		} else {
			log.Print(resp)
		}
	*/
}

// raw "/itd" not "itd" , add the etcd will ignore "/". But better change to "itd"

func GetDockerNameFromID(ID string) string {
	docker, err := dockerclient.NewDockerClient("unix:///var/run/docker.sock", nil)
	if err != nil {
		log.Warnf("Fail to connect to Docker daemon")
	}
	dockerInfo, err := docker.InspectContainer(ID)
	if err != nil {
		log.Warnf("Fail to inspcet the containername ID: %s", ID)
	}

	return strings.TrimPrefix(dockerInfo.Name, "/")
}
func GetEnv(data []byte) map[string]string {
	env := make(map[string]string)
	var dat map[string]interface{}
	json.Unmarshal(data, &dat)
	envInter := dat["Env"].([]interface{})
	for _, v := range envInter {
		sli := strings.Split(v.(string), "=")
		env[sli[0]] = sli[1]
	}
	return env
}

func GetNetworkMode(data []byte) string {
	js, _ := simplejson.NewJson(data)
	networkmode, _ := js.Get("HostConfig").Get("NetworkMode").String()
	return networkmode
}
