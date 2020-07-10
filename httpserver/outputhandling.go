package httpserver

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

func LoadPage(w http.ResponseWriter, r *http.Request) error {
	page, err := ioutil.ReadFile("resources/send.html")
	if err != nil {
		return errors.New("Error loading page send.html")
	}
	fmt.Fprintf(w, "%s", page)
	return nil
}
