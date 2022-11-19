package testing

import (
	"fmt"
	"net/http"
	"os"
	"testing"
)

var done chan bool

func recordAuthCode(w http.ResponseWriter, req *http.Request) {
	values := req.URL.Query()
	authCode := values.Get("code")
	LoginGetToken(authCode)
	w.WriteHeader(http.StatusOK)
	done <- true
}

// pre-test setup
func setup() {
	fmt.Println("Before all tests")
	http.HandleFunc("/myapp/", recordAuthCode)
	go http.ListenAndServe(":80", nil)
}

// post-test teardown
func teardown() {
	fmt.Println("After all tests")
}

// if TestMain() presents, no individual testing methods will be called until m.Run()
func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

/*
func TestLogin(t *testing.T) {
	stateError(t, Login())

}

func TestStorage(t *testing.T) {
	stateError(t, Storage())
}
*/

func TestLogin2(t *testing.T) {
	done = make(chan bool)
	stateError(t, Login2())

	<-done
	stateError(t, Storage())
}

/*
func TestListFiles(t *testing.T) {
	stateError(t, ListFiles())
}

*/

func TestUploadFile(t *testing.T) {
	stateError(t, UploadFile())
}

func stateError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("Unwanted error: %v", err)
	}
}
