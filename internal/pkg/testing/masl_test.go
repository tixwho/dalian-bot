package testing

import "testing"

func TestLogin(t *testing.T) {
	stateError(t, Login())

}

func TestStorage(t *testing.T) {
	stateError(t, Storage())
}

func TestLogin2(t *testing.T) {
	stateError(t, Login2())
	stateError(t, Storage())
}

func stateError(t *testing.T, err error) {
	if err != nil {
		t.Errorf("Unwanted error: %v", err)
	}
}
