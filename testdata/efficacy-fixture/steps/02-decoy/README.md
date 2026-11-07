# Notes

A small notes app. The Go API listens on `:8080`. The Svelte page is in `web/`.

## API

- `GET /api/notebooks` and `POST /api/notebooks`
- `GET /api/notes?notebook_id=` and `POST /api/notes`
- `PATCH /api/notes/{id}` and `DELETE /api/notes/{id}`

Two notebooks, Inbox and Archive, are created on an empty database.

## Transfer (current)

Transfer copies the note, then deletes the source (COPY-THEN-DELETE).
