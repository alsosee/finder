# finder-web

A project to bring the Finder experience to the web.

## Local development

Use Make to run the project locally:

```bash
make run arguments="-dir=~/Information/ -he=true"
```

Available arguments:

- `bind`: The address to bind to. Optional. Default: `:8080`.
- `dir`: The directory to serve. Required.
- `he`: Hide file extensions. Optional. Default: `false`.
- `ignore`: Path to a file containing a list of files to ignore. Default: `.ignore`.
