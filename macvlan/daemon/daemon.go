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
	"github.com/fmzhen/docker-macvlan/macvlan/utils"
	"github.com/gorilla/mux"
)

type EnvConfig struct {
	TYPE   string
	IP     string
	GW     string
	HOSTIF string
}

func Listen(absSocket string) {
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

	// create a container
	//var dockerName string
	r.ParseForm()
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

	data, err := utils.DoRequest(httpClient, u, method, path, body, h2)
	if err != nil {
		log.Fatalln("dorequest error: ", err)
	}

	if method == "POST" && strings.Contains(path, "/containers/create") {
		env := utils.GetEnv(body)
		if _, ok := env["TYPE"]; ok {
			if env["TYPE"] == "dhcp" {

			} else if env["TYPE"] == "flat" {

			}
		}
		if strings.Contains(path, "?name=") {
			//dockerName = strings.Join(r.Form["name"], "")
		} else {
			json.Unmarshal(data)
			//dockerName =
		}
	}
	fmt.Printf("%s", data)
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
