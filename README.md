<h1 align="center">
  debounce
</h1>

<p align="center">
  <strong>
    Go package providing a flexible set of debounce operations.
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
expensive and only needs to be performed once for each batch of calls.

## Features

- Static debouncer where the function to invoke is specified up-front.
- Mutable debouncer allowing you to change the underlying function to invoke at
  every call to the debouncer.
- Trailing invocation, where the function is invoked after no calls
  for X period of time. This is the default behavior.
- Leading invocation, where first call immediately invokes the function, but
  subsequent calls within X time of a previous call do not trigger an
  invocation. Can be combined with Trailing behavior.
- Max wait invocation, forcing invocation at least every X time, even during
  bursts of calls that keep delaying invocation. Without this, invocation could
  in theory be blocked forever if the debouncer is always called before wait
  time expires.

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
