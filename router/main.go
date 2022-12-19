package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"
)

var Router *httprouter.Router
var HttpRouter *httprouter.Router

func init() {
	Router = httprouter.New()
	HttpRouter = httprouter.New()
}

func testHttpsAPI(
	w http.ResponseWriter,
	r *http.Request,
	p httprouter.Params,
) {
	json.NewEncoder(w).Encode(map[string]bool{"https-success": true})
}

func testCookies(
	w http.ResponseWriter,
	r *http.Request,
	p httprouter.Params,
) {
	fmt.Println(r.Cookies())
	json.NewEncoder(w).Encode(map[string]int{"cookie-length": len(r.Cookies())})
}

func Listen() {
	Router.GET("/", testHttpsAPI)
	Router.GET("/test-cookies", testCookies)
	http.ListenAndServe(":"+os.Getenv("PROJMGMT_BACKEND_PORT"), Router)
}
