# Susnames

A boardgame.

## Install

To run, you must have [golang](https://go.dev/) and [templ](https://templ.guide/) installed. One note on this, for templ's installation to work, the Go bin directory must be exported to your PATH environment variable - something that isn't a guarantee off the bat from installing Go.

Now, clone this repository, and generate the components using templ:
```sh
git clone git@github.com:MatthewJM96/susnames.git
cd susnames
templ generate
```

You can now spin up the webserver and have a play, you can access the webpage [here](http://localhost:9000/):
```sh
go run .
```
