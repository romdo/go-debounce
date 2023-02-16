<h1 align="center">
  debounce
</h1>

<p align="center">
  <strong>
    Go package that provides a number of debounce patterns as a set of easy to
    use functions.
  </strong>
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/romdo/go-debounce"><img src="https://img.shields.io/badge/%E2%80%8B-reference-387b97.svg?logo=go&logoColor=white" alt="Go Reference"></a>
  <a href="https://github.com//romdo/go-debounce/releases"><img src="https://img.shields.io/github/v/tag/romdo/go-debounce?label=release" alt="GitHub tag (latest SemVer)"></a>
  <a href="https://github.com/romdo/go-debounce/actions"><img src="https://img.shields.io/github/actions/workflow/status/romdo/go-debounce/ci.yml?logo=github" alt="Actions Status"></a>
  <a href="https://codeclimate.com/github/romdo/go-debounce"><img src="https://img.shields.io/codeclimate/coverage/romdo/go-debounce.svg?logo=code%20climate" alt="Coverage"></a>
  <a href="https://github.com/romdo/go-debounce/issues"><img src="https://img.shields.io/github/issues-raw/romdo/go-debounce.svg?style=flat&logo=github&logoColor=white" alt="GitHub issues"></a>
  <a href="https://github.com/romdo/go-debounce/pulls"><img src="https://img.shields.io/github/issues-pr-raw/romdo/go-debounce.svg?style=flat&logo=github&logoColor=white" alt="GitHub pull requests"></a>
  <a href="https://github.com/romdo/go-debounce/blob/main/LICENSE"><img src="https://img.shields.io/github/license/romdo/go-debounce.svg?style=flat" alt="License Status"></a>
</p>

Provides functions to debounce function calls, i.e., to ensure that a function
is only executed after a certain amount of time has passed since the last call.

Debouncing can be useful in scenarios where function calls may be triggered
rapidly, such as in response to user input, but the underlying operation is
expensive and only needs to be performed once per batch of calls.

This package provides several debouncing functions, each with different
characteristics:

- [`New`][1]: creates a new debounced function that will wait a fixed duration
  before calling the original function, regardless of how many times the
  debounced function is called in the meantime.
- [`NewMutable`][3]: creates a new debounced function that will wait a fixed
  duration before calling the last function that was passed to the debounced
  function, regardless of how many times the debounced function is called in the
  meantime. This allows the function to be changed dynamically.
- [`NewWithMaxWait`][2]: creates a new debounced function that will wait a fixed
  duration before calling the original function, but will also enforce a maximum
  wait time, after which the function will be called regardless of whether new
  calls have been made. This ensures that the function is not delayed
  indefinitely if calls keep coming in.
- [`NewMutableWithMaxWait`][4]: creates a new debounced function that combines
  the characteristics of NewMutable and NewWithMaxWait, i.e., it will wait a
  fixed duration before calling the last function that was passed to the
  debounced function, but will also enforce a maximum wait time. All debouncing
  functions are safe for concurrent use in goroutines and can be called multiple
  times.

[1]: https://pkg.go.dev/github.com/romdo/go-debounce#New
[2]: https://pkg.go.dev/github.com/romdo/go-debounce#NewWithMaxWait
[3]: https://pkg.go.dev/github.com/romdo/go-debounce#NewMutable
[4]: https://pkg.go.dev/github.com/romdo/go-debounce#NewMutableWithMaxWait

## Import

```go
import "github.com/romdo/go-debounce"
```

## Usage

```go
func main() {
	// Create a new debouncer that will wait 100 milliseconds since the last
	// call before calling the callback function.
	debounced, _ := debounce.New(100*time.Millisecond, func() {
		fmt.Println("Hello, world!")
	})

	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 75ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 150ms
	debounced()
	time.Sleep(150 * time.Millisecond) // +150ms = 300ms, wait expired at 250ms

	// Output:
	// Hello, world!
}
```

## Documentation

Please see the
[Go Reference](https://pkg.go.dev/github.com/romdo/go-debouce#section-documentation).

## Contributing

Contributions to this package are welcome! Please open an issue or a pull
request on GitHub.

## License

[MIT](https://github.com/romdo/go-debounce/blob/main/LICENSE)
