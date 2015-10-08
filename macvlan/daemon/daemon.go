package daemon

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/etcd/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/coreos/etcd/client"
	"github.com/fmzhen/docker-macvlan/macvlan/dhcp"
	"github.com/fmzhen/docker-macvlan/macvlan/flat"
	"github.com/fmzhen/docker-macvlan/macvlan/utils"
	"github.com/gorilla/mux"
)

var kapi client.KeysAPI

type EnvConfig struct {
	TYPE   string
	IP     string
	GW     string
	HOSTIF string
}

func Listen(absSocket string) {
	//etcd init
	kapi = utils.EtcdClientNew(strings.Split(flat.CliEtcd, ","))

	listener, err := net.Listen("unix", absSocket)
	if err != nil {
		log.Fatalf("net listen error: ", err)
	}
	router := mux.NewRouter()
	// can match all request, ugly method
	router.PathPrefix("/").HandlerFunc(justForward)
	http.Serve(listener, router)
}

func justForward(w http.ResponseWriter, r *http.Request) {
	//conn, err := net.Dial("unix", "/var/run/docker.sock")

	// the request header cann't be set
	//r.Header.Set("Host", "/var/run/docker.sock")
	/*
		fmt.Println("request:", r)
		fmt.Println("url:", r.URL)
		fmt.Println("host", r.Host)
		fmt.Println("header:", r.Header)
		fmt.Println("body:", r.Body)

		//the response is nil, Request.RequestURI (or host, where set uri?) can't be set in client requests.
		httpClient := &http.Client{}
		_, err := httpClient.Do(r)
		if err != nil {
			fmt.Print("response error:", err)
		}
	*/
	// redirect is not support unix scheme maybe ,  fail.
	//http.Redirect(w, r, "/var/run/docker.sock", 301)

	fmt.Println("request:", r)
	fmt.Println("url:", r.URL)
	fmt.Println("host", r.Host)
	fmt.Println("header:", r.Header)
	fmt.Println("body:", r.Body)

	// forward the request
	method := r.Method
	path := r.URL.String()
	body, _ := ioutil.ReadAll(r.Body)

	//inspire from samaba dockerclient
	daemonUrl := "unix:///var/run/docker.sock"
	u, _ := url.Parse(daemonUrl)

	// u : http://unix.sock
	httpClient := utils.NewHTTPClient(u, nil)

	h := r.Header
	h2 := make(map[string]string, len(h))
	for k, vv := range h {
		vv2 := strings.Join(vv, ", ")
		h2[k] = vv2
	}

	// process delete a container,and remove etcd data, this should before Dorequest
	var dockerName string
	var dockerNameOrId string
	if method == "DELETE" {
		sIndex := strings.LastIndex(path, "/")
		eIndex := strings.LastIndex(path, "?")
		if eIndex == -1 {
			dockerNameOrId = path[sIndex+1:]
		} else {
			dockerNameOrId = path[sIndex+1 : eIndex]
		}
		dockerName = utils.GetDockerNameFromID(dockerNameOrId)
	}

	statusCode, data, err := utils.DoRequest(httpClient, u, method, path, body, h2)
	//not should use fatal, it will exit the daemon. wo don;t want to do that.example user rm a not exist container
	if err != nil {
		log.Warnln("dorequest error: ", err)
	}
	fmt.Printf("response: %s \n", data)
	fmt.Printf("statusCode: %v \n", statusCode)

	// process  create a container,and store etcd
	r.ParseForm()
	if method == "POST" && strings.Contains(path, "/containers/create") && statusCode == 201 {
		//get docker network mode
		networkMode := utils.GetNetworkMode(body)
		ok5 := networkMode == "bridge" || networkMode == "none"
		//get docker name
		if strings.Contains(path, "?name=") {
			dockerName = strings.Join(r.Form["name"], "")
		} else {
			var dat map[string]interface{}
			json.Unmarshal(data, &dat)
			dockerId := dat["Id"].(string)
			dockerName = utils.GetDockerNameFromID(dockerId)
		}

		// get env and store etcd
		env := utils.GetEnv(body)
		_, ok1 := env["TYPE"]
		_, ok2 := env["HOSTIF"]
		_, ok3 := env["IP"]
		_, ok4 := env["GW"]
		if ok1 && ok2 && ok5 {
			if env["TYPE"] == "dhcp" {
				key := "/dhcp/" + dockerName
				_, err := kapi.Set(context.Background(), key, env["HOSTIF"], nil)
				if err != nil {
					log.Warnf("set dhcp etcd error,docker name:%s, error: %v", dockerName, err)
				}
				log.Infof("%s writed to dhcp etcd, host-interface: %s \n", dockerName, env["HOSTIF"])
			} else if env["TYPE"] == "flat" && ok3 && ok4 {
				key := "/flat/" + dockerName
				value := env["IP"] + "," + env["GW"] + "," + env["HOSTIF"]
				_, err := kapi.Set(context.Background(), key, value, nil)
				if err != nil {
					log.Warnf("set flat etcd error, docker name: %s, err: %v", dockerName, err)
				}
				log.Infof("%s writed to flat etcd, value: %s \n", dockerName, value)
			}
		}
	}
	// process start a container,and config
	if method == "POST" && strings.Contains(path, "/start") && statusCode == 204 {
		eIndex := strings.LastIndex(path, "/")
		sIndex := strings.LastIndex(path[:eIndex], "/")
		dockerNameOrId = path[sIndex+1 : eIndex]
		dockerName = utils.GetDockerNameFromID(dockerNameOrId)
		if resp1, err1 := kapi.Get(context.Background(), "/dhcp/"+dockerName, nil); err1 == nil {
			value := resp1.Node.Value
			flat.CliCName = dockerName
			flat.CliIF = value
			log.Infof("start the container %s, config the dhcp network from the hostinterface : %s \n", dockerName, value)

			dhcp.AddDHCPNetwork()

			log.Infof("start the container %s complete, and complete the dhcp config \n", dockerName)
		} else if resp2, err2 := kapi.Get(context.Background(), "/flat/"+dockerName, nil); err2 == nil {
			value := resp2.Node.Value
			vv := strings.Split(value, ",")
			flat.CliCName = dockerName
			flat.CliIP = vv[0]
			flat.CligwIP = vv[1]
			flat.CliIF = vv[2]

			fmt.Println("gw is right : ", flat.CligwIP)
			log.Infof("start the container %s, config the flat network form config: %s \n", dockerName, value)

			flat.AddContainerNetworking()

			log.Infof("start the container %s complete, end flat config \n", dockerName)
		}

	}

	// process delete,the dockerName should get before the container delete.
	if method == "DELETE" && statusCode == 204 {
		dhcpkey := "/dhcp/" + dockerName
		flatkey := "/flat/" + dockerName
		kapi.Delete(context.Background(), dhcpkey, nil)
		kapi.Delete(context.Background(), flatkey, nil)
	}

	fmt.Fprintf(w, "%s", data)

}

// test socket avaiable ,hello world
func sayhelloName(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()       //解析参数，默认是不会解析的
	fmt.Println(r.Form) //这些信息是输出到服务器端的打印信息
	fmt.Println("path", r.URL.Path)
	fmt.Println("scheme", r.URL.Scheme)
	fmt.Println(r.Form["url_long"])
	for k, v := range r.Form {
		fmt.Println("key:", k)
		fmt.Println("val:", strings.Join(v, ""))
	}
	fmt.Fprintf(w, "Hello astaxie!") //这个写入到w的是输出到客户端的
}
