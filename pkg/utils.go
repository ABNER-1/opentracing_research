package pkg

import (
	"fmt"
	"io/ioutil"
)

func ReadYamlFromFile(filename string) string {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("error happen during read file %s: %v\n", filename, err)
	}
	str := string(bytes)
	fmt.Println("--- raw yaml from file:\n", str)
	return str
}