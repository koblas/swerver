# A golang rewrite of the JavaScript serve tool

A webserve designed for basic local development testing, with a focus on being
lean and feature rich for testing purposes.

# Static

[![Build Status](https://circleci.com/gh/zeit/serve.svg?&style=shield)](https://circleci.com/gh/zeit/serve)
[![Install Size](https://packagephobia.now.sh/badge?p=serve)](https://packagephobia.now.sh/result?p=serve)

Assuming you would like to serve a static site, single page application or just a static file (no matter if on your device or on the local network), this package is just the right choice for you.

It behaves exactly like static deployments on [Now](https://zeit.co/now), so it's perfect for developing your static project. Then, when it's time to push it into production, you [deploy it](https://zeit.co/docs/examples/static).

Furthermore, it provides a neat interface for listing the directory's contents:

### Differences from Serve

Can mimic a true production webserver better:

- Support localhost SSL connections
- Data responses are compresses
- Proxy support for local testing

## Installation

```bash
go install github.com/koblas/swerver
```

The quickest way to get started is to just run `swerver` in your project's directory. By default it will
start on port `5000` and

```bash
swerver
```

Finally, run this command to see a list of all available options:

```bash
swerver --help
```

Now you understand how the package works! :tada:

## Configuration

To customize `serve`'s behavior, create a `serve.json` file in the public folder and insert any of these properties.

| Property                                             | Description                                                           |
| ---------------------------------------------------- | --------------------------------------------------------------------- |
| [`public`](#public-string)                           | Set a sub directory to be served                                      |
| [`cleanUrls`](#cleanurls-booleanarray)               | Have the `.html` extension stripped from paths                        |
| [`proxy`](#proxy-array)                              | Proxy Endpoint                                                        |
| [`rewrites`](#rewrites-array)                        | Rewrite paths to different paths                                      |
| [`redirects`](#redirects-array)                      | Forward paths to different paths or external URLs                     |
| [`headers`](#headers-array)                          | Set custom headers for specific paths                                 |
| [`directoryListing`](#directorylisting-booleanarray) | Disable directory listing or restrict it to certain paths             |
| [`unlisted`](#unlisted-array)                        | Exclude paths from the directory listing                              |
| [`trailingSlash`](#trailingslash-boolean)            | Remove or add trailing slashes to all paths                           |
| [`renderSingle`](#rendersingle-boolean)              | If a directory only contains one file, render it                      |
| [`symlinks`](#symlinks-boolean)                      | Resolve symlinks instead of rendering a 404 error                     |
| [`etag`](#etag-boolean)                              | Calculate a strong `ETag` response header, instead of `Last-Modified` |
| [`ssl`](#ssl-array)                                  | SSL Certificate and Private Key                                       |

### public (String)

By default, the current working directory will be served. If you only want to serve a specific path, you can use this options to pass an absolute path or a custom directory to be served relative to the current working directory.

For example, if serving a [Jekyll](https://jekyllrb.com/) app, it would look like this:

```json
{
  "public": "_site"
}
```

Using absolute path:

```json
{
  "public": "/path/to/your/_site"
}
```

**NOTE:** The path cannot contain globs or regular expressions.

### cleanUrls (Boolean|Array)

By default, all `.html` files can be accessed without their extension.

If one of these extensions is used at the end of a filename, it will automatically perform a redirect with status code [301](https://en.wikipedia.org/wiki/HTTP_301) to the same path, but with the extension dropped.

You can disable the feature like follows:

```json
{
  "cleanUrls": false
}
```

However, you can also restrict it to certain paths:

```json
{
  "cleanUrls": ["/app/**", "/!components/**"]
}
```

**NOTE:** The paths can only contain globs that are matched using [minimatch](https://github.com/isaacs/minimatch).

### proxy (Array)

```json
{
  "proxy": [
    { "source": "/v1/**", "destination": "http://localhost:8081/" }
    { "source": "/**", "destination": "http://localhost:8080/" },
  ]
}
```

### rewrites (Array)

If you want your visitors to receive a response under a certain path, but actually serve a completely different one behind the curtains, this option is what you need.

It's perfect for [single page applications](https://en.wikipedia.org/wiki/Single-page_application) (SPAs), for example:

```json
{
  "rewrites": [
    { "source": "app/**", "destination": "/index.html" },
    { "source": "projects/*/edit", "destination": "/edit-project.html" }
  ]
}
```

You can also use so-called "routing segments" as follows:

```json
{
  "rewrites": [{ "source": "/projects/:id/edit", "destination": "/edit-project-:id.html" }]
}
```

Now, if a visitor accesses `/projects/123/edit`, it will respond with the file `/edit-project-123.html`.

**NOTE:** The paths can contain globs (matched using [minimatch](https://github.com/isaacs/minimatch)) or regular expressions (match using [path-to-regexp](https://github.com/pillarjs/path-to-regexp)).

### redirects (Array)

In order to redirect visits to a certain path to a different one (or even an external URL), you can use this option:

```json
{
  "redirects": [
    { "source": "/from", "destination": "/to" },
    { "source": "/old-pages/**", "destination": "/home" }
  ]
}
```

By default, all of them are performed with the status code [301](https://en.wikipedia.org/wiki/HTTP_301), but this behavior can be adjusted by setting the `type` property directly on the object (see below).

Just like with [rewrites](#rewrites-array), you can also use routing segments:

```json
{
  "redirects": [
    { "source": "/old-docs/:id", "destination": "/new-docs/:id" },
    { "source": "/old", "destination": "/new", "type": 302 }
  ]
}
```

In the example above, `/old-docs/12` would be forwarded to `/new-docs/12` with status code [301](https://en.wikipedia.org/wiki/HTTP_301). In addition `/old` would be forwarded to `/new` with status code [302](https://en.wikipedia.org/wiki/HTTP_302).

**NOTE:** The paths can contain globs (matched using [minimatch](https://github.com/isaacs/minimatch)) or regular expressions (match using [path-to-regexp](https://github.com/pillarjs/path-to-regexp)).

### headers (Array)

Allows you to set custom headers (and overwrite the default ones) for certain paths:

```json
{
  "headers": [
    {
      "source": "**/*.@(jpg|jpeg|gif|png)",
      "headers": [
        {
          "key": "Cache-Control",
          "value": "max-age=7200"
        }
      ]
    },
    {
      "source": "404.html",
      "headers": [
        {
          "key": "Cache-Control",
          "value": "max-age=300"
        }
      ]
    }
  ]
}
```

If you define the `ETag` header for a path, the handler will automatically reply with status code `304` for that path if a request comes in with a matching `If-None-Match` header.

If you set a header `value` to `null` it removes any previous defined header with the same key.

**NOTE:** The paths can only contain globs that are matched using [minimatch](https://github.com/isaacs/minimatch).

### directoryListing (Boolean|Array)

For paths are not files, but directories, the package will automatically render a good-looking list of all the files and directories contained inside that directory.

If you'd like to disable this for all paths, set this option to `false`. Furthermore, you can also restrict it to certain directory paths if you want:

```json
{
  "directoryListing": ["/assets/**", "/!assets/private"]
}
```

**NOTE:** The paths can only contain globs that are matched using [minimatch](https://github.com/isaacs/minimatch).

### unlisted (Array)

In certain cases, you might not want a file or directory to appear in the directory listing. In these situations, there are two ways of solving this problem.

Either you disable the directory listing entirely (like shown [here](#directorylisting-booleanarray)), or you exclude certain paths from those listings by adding them all to this config property.

```json
{
  "unlisted": [".DS_Store", ".git"]
}
```

The items shown above are excluded from the directory listing by default.

**NOTE:** The paths can only contain globs that are matched using [minimatch](https://github.com/isaacs/minimatch).

### trailingSlash (Boolean)

By default, the package will try to make assumptions for when to add trailing slashes to your URLs or not. If you want to remove them, set this property to `false` and `true` if you want to force them on all URLs:

```js
{
  "trailingSlash": true
}
```

With the above config, a request to `/test` would now result in a [301](https://en.wikipedia.org/wiki/HTTP_301) redirect to `/test/`.

### renderSingle (Boolean)

Sometimes you might want to have a directory path actually render a file, if the directory only contains one. This is only useful for any files that are not `.html` files (for those, [`cleanUrls`](#cleanurls-booleanarray) is faster).

This is disabled by default and can be enabled like this:

```js
{
  "renderSingle": true
}
```

After that, if you access your directory `/test` (for example), you will see an image being rendered if the directory contains a single image file.

### symlinks (Boolean)

For security purposes, symlinks are disabled by default. If `serve-handler` encounters a symlink, it will treat it as if it doesn't exist in the first place. In turn, a 404 error is rendered for that path.

However, this behavior can easily be adjusted:

```js
{
  "symlinks": true
}
```

Once this property is set as shown above, all symlinks will automatically be resolved to their targets.

### etag (Boolean)

HTTP response headers will contain a strong [`ETag`][etag] response header, instead of a [`Last-Modified`][last-modified] header. Opt-in because calculating the hash value may be computationally expensive for large files.

Sending an `ETag` header is disabled by default and can be enabled like this:

```js
{
  "etag": true
}
```

## SSL Certificates

See -- https://github.com/FiloSottile/mkcert

## Error templates

The handler will automatically determine the right error format if one occurs and then sends it to the client in that format.

Furthermore, this allows you to not just specifiy an error template for `404` errors, but also for all other errors that can occur (e.g. `400` or `500`).

Just add a `<status-code>.html` file to the root directory and you're good.

## Credits

This is based on the [Serve](https://github.com/zeit/serve) project by Zeit.

## Author

David Koblas ([@koblas](https://twitter.com/koblas))
