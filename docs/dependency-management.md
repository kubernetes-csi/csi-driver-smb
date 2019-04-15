# Dependency management

cloud-provider-azure uses [go modules] for Go dependency management.

## Usage

Run [`hack/update-dependencies.sh`] whenever vendored dependencies change.
This takes a minute to complete.

### Updating dependencies

New dependencies causes golang to recompute the minor version used for each major version of each dependency. And
golang automatically removes dependencies that nothing imports any more.

To upgrade to the latest version for all direct and indirect dependencies of the current module:

* run `go get -u <package>` to use the latest minor or patch releases
* run `go get -u=patch <package>` to use the latest patch releases
* run `go get <pacakge>@VERSION` to use the specified version

You can also manually editing `go.mod`.

Always run `hack/update-dependencies.sh` after changing `go.mod` by any of these methods (or adding new imports).

See golang's [go.mod], [Using Go Modules] and [Kubernetes Go modules] docs for more details.


[go.mod]: https://github.com/golang/go/wiki/Modules#gomod
[go modules]: https://github.com/golang/go/wiki/Modules
[`hack/update-dependencies.sh`]: hack/update-dependencies.sh
[Using Go Modules]: https://blog.golang.org/using-go-modules
[[Kubernetes Go modules]: https://github.com/kubernetes/enhancements/blob/master/keps/sig-architecture/2019-03-19-go-modules.md
