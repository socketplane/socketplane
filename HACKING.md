HACKING
=======

How to contribute to SocketPlane

## Development Environment

- [Golang](http://golang.org/doc/code.html)
- [Docker](https://docs.docker.com/installation/#installation)
- [Fig](http://www.fig.sh/install.html)

The system running Docker must have the `openvswitch` kernel module must be loaded for the test suite to be run. You can load it using `modprobe openvswitch`

Support for boot2docker is provided using @dave-tucker's [fork](https://github.com/dave-tucker/boot2docker/tree/openvswitch)

```bash
git clone https://github.com/dave-tucker/boot2docker.git
git checkout openvswitch
docker build -t boot2docker . && docker run --rm boot2docker > boot2docker.iso
# if b2d is running, destroy it
boot2docker destroy
boot2docker init --iso="$(pwd)/boot2docker.iso"
boot2docker up
boot2docker ssh modprobe openvswitch
```

## Workflow

We use the standard GitHub workflow which is made a lot easier by using [`hub`](https://hub.github.com/)

1. Create a fork

        hub fork

2. Create a branch for your work

        # For bugs
        git checkout -b bug/42
        # For long-lived feature branches
        git checkout -b feature/something-cool

3. Make your changes and commit

        git add --all
        git commit -s

4. Push your changes to your GitHub fork

        git push <github-user> <branch-name>

5. Raise a Pull Request

        git pull-request

6. To make changes following a code review, checkout your working branch

        git checkout <branch-name>

7. Make changes and then commit

        git add --all
        git commit --amend
        git push --force

## Running the Tests

To run the tests inside a Docker container

```bash
make test
# or
make test-all
```

To run the tests locally you must run Open vSwitch, either via `fig up -d` or by installing it through your distribution's package manager.

```bash
make test-local
# or
make test-all-local
```
