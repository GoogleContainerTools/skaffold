mquery
===

```go
import mquery

var m = map[string]interface{}{
	"foo": "bar",
	"hoge": map[string]interface{}{
		"name": "otiai10",
	},
	"fuga": map[int]map[string]interface{}{
		0: {"greet": "Hello"},
		1: {"greet": "こんにちは"},
	},
	"langs":    []string{"Go", "JavaScript", "English"},
	"baz":      nil,
	"required": false,
}

func main() {
    fmt.Println(
        Query(m, "foo"), // "bar"
        Query(m, "hoge.name"), // "otiai10"
        Query(m, "fuga.0.greet"), // "Hello"
        Query(m, "langs.2"), // "English"
        Query(m, "required"), // false
        Query(m, "baz.biz"), // nil
    )
}
```