package main

import (
	"net/http"

	"lark_web"
)

func main() {
	r := lark_web.Default()
	r.GET("/", func(c *lark_web.Context) {
		c.String(http.StatusOK, "Hello world\n")
	})
	// index out of range for testing Recovery()
	r.GET("/panic", func(c *lark_web.Context) {
		names := []string{"world"}
		c.String(http.StatusOK, names[100])
	})
	r.Run(":9999")
}
