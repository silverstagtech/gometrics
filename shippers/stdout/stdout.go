package stdout

import "fmt"

type StdOut struct {
	onErr func(error) bool
}

func New() *StdOut {
	return &StdOut{
		onErr: func(error) bool { return true },
	}
}

func (_ StdOut) Ship(b []byte) {
	fmt.Println(string(b))
}
