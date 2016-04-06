# mccdaq

[![GoDoc][godoc image]][godoc link]
[![License Badge][license image]][LICENSE.txt]

Go library to communicate with Measurement Computing Corporation (MCC)
USB DAQs. Based on [Dr. Warren J. Jasper][jasper]'s [MCC DAQ C-based
drivers][mcc-linux]. Many thanks to Dr. Jasper for his work.

**WARNING:** This code is in active development and should not be
considered a stable API. Currently, it is not idomatic GO code, but I
hope to change that in the near future.

## Supported MCC Devices

Below is a list of supported MCC devices. If you want to add a device,
please open an issue. Better yet, please open a pull request with the
updated code.

- [USB-1608FS-Plus][]


## Installation

```bash
$ go get github.com/gotmc/mccdaq
```

## Dependencies

- [libusb C library][libusb-c] — Library for USB device access
  - OS X: `$ brew install libusb`
  - Debian/Ubuntu: `$ sudo apt-get install -y libusb-1.0-0 libusb-1.0-0-dev`
- [Go libusb][libusb] — Go bindings for the [libusb C library][libusb-c]
  - `$ go get github.com/gotmc/libusb`

## Documentation

Documentation can be found at either:

- <https://godoc.org/github.com/gotmc/libusb>
- <http://localhost:6060/pkg/github.com/gotmc/mccdaq/> after running `$
  godoc -http=:6060`

## Contributing

[mccdaq][] is developed using [Scott Chacon][]'s [GitHub Flow][]. To
contribute, fork [mccdaq][], create a feature branch, and then
submit a [pull request][].  [GitHub Flow][] is summarized as:

- Anything in the `master` branch is deployable
- To work on something new, create a descriptively named branch off of
  `master` (e.g., `new-oauth2-scopes`)
- Commit to that branch locally and regularly push your work to the same
  named branch on the server
- When you need feedback or help, or you think the branch is ready for
  merging, open a [pull request][].
- After someone else has reviewed and signed off on the feature, you can
  merge it into master.
- Once it is merged and pushed to `master`, you can and *should* deploy
  immediately.

## Testing

Prior to submitting a [pull request][], please run:

```bash
$ gofmt
$ golint
$ go vet
$ go test
```

To update and view the test coverage report:

```bash
$ go test -coverprofile coverage.out
$ go tool cover -html coverage.out
```

## License

[mccdaq][] is released under the MIT license. Thanks to Warren J. Jasper
at NCSU for allowing us to use the MIT license for this derivative work
of his MCC drivers. Please see the [LICENSE.txt][] file for more
information.

[GitHub Flow]: http://scottchacon.com/2011/08/31/github-flow.html
[godoc image]: https://godoc.org/github.com/gotmc/mccdaq?status.svg
[godoc link]: https://godoc.org/github.com/gotmc/mccdaq
[jasper]: https://textiles.ncsu.edu/blog/team/warren-jasper/
[libusb]: https://github.com/gotmc/libusb
[libusb-c]: http://libusb.info
[LICENSE.txt]: https://github.com/gotmc/mccdaq/blob/master/LICENSE.txt
[license image]: https://img.shields.io/badge/license-MIT-blue.svg
[mccdaq]: https://github.com/gotmc/mccdaq
[mcc-linux]: http://www.mccdaq.com/daq-software/Linux-Support.aspx
[pull request]: https://help.github.com/articles/using-pull-requests
[Scott Chacon]: http://scottchacon.com/about.html
[usb-1608fs-plus]: http://www.mccdaq.com/usb-data-acquisition/USB-1608FS-Plus.aspx
