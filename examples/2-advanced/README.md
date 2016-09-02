# Advanced sandwich usage: Simple TODO app.

This example demonstrates more advanced features of sandwich, including:

* Providing interface types to the middleware chain.
  TaskDb is the interface provided to the handlers, the actual value injected
  in main() is a taskDbImpl.

* Using 3rd party middleware (go.auth, go.rice)

* Using a 3rd party router (gorilla/mux)

* Using multiple error handlers, and custom error handlers.
  Most web servers will want to server a custom HTML error page for user-facing
  error pages.  An example of that is included here.  For AJAX calls, however,
  we don't want to serve HTML.  Instead, we always respond with JSON using the sandwich..  With
  sandwich, we the errors returned from handlers are agnostic.  Instead, the
  error handler decides what format to respond in.

* Early exit of the middleware chain via the sandwich.Done error
  See `CheckForFakeLogin()` for usage.

* Auto-generating handler code
  Besides using the normal sandwich middleware flow, this example also shows how
  to generate & use auto-generated handlers.  See the "auto_handlers.go" file
  for an example of the generated code, and the /auto/* endpoints use the
  automatic handlers.

## Google Login (Oauth2 Authentication)

In order to use the Google login, you need an Oauth2 client ID & client secret.
See this animation for an example of getting these values:

![oauth2 client id demo](oauth2-client-id.gif)

More documentation is available at https://developers.google.com/identity/protocols/OAuth2WebServer
