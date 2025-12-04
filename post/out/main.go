package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/apptweak/concourse-slack-chat-resources/utils"
	"github.com/slack-go/slack"
)

func main() {
	if len(os.Args) < 2 {
		println("usage: " + os.Args[0] + " <source>")
		os.Exit(1)
	}

	source_dir := os.Args[1]

	var request utils.OutRequest

	request_err := json.NewDecoder(os.Stdin).Decode(&request)
	if request_err != nil {
		fatal("Parsing request.", request_err)
	}

	if len(request.Source.Token) == 0 {
		fatal1("Missing source field: token.")
	}

	if len(request.Source.ChannelId) == 0 {
		fatal1("Missing source field: channel_id.")
	}

	if len(request.Params.MessageFile) == 0 && request.Params.Message == nil {
		fatal1("Missing params field: message or message_file.")
	}

	var message *utils.OutMessage

	if len(request.Params.MessageFile) != 0 {
		message = new(utils.OutMessage)
		read_message_file(filepath.Join(source_dir, request.Params.MessageFile), message)
	} else {
		message = request.Params.Message
		interpolate_message(message, source_dir)
	}

	{
		fmt.Fprintf(os.Stderr, "About to send this message:\n")
		m, _ := json.MarshalIndent(message, "", "  ")
		fmt.Fprintf(os.Stderr, "%s\n", m)
	}

	slack_client := slack.New(request.Source.Token)

	var response utils.OutResponse

	// send message
	if len(request.Params.Ts) == 0 {
		response = send(message, &request, slack_client)
	} else {
		request.Params.Ts = get_file_contents(filepath.Join(source_dir, request.Params.Ts))
		response = update(message, &request, slack_client)
	}

	//Attach file
	if request.Params.Upload != nil {
		uploadFile(&response, &request, slack_client, source_dir)
	}

	// Add emoji reactions to the posted/updated message
	if len(request.Params.EmojiReactions) > 0 {
		ts := response.Version["timestamp"]
		fmt.Fprintf(os.Stderr, "Adding emoji reactions to the posted/updated message ts=%s %+v\n", ts, request.Params.EmojiReactions)
		addReactions(slack_client, request.Source.ChannelId, ts, request.Params.EmojiReactions)
	}

	// Add emoji reactions to the thread parent (message.thread_ts) if provided
	if message.ThreadTimestamp != "" && len(request.Params.ThreadEmojiReactions) > 0 {
		fmt.Fprintf(os.Stderr, "Adding emoji reactions to the thread parent: ts=%s %+v\n", message.ThreadTimestamp, request.Params.ThreadEmojiReactions)
		addReactions(slack_client, request.Source.ChannelId, message.ThreadTimestamp, request.Params.ThreadEmojiReactions)
	}

	response_err := json.NewEncoder(os.Stdout).Encode(&response)
	if response_err != nil {
		fatal("encoding response", response_err)
	}
}

func read_message_file(path string, message *utils.OutMessage) {
	file, open_err := os.Open(path)
	if open_err != nil {
		fatal("opening message file", open_err)
	}

	read_err := json.NewDecoder(file).Decode(message)
	if read_err != nil {
		fatal("reading message file", read_err)
	}
}

func interpolate_message(message *utils.OutMessage, source_dir string) {
	message.Text = interpolate(message.Text, source_dir)
	message.ThreadTimestamp = interpolate(message.ThreadTimestamp, source_dir)

	// for i := 0; i < len(message.Attachments); i++ {
	//     attachment := &message.Attachments[i]
	//     attachment.Fallback = interpolate(attachment.Fallback, source_dir)
	//     attachment.Title = interpolate(attachment.Title, source_dir)
	//     attachment.TitleLink = interpolate(attachment.TitleLink, source_dir)
	//     attachment.Pretext = interpolate(attachment.Pretext, source_dir)
	//     attachment.Text = interpolate(attachment.Text, source_dir)
	//     attachment.Footer = interpolate(attachment.Footer, source_dir)
	// }
}

func update(message *utils.OutMessage, request *utils.OutRequest, slack_client *slack.Client) utils.OutResponse {

	fmt.Fprintf(os.Stderr, "About to post an update message: "+request.Params.Ts+"\n")
	_, timestamp, _, err := slack_client.UpdateMessage(request.Source.ChannelId,
		request.Params.Ts,
		slack.MsgOptionText(message.Text, false),
		// slack.MsgOptionAttachments(message.Attachments...),
		// slack.MsgOptionBlocks(message.Blocks.BlockSet...),
		slack.MsgOptionPostMessageParameters(message.PostMessageParameters))

	if err != nil {
		fatal("sending", err)
	}

	var response utils.OutResponse
	response.Version = utils.Version{"timestamp": timestamp}
	return response
}

func get_file_contents(path string) string {
	file, open_err := os.Open(path)
	if open_err != nil {
		fatal("opening file", open_err)
	}

	data, read_err := ioutil.ReadAll(file)
	if read_err != nil {
		fatal("reading file", read_err)
	}

	return string(data)
}

func interpolate(text string, source_dir string) string {

	var out_text string

	start_var := 0
	end_var := 0
	inside_var := false
	c0 := '_'

	for pos, c1 := range text {
		if inside_var {
			if c0 == '}' && c1 == '}' {
				inside_var = false
				end_var = pos + 1

				var value string

				if text[start_var+2] == '$' {
					var_name := text[start_var+3 : end_var-2]
					value = os.Getenv(var_name)
				} else {
					var_name := text[start_var+2 : end_var-2]
					value = get_file_contents(filepath.Join(source_dir, var_name))
				}

				out_text += value
			}
		} else {
			if c0 == '{' && c1 == '{' {
				inside_var = true
				start_var = pos - 1
				out_text += text[end_var:start_var]
			}
		}
		c0 = c1
	}

	out_text += text[end_var:]

	return out_text
}

func send(message *utils.OutMessage, request *utils.OutRequest, slack_client *slack.Client) utils.OutResponse {

	_, timestamp, err := slack_client.PostMessage(request.Source.ChannelId, slack.MsgOptionText(message.Text, false), slack.MsgOptionPostMessageParameters(message.PostMessageParameters))

	if err != nil {
		fatal("sending", err)
	}

	var response utils.OutResponse
	response.Version = utils.Version{"timestamp": timestamp}
	return response
}

func uploadFile(response *utils.OutResponse, request *utils.OutRequest, slack_client *slack.Client, source_dir string) {
	// initialise UploadFileV2Parameters
	params := slack.UploadFileV2Parameters{
		Filename:        request.Params.Upload.FileName,
		Title:           request.Params.Upload.Title,
		ThreadTimestamp: response.Version["timestamp"],
		Channel:         request.Params.Upload.Channels, // Channel is a single string ID, not a list of channels
	}
	// If no specific channel is provided for the upload, use the main channel ID
	if params.Channel == "" {
		params.Channel = request.Source.ChannelId
	}

	if request.Params.Upload.File != "" {
		matched, glob_err := filepath.Glob(filepath.Join(source_dir, request.Params.Upload.File))
		if glob_err != nil {
			fatal("Gloing Pattern", glob_err)
		}
		if len(matched) == 0 {
			fatal1("No file matched the pattern: " + request.Params.Upload.File)
		}

		params.File = matched[0]
		fmt.Fprintf(os.Stderr, "About to upload: "+params.File+"\n")
	} else if request.Params.Upload.Content != "" {
		params.Content = request.Params.Upload.Content
		fmt.Fprintf(os.Stderr, "About to upload specify content as file\n")
	} else {
		fmt.Printf("You must either set Upload.Content or provide a local file path in Upload.File to upload it from your filesystem.")
		return
	}

	p, _ := json.MarshalIndent(params, "", "  ")
	fmt.Fprintf(os.Stderr, "%s\n", p)

	file, err := slack_client.UploadFileV2(params)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	// The V2 API returns a FileSummary, but we need details like URLPrivate which might not be directly populated
	// in the same way. However, the summary contains the ID, and we can use that if needed, or rely on what's returned.
	fmt.Fprintf(os.Stderr, "Uploaded file: ID="+file.ID+", Name="+file.Title+"\n")

	response.Metadata = append(response.Metadata, utils.MetadataField{Name: file.Title, Value: file.ID})
}

func addReactions(slack_client *slack.Client, channelId string, timestamp string, emojis []string) {
	if timestamp == "" || len(emojis) == 0 {
		return
	}
	ref := slack.NewRefToMessage(channelId, timestamp)
	for _, emoji := range emojis {
		if emoji == "" {
			continue
		}

		if err := slack_client.AddReaction(sanitizeEmojiName(emoji), ref); err != nil {
			// Ignore if the reaction is already present
			if strings.Contains(err.Error(), "already_reacted") {
				continue
			}
			fmt.Fprintf(os.Stderr, "Error adding reaction to timestamp "+timestamp+": "+err.Error()+"\n")
		}
	}
}

// sanitizeEmojiName removes a single leading and/or trailing colon while preserving
// internal colons (e.g., :thumbsup:). It also trims surrounding whitespace.
func sanitizeEmojiName(name string) string {
	n := strings.TrimSpace(name)
	if n == "" {
		return ""
	}
	if strings.HasPrefix(n, ":") && len(n) > 1 {
		n = n[1:]
	}
	if strings.HasSuffix(n, ":") && len(n) > 1 {
		n = n[:len(n)-1]
	}
	return n
}

func fatal(doing string, err error) {
	fmt.Fprintf(os.Stderr, "Error "+doing+": "+err.Error()+"\n")
	os.Exit(1)
}

func fatal1(reason string) {
	fmt.Fprintf(os.Stderr, reason+"\n")
	os.Exit(1)
}
