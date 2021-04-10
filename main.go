package go_common

import "fmt"

func version() string {
	return "0.0.0"
}

func init() {

}

func main() {
	fmt.Printf("version: %s", version())
}
