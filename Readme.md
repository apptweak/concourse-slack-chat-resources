# Slack Chat Resources

This repository provides Concourse resource types to read, act on, and reply to messages on Slack.

There are two resource types:

- `slack-read-resource`: For reading messages. (uses [chat.postMessage](https://api.slack.com/methods/chat.postMessage))
- `slack-post-resource`: For posting/updating messages. Uploading files (uses [channels.history](https://slack.com/api/channels.history) or [chat.update](https://api.slack.com/methods/chat.update), [files.upload](https://api.slack.com/methods/files.upload)

There are two resource types because a system does not want to respond to messages that it posts itself. Concourse assumes that an output of a resource is also a valid input. Therefore, separate resources are used for reading and posting. Since using a single resource has no benefits over separate resources, reading and posting are split into two resource types.

The posting resource offers similar functionality to the older
[slack-notification-resource](https://github.com/cloudfoundry-community/slack-notification-resource),
with the following benefits:

- Support for replying to threads.
- More powerful interpolation of contents of arbitrary files into message text and other parameters.
- Written in Go as opposed to Bash, in case you care about this :).

## How to deploy

```bash
  # Build and push 2 images to GHCR: ghcr.io/apptweak/slack-read-resource and slack-post-resource
  make all

  # Login example (CI recommended):
  # gh auth token | docker login ghcr.io -u $(gh api user --jq .login) --password-stdin
```


Images on GHCR:

- [ghcr.io/apptweak/slack-read-resource](https://github.com/orgs/apptweak/packages/container/package/slack-read-resource)
- [ghcr.io/apptweak/slack-post-resource](https://github.com/orgs/apptweak/packages/container/package/slack-post-resource)

## Version Format

Timestamps of Slack messages are used as resource versions. For example, a message may be represented by a version like this:

    timestamp: 1234567890.123

A timestamp uniquely identifies a message within a channel. See [Slack API](https://api.slack.com/events/message) for details.

## Reading Messages

Usage in a pipeline:

    resource_types:
        - name: slack-read-resource
          type: docker-image
          source:
            repository: ghcr.io/apptweak/slack-read-resource

    resources:
        - name: slack-in
          type: slack-read-resource
          source: ...

### Source Configuration

The `source` field configures the resource for reading messages from a specific channel. It allows filtering messages by their author and text pattern:

- `token`: *Required*. A Slack API token that allows reading all messages on a selected channel.
- `channel_id`: *Required*. The selected channel ID. The resource only reads messages on this channel.
- `matching`: *Optional*. Only report messages matching this filter. See below for details.
- `not_replied_by`: *Optional*. Ignore messages that have a reply matching this filter. See below for details.

The values of `matching` and `not_replied_by` represent message filters. They are maps with the following elements:

- `author`: *Optional*. User ID that must match the author of the message - either the `user` or the `bot_id` field.
  See [Slack API](https://api.slack.com/events/message) regarding authorship.
- `text_pattern`: *Optional*. Regular expression that must match the message text.
  Wrap in single quotes instead of double, to avoid having to escape `\`.
  See [Slack API](https://api.slack.com/docs/message-formatting) for details on text formatting.


The resource only reports messages that begin new threads and not replies to other messages.

When given a message timestamp as the current version, it only reads messages with that timestamp and later. In any case though, it reads at most 100 of the latest messages. Therefore, the resource must be checked often enough to avoid missing messages.

If `source` has a `not_replied_by` filter, and it matches a message that also matches the `matching` filter, then all messages older than the latest such message are also considered obsolete and are not read.

#### Example

    resources:
      - name: slack-in
        type: slack-read-resource
        source:
          token: "xxxx-xxxxxxxxxx-xxxx"
          channel_id: "C11111111"
          matching:
            text_pattern: '<@U22222222>\s+(.+)'
          not_replied_by:
            author: U22222222

This configures a resource reading messages from channel with ID `C11111111`. It reads only messages that begin by mentioning the user with ID `U22222222`. It ignores messages already replied to by that same user.

### `get`: Read a Message

Reads the message with the requested timestamp and produces the following files:

- `timestamp`: The message timestamp.
- `text`: The message text.
- `text_part1`, `text_part2`, etc.: Parts of text parsed using the `text_pattern` parameter described below.

Parameters:

- `text_pattern`: *Optional*. A regular expression to match against the message text.
  The text matched by each [capturing group](https://www.regular-expressions.info/brackets.html)
  is stored into a file `text_part<num>` where `<num>` is the group index starting with 1.
  Wrap in single quotes instead of double, to avoid having to escape `\`.
  See [Slack API](https://api.slack.com/docs/message-formatting) for details on text formatting.

#### Example

    - get: slack-in
      params:
          text_pattern: '([A-Z]+) ([0-9]+)'

When this configuration sees a message with the text `abc 123` and timestamp `111.222`, it will produce the following files and contents:

- `timestamp`: `111.222`
- `text`: `abc 123`
- `text_part1`: `abc`
- `text_part2`: `123`


## Posting Messages

Usage in a pipeline:

    resource_types:
        - name: slack-post-resource
          type: docker-image
          source:
            repository: ghcr.io/apptweak/slack-post-resource

    resources:
        - name: slack-out
          type: slack-post-resource
          source: ...

### Source Configuration

The `source` field configures the resource for posting on a specific channel:

- `token`: *Required*. A Slack API token that allows posting on a selected channel.
- `channel_id`: *Required*. The selected channel ID. The resource only posts messages on this channel.

#### Example

    resources:
      - name: slack-out
        type: slack-post-resource
        source:
          token: "xxxx-xxxxxxxxxx-xxxx"
          channel_id: "C11111111"

This configures the resource to post on the channel with ID `C11111111`.

### `put`: Post a Message

Posts a message to the selected channel.

Parameters:

- `message`: *Optional*. The message to send described in YAML.
- `message_file`: *Optional*. The file containing the message to send described in JSON.
- `update_ts`: *Optional*. Instead of pusting new message update this message. (support interpolation see below)
- `upload` : *Optional* Upload a file and attached it to the message. (see [files.upload](https://api.slack.com/methods/files.upload))
 - `emoji_reactions` : *Optional* List of emoji names to add as reactions to the posted/updated message (e.g. `["white_check_mark", "rocket"]`).

Either `message` or `message_file` must be present. If both are present, `message_file` takes precedence and `message` is ignored.

The message is described just as the argument to the [`chat.postMessage`](https://api.slack.com/methods/chat.postMessage) method of the Slack API. All fields are supported, except that `token` and `channel` are ignored and instead the resource configuration in `source` is used.

When using `message`, some message parameters support string interpolation to insert contents of arbitrary files or values of environment variables. The following table gives rules for substitution:

| Pattern | Substituted By |
|---------|----------------|
| `{{filename}}` | Contents of file `filename`. You can use globs in the filename (the first match is used as the file to read) |
| `{{$variable}}` | Value of environment variable `variable` |

The following message fields support string interpolation:

- `text`
- `thread_ts`

The following fields of an attachment support string interpolation:

- `fallback`
- `title`
- `title_link`
- `pretext`
- `text`
- `footer`

### Examples

#### Create a thread

Consider a job with the `get: slack-in` step from the example above followed by this step:

    - put: slack-out
      params:
        message:
            thread_ts: "{{slack-in/timestamp}}"
            text: "Hi {{slack-in/text_part1}}! I will do {{slack-in/text_part2}} right away!"

This will reply to the message read by the `get` step (since `thread` is the timestamp of the original message), and the reply will read:

    Hi abc! I will do 123 right away!

#### Send message and upload file

Consider a job with the `get: something` step from the example above followed by this step:

    - put: slack-out
      params:
        message:
            text: "Hi {{slack-in/text_part1}}! I will do {{slack-in/text_part2}} right away!"
        upload:
          file: something/path/to/file
          channels: C.......M
          title: My awesome file
          filetype: php

This will create a message and post *something/path/to/file* to a thread.

> Notice that in order for your message to be visible *channels* is mandatory.

#### Send message and add reactions

Example adding multiple reactions to the message that was just posted:

```yaml
- put: slack-out
  params:
    message:
      text: "Build finished successfully"
    emoji_reactions:
      - "white_check_mark"
      - "rocket"
```

## Releases

This repository publishes container images to GHCR when a version tag is pushed to `master`.

- Tag format: `vX.Y.Z` (e.g., `v1.2.3`)
- Images published:
  - `ghcr.io/apptweak/slack-read-resource:vX.Y.Z` and `:latest`
  - `ghcr.io/apptweak/slack-post-resource:vX.Y.Z` and `:latest`

How it works:
- Pushing a tag `v*.*.*` triggers the `Tag Release` workflow, which verifies the tag commit is on `master` and then invokes the reusable `Build and Push Images` workflow.
- The reusable workflow logs in to GHCR using the built-in `GITHUB_TOKEN`, syncs the `VERSION` file from the tag, and runs `make all` to build and push both images.

Manual release:
- From the GitHub Actions tab, run the `Build and Push Images` workflow and provide a `version` input (e.g., `v1.2.3`).

## Development

### Prerequisites

- Go 1.22+ (for local builds with modules)
- Docker (for building/pushing images)

### Tool versions with mise

Install [mise](https://mise.jdx.dev/install.html):

Usage:

```bash
# Install pinned tool versions
mise install

# Verify tools in use
mise which go
mise which gh
```

### Install dependencies

```bash
go mod tidy
```

### Build images locally

Build and tag both images (`read` and `post`) with the version from `VERSION` and `latest`:

```bash
make all
```

Or build a single image:

```bash
make build-read-resource
make build-post-resource
```

### Docker build context and .dockerignore

- Images are built from the repository root using `-f read/Dockerfile .` or `-f post/Dockerfile .`.
- The build context is the final `.` argument. Only the root `.dockerignore` is honored by Docker; per-directory `.dockerignore` files are ignored when building from the repo root.
- Multi-stage copy lines like:

  ```
  COPY --from=build-env /assets /opt/resource
  ```

  copy artifacts from a prior stage; they do not use the build context. Ensure the producing stage name and output path (`/assets`) match what is built earlier in that stage.

## CI Workflows

- Pull Requests and pushes to `main`/`master`/`feature/*`:
  - `PR Build (No Push)`: builds both images to catch compile issues; no registry push.
- Tags matching `v*.*.*` on `master`:
  - `Tag Release`: verifies the tag commit is on `master`, then calls `Build and Push Images`.
- Manual publish:
  - `Build and Push Images`: can be dispatched from the Actions UI with an input `version` (e.g. `v1.2.3`).
