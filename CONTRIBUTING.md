Contributing to `rhtap-cli`
---------------------------

In order to contribute to this project you need the following requirements:

- [Golang 1.16 or higher][golang]
- [GNU Make][gnuMake]

All the automation needed for the project lives in the [Makefile](Makefile). This file is the entry point for all the automation tasks in the project for CI, development and release

# Building

After you have cloned the repository and installed the requirements, you can start building. For that purpose you can simply run `make`, the default target builds the application in the `bin` directory

```bash
make
```

# Testing

Unit testing is done using the `go test` command, you can run the tests with the following target:

```bash
make test-unit
```

Alternatively, run all tests with:

```bash
make test
```

# Running

To run the application you can rely on the `run` target, this is the equivalent of `go run` command. For instance:

```bash
make run ARGS='deploy --help'
```

Which is the equivalent of:

```bash
make &&
    bin/rhtap-cli deploy --help
```

[gnuMake]: https://www.gnu.org/software/make/
[golang]: https://golang.org/dl/
