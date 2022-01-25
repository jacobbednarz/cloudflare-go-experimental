# cloudflare-go-experimental

An experimental fork of the cloudflare-go library. Not ready to be used.

## Improvements

### Namespaced functionality

Allows importing of specific functionality instead ofthe whole library. Example: `import github.com/cloudflare/cloudflare-go/zone`

### Consistent CRUD method signatures

Majority of entities follow a standard method signature.

- `Get(id, ...params)`: fetches a single entity
- `List(...params)`: fetches all entities and automatically paginates
- `Create(...params)`: creates a new entity with the provided parameters
- `Delete(id)`: deletes a single entity
