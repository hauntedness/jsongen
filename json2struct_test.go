package jsongen

import (
	"testing"
)

func Test_structGen_Exec(t *testing.T) {
	err := Json2StructDir("testdata", "testdata/api", &Options{
		UseJsonNumber: true,
	})
	if err != nil {
		t.Fatal(err)
	}
}
