package router

import (
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"
)

var Router *httprouter.Router

func init() {
	Router = httprouter.New()
}

func Listen() {
	handler := cors.Default().Handler(Router)
	http.ListenAndServe(":"+os.Getenv("PORT"), handler)
}
