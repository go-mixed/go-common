package main

import (
	"fmt"
	"go-common/utils"
	"go-common/utils/conv"
	"go-common/utils/core"
	http_utils "go-common/utils/http"
	"go-common/utils/io"
	list_utils "go-common/utils/list"
	text_utils "go-common/utils/text"
	"io"
	"time"
)

func version() string {
	return "0.0.0"
}

func init() {

}

func main() {

	fmt.Printf("version: %s\n", version())

	fmt.Printf("Atoi %d\n", conv.Atoi("0000123", 0))

	fmt.Printf("indexOf interface{}: %d\n", list_utils.IndexOf([]string{"1", "2"}, "2"))

	domains := http_utils.Domains{
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
		Age  int
	}

	var a map[string]string
	fmt.Printf("map a is nil: %v\n", core.IsInterfaceNil(a))

	var users []*User
	fmt.Printf("struct is nil: %v\n", core.IsInterfaceNil(users))

	type _b struct {
		A int           `json:"a"`
		B float32       `json:"b"`
		C time.Duration `json:"c"`
		D string        `json:"d"`
		E []string      `json:"e"`
		F time.Time     `json:"-"`
	}
	var b = _b{
		A: 1,
		B: 3.1415926,
		C: 3 * time.Second,
		D: "string",
		E: []string{"l1", "l2"},
	}
	m, err := utils.ToMap(b, "json")
	if err != nil {
		fmt.Printf("err: %s\n", err.Error())
		return
	}
	fmt.Printf("struct to map: %#v\n", m)

	m, err = utils.ToMap([]string{"a", "b", "c"}, "")
	if err != nil {
		fmt.Printf("err: %s\n", err.Error())
		return
	}
	fmt.Printf("struct to map: %#v\n", m)

	b1 := map[interface{}]interface{}{}
	b1["a"] = "v2"
	b1[1] = 123
	b1[3.141] = "pi"
	m, err = utils.ToMap(b1, "")
	if err != nil {
		fmt.Printf("err: %s\n", err.Error())
		return
	}
	fmt.Printf("struct to map: %#v\n", m)

	var c = map[string]string{}
	c["a"] = "3"
	c["b"] = "4"
	c["v"] = "53"

	var values []string
	t := time.Now()
	for i := 0; i < 100000; i++ {
		values = utils.MapKeys(c).([]string)
	}
	fmt.Printf("map keys: %#v %d\n", values, time.Since(t).Milliseconds())

	t = time.Now()
	for i := 0; i < 100000; i++ {
		values = utils.MapStringKeys(c)
	}
	fmt.Printf("map keys: %#v %d\n", values, time.Since(t).Milliseconds())
	t = time.Now()
	for i := 0; i < 100000; i++ {
		values = utils.MapValues(c).([]string)
	}
	fmt.Printf("map values: %#v %d\n", values, time.Since(t).Milliseconds())

	t = time.Now()
	for i := 0; i < 100000; i++ {
		values = utils.MapStringValues(c)
	}
	fmt.Printf("map values: %#v %d\n", values, time.Now().Sub(t)/time.Millisecond)

	if err := text_utils.JsonListUnmarshal([]string{
		"{\"Name\": \"a\", \"Age\": 20}",
		"",
		"{\"Name\": \"b\", \"Age\": 21}"}, &users); err != nil {
		fmt.Printf("err: %s\n", err.Error())
		return
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
	if err := text_utils.JsonExtractIntoPtr([]byte(j), &user, "a.b.0"); err != nil {
		fmt.Printf("err: %s\n", err.Error())
		return
	}

	fmt.Printf("json with label %#v\n", user)

	f, err := io_utils.NewMultipartFileReader([]string{"examples/part1.txt", "examples/part2.txt", "examples/part3.txt"})
	if err != nil {
		fmt.Printf("err: %s\n", err.Error())
		return
	}
	defer f.Close()
	if 45954 != f.Size() {
		fmt.Printf("reader size not equal to 45954")
		return
	}
	f.Seek(1000, io.SeekStart)
	f.DryRead(68000)
	/*
		s = ''
		for i in range(1, 1000):
		    s += "%010dabcdefghijklmnopqrstuvwxyz1234567890" % i
		import hashlib
		print(hashlib.md5(s[1000:69000].encode('utf-8')).hexdigest())
	*/
	fmt.Printf("cross 3 multipart file md5: ab0723708785f96b305a828349858d16 == %x", f.Checksums(nil))

}
