# TwilyName's website

Contains backend and HTML templates of my website. Demo available at https://twily.name.

# Initial launch

Go version 1.23.9 minimal tested, 1.23 minimal required.

```
$ git clone https://github.com/TwilyName/website.git
$ pushd website
$ go build
$ cp config.yaml{.sample,}
$ $EDITOR config.yaml
$ ./website
$ popd
```
