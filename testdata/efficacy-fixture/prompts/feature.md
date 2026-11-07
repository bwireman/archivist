Make `go test ./transfer` pass.

Add package `transfer` and `POST /api/notes/{id}/transfer`.
The JSON body is `{"notebook_id":"<id>"}`.
Use `notes.Handler` for the HTTP server.
