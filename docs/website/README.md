# KSOT Website

This repo contains the code of the [ksot documentation website](nttcom.github.io/ksot-website),
built using [Hugo](https://gohugo.io/) and the [docsy](https://www.docsy.dev/) theme.

## Requirements

* [NodeJS](https://nodejs.org/):  `>= v18.13.0`
* [Go](https://golang.org/dl/) `>= go1.18`
* [Hugo](https://github.com/gohugoio/hugo/releases) `>= v0.108.0`

Building and running the site locally requires a `extended` version of [Hugo](https://gohugo.io).
You can find out more about how to install Hugo for your environment in our
[Getting started](https://www.docsy.dev/docs/getting-started/#prerequisites-and-installation) guide.


## Running the website locally

From the repo root folder, run:

```bash
git submodule update -f --init --recursive
npm install
hugo server
```

Hugo dev server supports hot reload, you can check your changes on the preview page from http://localhost:1313/.
