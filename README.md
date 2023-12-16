# finder

A project to bring the Finder experience to the web.

It is part of a larger project to store information in a structured way:

* [info](https://github.com/alsosee/info)
* [images](https://github.com/alsosee/images)

On a high level, `finder` takes a `info` directory and generates a static website that served via Cloudflare Pages.
It also procceses the `images` directory and generates thumbnails sprites for all directories and images, and uploads them to Cloudflare R2 Storage.

On a lower level, `finder` walks the `info` directory, using go routines to process each YAML file concurrently.
While doing so, it keeps track of all "connections" between files, to use later in go templates.

## Local development

Use Make to build the static site locally:

```bash
export INPUT_INFO=/<path-to-info-directory>/info
export INPUT_MEDIA=/<path-to-media-directory>/media
export INPUT_STATIC=static
export INPUT_OUTPUT=output

make build
```

Use Cloudflare Wrangler to preview the site locally:

```bash
wrangler pages dev --local-protocol=https output/
```
