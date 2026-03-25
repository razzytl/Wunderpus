package ags

import (
	"io/ioutil"
	"os"
	"testing"
)

func tempGoalDB(t *testing.T) string {
	f, err := ioutil.TempFile("", "ags_test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}
