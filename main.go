package main

import (
	"fmt"
	"go-common/utils"
	"go-common/utils/list"
)

func version() string {
	return "0.0.0"
}

func init() {

}

func main() {
	fmt.Printf("version: %s\n", version())

	fmt.Printf("indexOf interface{}: %d\n", list.IndexOf([]string{"1", "2"}, "2"))

	domains := utils.Domains{
		"*.b.com",
		"b.com",
		"*.com",
		"a.*.com",
		"a*.b.com",
		"c.*.b.com",
		"a.b.com",
		"*",
		"c.b.com",
		"c?.*.b.com",
		"b.*",
	}
	domains = domains.Sort()

	fmt.Printf("sort domains: %#v", domains)
}
