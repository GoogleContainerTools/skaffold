# yaml-patch

`yaml-patch` is a version of Evan Phoenix's
[json-patch](https://github.com/evanphx/json-patch), which is an implementation
of [JavaScript Object Notation (JSON) Patch](https://tools.ietf.org/html/rfc6902),
but for YAML.


## Installing

`go get github.com/krishicks/yaml-patch`

If you want to use the CLI:

`go get github.com/krishicks/yaml-patch/cmd/yaml-patch`

## API

Given the following RFC6902-ish YAML document, `ops`:

```
---
- op: add
  path: /baz/waldo
  value: fred
```

And the following YAML that is to be modified, `src`:

```
---
foo: bar
baz:
  quux: grault
```

Decode the ops file into a patch:

```
patch, err := yamlpatch.DecodePatch(ops)
// handle err
```

Then apply that patch to the document:

```
dst, err := patch.Apply(src)
// handle err

// do something with dst
```

### Example

```
doc := []byte(`---
foo: bar
baz:
  quux: grault
`)

ops := []byte(`---
- op: add
  path: /baz/waldo
  value: fred
`)

patch, err := yamlpatch.DecodePatch(ops)
if err != nil {
  log.Fatalf("decoding patch failed: %s", err)
}

bs, err := patch.Apply(doc)
if err != nil {
  log.Fatalf("applying patch failed: %s", err)
}

fmt.Println(string(bs))
```

```
baz:
  quux: grault
  waldo: fred
foo: bar
```
