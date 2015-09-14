package daemon

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

func Listen(absSocket string) {
	listener, err := net.Listen("unix", absSocket)
	if err != nil {
		log.Fatalf("net listen error: ", err)
	}
	router := mux.NewRouter()

	router.HandleFunc("*", justForward)
	//router.HandleFunc("/", sayhelloName)
	http.Serve(listener, router)
}

func justForward(w http.ResponseWriter, r *http.Request) {
	conn, err := net.Dial("unix", "/var/docker")
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
