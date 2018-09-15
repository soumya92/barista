---
title: OAuth
import_as: baristaOauth
---

To simplify some aspects of oauth (especially token management), barista provides
a simplified interface through the `oauth` package.

## For Modules

**Before Stream()**, usually during construction, a module should `Register` an
[oauth2 configuration](https://godoc.org/golang.org/x/oauth2#Config), and save
the resulting value.

```go
config, _ := google.ConfigFromJSON([]byte(/* json */), /* scopes */)
clientConfig = baristaOauth.Register(config)
```

Later, during Stream, an [http client](https://golang.org/pkg/net/http/#Client)
configured to use the saved oauth token can be obtained using `Client()`

```go
client := clientConfig.Client()
for {
	resp, err := client.Do(/* request */)
	if sink.Error(err) {
		return
	}
	sink.Output(someFunction(resp))
	<-scheduler.Tick()
}
```

Constructing the client in `Stream` also means that if the token is invalid, the
user can go through the interactive flow and simply click to restart the module,
which will pick up the newly saved token.

## For Users

**IMPORTANT**: Oauth tokens are very sensitive. Barista makes every effort to
ensure that the tokens are secured, by encrypting each token with a uniquely
salted PBKDF2 derived key. However it still needs a master encryption key from
which all other keys are derived, which can be considerd a password of sorts.

The key can be set using `SetEncryptionKey([]byte)`. The recommended way to do
this is to create a random key (using [`rand.Read`](https://golang.org/pkg/crypto/rand/#Read)),
and storing the generated key for re-use using libsecret or equivalent.

Directly calling `SetEncryptionKey([]byte("password"))` is highly discouraged.

To save oauth tokens for all modules configured in a bar instance, simply run
the binary from the command line with the argument `setup-oauth`

```shell
~/bin/mybar setup-oauth
```

Tokens are saved in `~/.config/barista/oauth` (or `$XDG_CONFIG_HOME/barista/oauth`),
so if you encounter any trouble with a specific provider, you can simply delete
associated files and re-run the oauth setup.
