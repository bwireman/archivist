# Publish the archive

Export a portable bundle and push to configured destinations.

1. Configure `publish.destinations` in `.archivist.json` with shell commands using `{{bundle}}`.
2. Run `archivist publish <destination-name>`.

Core only writes the bundle; destination commands handle Confluence, git repos, etc.
