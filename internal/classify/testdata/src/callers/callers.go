// Package callers provides a test fixture that imports and calls
// functions from the contracts and incidental packages, enabling
// caller analysis testing.
package callers

import (
	"github.com/unbound-force/gaze/internal/classify/testdata/src/contracts"
	"github.com/unbound-force/gaze/internal/classify/testdata/src/incidental"
)

// UseGetData calls contracts.GetData and uses the return value.
// This provides caller evidence that GetData's return is contractual.
func UseGetData() []byte {
	fs := &contracts.FileStore{Path: "/tmp/test"}
	return contracts.GetData(fs)
}

// UseFetchConfig calls FetchConfig and uses both return values.
func UseFetchConfig() map[string]string {
	cfg, err := contracts.FetchConfig("/etc/app.conf")
	if err != nil {
		return nil
	}
	return cfg
}

// UseStore calls Save via the Store interface.
func UseStore(s contracts.Store) error {
	return s.Save([]byte("test data"))
}

// UseProcessItem calls the incidental ProcessItem function
// but does not depend on its log output.
func UseProcessItem() {
	_ = incidental.ProcessItem("item-1")
	_ = incidental.ProcessItem("item-2")
}
