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

	fmt.Printf("sort domains: %#v\n", domains)

	type User struct {
		Name string
		Age int
	}

	var a map[string]string
	fmt.Printf("map a is nil: %v\n", utils.IsInterfaceNil(a))

	var users []User
	fmt.Printf("struct is nil: %v\n", utils.IsInterfaceNil(users))

	if err := utils.JsonListUnmarshal([]string{
		"{\"Name\": \"a\", \"Age\": 20}",
		"{\"Name\": \"b\", \"Age\": 21}"}, &users); err != nil {
		fmt.Printf("err: %s\n", err.Error())
	}

	fmt.Printf("json to slice %#v\n", users)

	j := `{
	"a": {
		"b": [
			{
				"Name": "A",
				"Age": 20
			}
		]
	}
	}`

	var user User
	if err := utils.JsonExtractIntoPtr(j, &user, "a.b.0"); err != nil {
		fmt.Printf("err: %s\n", err.Error())
	}

	fmt.Printf("json with label %#v\n", user)

}

