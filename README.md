This repository contains a random collection of go utility functions largely
unrelated to each other, which however have proved useful across different
projects.

You can install each directory independently, with something like:

    go get github.com/ccontavalli/goutils/templates

for example, and each directory has its own README file with instructions and
inline documentation.

To provide a quick overview:

- [**config/**](config/) contains some tiny wrappers for handling config files.
  For example, use:

      config := &struct { User, Password string}{}
      err := config.ReadJsonConfigFromFile("./myconfig.json", &config)

  to read a json config file into a struct.

- [**template/**](templates/) contains a small wrapper around `html/template`.
  The only reason to use this is that it provides a simple model to create
  template inheritance entirely controlled from files on disk (read: no changes
  to the code necessary).  It is also friendly to `go-binddata`, and allows you
  to override templates
  configurations easily (so, for example, you are not stuck with `{{` and `}}`
  conflicting with `angular.js`templates). Example:

      mytemplates, err := templates.NewStaticTemplatesFromDir("./tpls", nil)
      mytemplates.Expand(
        "news", struct { Name, Address }{ "Mr. Bean", "987 Broadway" }, writer)

- [**email/**](email/) contains a small wrapper around common mail handling
  libraries.  You can use this wrapper to drive your mail sending needs from a
  config file, rather than code. The API is farily simple to use, but it reads
  both the settings and email templates from files, with a template engine of
  your choice. For example:

      sender, err := email.NewMailSenderFromConfigFile("./mail-sending.json",
          json.Unmarshal, mytemplates)

  at time of writing, the config file can specify to send emails via
  [mailgun](https://mailgun.com), a relay smtp server like [gmail](https://gmail.com), or using
  a shell command, like `/usr/sbin sendmail -t`.

  New ones should be trivial to add.

- [**scanner/**](scanner/) to scan all subdirectories and files in a path,
  similar to `filepath.Walk`. The main differences being:
  - breadth first walk rather than depth first.
  - detection of symlink loops.
  - carrying and propagating state across directories (and files). Handy
    if you need to implement something like `.htaccess` parsing, where
    you have to build a state per folder, and propagate it throughout.
  - clean and separate file, directory, and error handling.

- [**token/**](token/), to generate cyrptographically strong cookies for
  authentication, with little external dependencies, compact output (the
  cookie is shorter than many others), and API to support renewal.
  This uses AES-GCM to create a `sealed` (read: tampering is detected)
  cookie, containing pretty much arbitrary data, a timestamp, and a nonce.
  The code will by itself tell you if the token is still valid, or expired.

- [**misc/**](misc/) random functions I really didn't know where else to put.
  Things like removing duplicates from a sorted array of string, to checking
  for the presence of a key in the array, or implementing a string queue.

- [**gin/**](gin/) wrappers around many of the utilities above to use them
  from a [gin gonic](http://github.com/gin-gonic) web server.
