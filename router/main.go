package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"
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

/*
func testHttpAPI(

	w http.ResponseWriter,
	r *http.Request,
	p httprouter.Params,

	) {
		json.NewEncoder(w).Encode(map[string]bool{"http-success": true})
	}
*/

func Listen() {
	Router.GET("/", testHttpsAPI)
	Router.GET("/test-cookies", testCookies)
	handler := cors.Default().Handler(Router)
	http.ListenAndServe(":"+os.Getenv("PORT"), handler)
}

/*
func Listen() {
	// http
	HttpRouter.GET("/testhttpapi", testHttpAPI)
	httpHandler := cors.Default().Handler(HttpRouter)
	go http.ListenAndServe(":"+os.Getenv("PORT"), httpHandler)

	// https
	// always !!! => port 80 for http, port 443 for https
	Router.GET("/testhttpsapi", testHttpsAPI)
	httpsHandler := cors.Default().Handler(Router)

	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Cache:      autocert.DirCache("certs"),
		Email:      os.Getenv("EMAIL_ADDRESS"),
		HostPolicy: autocert.HostWhitelist(os.Getenv("DOMAIN_NAME")),
	}

	server := &http.Server{
		Addr:    ":443",
		Handler: httpsHandler,
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
	}

	go http.ListenAndServe(":80", certManager.HTTPHandler(nil))
	log.Panic(server.ListenAndServeTLS("", ""))
}
*/
