package main

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/edwingeng/wuid/callback"
)

func main() {
	g := wuid.NewWUID("default", nil)
	g.LoadH24WithCallback(func() (uint64, error) {
		resp, err := http.Get("https://stackoverflow.com/")
		if resp != nil {
			defer resp.Body.Close()
		}
		if err != nil {
			return 0, err
		}

		bytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return 0, err
		}

		fmt.Printf("Page size: %d (%#06x)\n\n", len(bytes), len(bytes))
		return uint64(len(bytes)), nil
	})

	for i := 0; i < 10; i++ {
		fmt.Printf("%#016x\n", g.Next())
	}
}
