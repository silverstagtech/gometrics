package measurements

type Poly struct {
	Prefix string
	Name   string
	Fields map[string]interface{}
	Tags   map[string]string
}
