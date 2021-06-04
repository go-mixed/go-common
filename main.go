package main

import (
	"fmt"
	"go-common/utils"
	"go-common/utils/list"
	"time"
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

	var b = map[string]interface{}{}
	b["a"] = 1
	b["b"] = 3.1415926
	b["c"] = 3 * time.Second
	b["d"] = "string"
	b["e"] = []string{"l1", "l2"}
	keys := utils.MapKeys(b).([]string)
	j, err := utils.JsonMarshal(utils.MapToUrlValues(b, keys))
	if err != nil {
		fmt.Printf("err: %s\n", err.Error())
	}
	fmt.Printf("map keys: %#v\n", keys)
	fmt.Printf("map to values json: %s\n", j)

	var c = map[string]string{}
	c["a"] = "3"
	c["b"] = "4"
	c["v"] = "53"

	var values []string
	t := time.Now()
	for i := 0; i < 100000; i++ {
		values = utils.MapKeys(c).([]string)
	}
	fmt.Printf("map keys: %#v %d\n", values, time.Now().Sub(t) / time.Millisecond)

	t = time.Now()
	for i := 0; i < 100000; i++ {
		values = utils.MapStringKeys(c)
	}
	fmt.Printf("map keys: %#v %d\n", values, time.Now().Sub(t) / time.Millisecond)
	t = time.Now()
	for i := 0; i < 100000; i++ {
		values = utils.MapValues(c).([]string)
	}
	fmt.Printf("map values: %#v %d\n", values, time.Now().Sub(t) / time.Millisecond)

	t = time.Now()
	for i := 0; i < 100000; i++ {
		values = utils.MapStringValues(c)
	}
	fmt.Printf("map values: %#v %d\n", values, time.Now().Sub(t) / time.Millisecond)


	if err := utils.JsonListUnmarshal([]string{
		"{\"Name\": \"a\", \"Age\": 20}",
		"{\"Name\": \"b\", \"Age\": 21}"}, &users); err != nil {
		fmt.Printf("err: %s\n", err.Error())
	}

	fmt.Printf("json to slice %#v\n", users)

	j = `{
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
	if err := utils.JsonExtractIntoPtr([]byte(j), &user, "a.b.0"); err != nil {
		fmt.Printf("err: %s\n", err.Error())
	}

	fmt.Printf("json with label %#v\n", user)

}

