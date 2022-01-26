# cloudflare-go-experimental

An experimental fork of the cloudflare-go library. Not ready to be used.

## Improvements

### Namespaced functionality

Allows importing of specific functionality instead ofthe whole library.
Example: `import github.com/cloudflare/cloudflare-go/zone`

### Consistent CRUD method signatures

Majority of entities follow a standard method signature.

- `Get(id, ...params)`: fetches a single entity
- `List(...params)`: fetches all entities and automatically paginates
- `Create(...params)`: creates a new entity with the provided parameters
- `Update(id, ...params)`: updates an existing entity
- `Delete(id)`: deletes a single entity


## Examples

A zone is used below for the examples however, all entites will implement the
same methods and interfaces.

**simple approach for initialising a new client with an API token**

```go
cloudflare.Token = "126a8a51b9d1bbd07fddc65819a542c3"
// do stuff
```

**simple approach for initialising a new client with an API key/email combination**

```go
cloudflare.Key = "3bc3be114fb6323adc5b0ad7422d193a"
cloudflare.Email = "someone@example.com"
// do stuff
```

NOTE: setting these values at the top level (`cloudflare.Key` or
`cloudflare.Token`) sets the value globally and overwrites individual client
instantiations.

**more advanced approach for initialising a new client with options like your
own `http.Client` (recommended)**

```go
params := cloudflare.ClientParams{
  Key: "3bc3be114fb6323adc5b0ad7422d193a",
  Email: "someone@example.com",
  HTTPClient: myCustomHTTPClient,
  // ...
}
c, err := cloudflare.New(params)
```

**create a new zone**

```go
cloudflare.Key = "3bc3be114fb6323adc5b0ad7422d193a",
cloudflare.Email = "someone@example.com"

zParams := &cloudflare.ZoneParams{
  Name: "example.com",
  AccountID: "d8e8fca2dc0f896fd7cb4cb0031ba249"
}
z, _ := zone.New(zParams)
```

**fetching a known zone ID**

```go
cloudflare.Key = "3bc3be114fb6323adc5b0ad7422d193a",
cloudflare.Email = "someone@example.com"
z, _ := zone.Get("3e7705498e8be60520841409ebc69bc1", nil)
```

**fetching all zones matching a single account ID**

```go
cloudflare.Key = "3bc3be114fb6323adc5b0ad7422d193a",
cloudflare.Email = "someone@example.com"

zParams := &cloudflare.ZoneParams{
  AccountID: "d8e8fca2dc0f896fd7cb4cb0031ba249"
}
z, _ := zone.List(zParams)
```

**update a zone**

```go
cloudflare.Key = "3bc3be114fb6323adc5b0ad7422d193a",
cloudflare.Email = "someone@example.com"

zParams := &cloudflare.ZoneParams{
  Nameservers: cloudflare.StringSlice([]string{
    "ns1.example.com",
    "ns2.example.com"
  })
}
z, _ := zone.Update("b5163cf270a3fbac34827c4a2713eef4", zParams)
```

**delete a zone**

```go
cloudflare.Key = "3bc3be114fb6323adc5b0ad7422d193a",
cloudflare.Email = "someone@example.com"
z, _ := zone.Delete("b5163cf270a3fbac34827c4a2713eef4")
```
