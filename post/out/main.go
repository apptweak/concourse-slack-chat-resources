package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
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
	var fileContent []byte
	var filename string
	var fileSize int64

	// Read file content and determine filename
	if request.Params.Upload.File != "" {
		matched, glob_err := filepath.Glob(filepath.Join(source_dir, request.Params.Upload.File))
		if glob_err != nil {
			fatal("Globbing Pattern", glob_err)
		}
		if len(matched) == 0 {
			fatal1("No files matched the pattern: " + request.Params.Upload.File)
		}

		filePath := matched[0]
		fmt.Fprintf(os.Stderr, "About to upload: %s\n", filePath)

		// Read file content
		content, read_err := ioutil.ReadFile(filePath)
		if read_err != nil {
			fatal("reading file", read_err)
		}
		fileContent = content

		// Get file size
		fileInfo, stat_err := os.Stat(filePath)
		if stat_err != nil {
			fatal("getting file stats", stat_err)
		}
		fileSize = fileInfo.Size()

		// Determine filename
		if request.Params.Upload.FileName != "" {
			filename = request.Params.Upload.FileName
		} else {
			filename = filepath.Base(filePath)
		}
	} else if request.Params.Upload.Content != "" {
		fmt.Fprintf(os.Stderr, "About to upload specified content as file\n")
		fileContent = []byte(request.Params.Upload.Content)
		fileSize = int64(len(fileContent))

		if request.Params.Upload.FileName != "" {
			filename = request.Params.Upload.FileName
		} else {
			filename = "upload.txt"
		}
	} else {
		fatal1("You must either set Upload.Content or provide a local file path in Upload.File to upload it from your filesystem.")
		return
	}

	// Determine title
	title := request.Params.Upload.Title
	if title == "" {
		title = filename
	}

	// Determine channel_id - use first channel from Channels or fallback to Source.ChannelId
	// For file uploads, we'll use the source channel ID to ensure consistency with the message
	channelId := request.Source.ChannelId
	if request.Params.Upload.Channels != "" {
		channels := strings.Split(request.Params.Upload.Channels, ",")
		if len(channels) > 0 && strings.TrimSpace(channels[0]) != "" {
			channelId = strings.TrimSpace(channels[0])
		}
	}

	// Step 1: Call files.getUploadURLExternal
	fmt.Fprintf(os.Stderr, "Step 1: Getting upload URL for file: %s (size: %d bytes)\n", filename, fileSize)
	uploadURL, fileID, err := getUploadURLExternal(request.Source.Token, filename, fileSize)
	if err != nil {
		fatal("getting upload URL", err)
	}
	fmt.Fprintf(os.Stderr, "Got upload URL and file ID: %s\n", fileID)

	// Step 2: Upload file content to the upload URL using HTTP PUT
	fmt.Fprintf(os.Stderr, "Step 2: Uploading file content to upload URL\n")
	err = uploadFileToURL(uploadURL, fileContent)
	if err != nil {
		fatal("uploading file to URL", err)
	}
	fmt.Fprintf(os.Stderr, "File content uploaded successfully\n")

	// Step 3: Call files.completeUploadExternal - complete upload first
	fmt.Fprintf(os.Stderr, "Step 3: Completing upload for file ID: %s\n", fileID)
	fileInfo, err := completeUploadExternal(request.Source.Token, fileID, title, "", "", "")
	if err != nil {
		fatal("completing upload", err)
	}

	// Step 4: Share the file to the channel using files.sharedPublicURL.create or by posting it
	// Since files.completeUploadExternal with channel_id doesn't seem to work,
	// let's try sharing via a message update or a separate share call
	fmt.Fprintf(os.Stderr, "Step 4: Attempting to share file to channel: %s\n", channelId)
	threadTs := response.Version["timestamp"]

	// Try using chat.postMessage with file reference or update the original message
	// Actually, let's check if we can use the file permalink in the message
	err = shareFileByUpdatingMessage(slack_client, request.Source.ChannelId, threadTs, fileInfo.Permalink)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not share file via message update: %s\n", err.Error())
		// File is still uploaded, just not shared - this is better than failing completely
	}

	fmt.Fprintf(os.Stderr, "Upload completed successfully. Name: %s, URL: %s\n", fileInfo.Name, fileInfo.URLPrivate)
	response.Metadata = append(response.Metadata, utils.MetadataField{Name: fileInfo.Name, Value: fileInfo.URLPrivate})
}

// getUploadURLExternal calls files.getUploadURLExternal API
func getUploadURLExternal(token, filename string, length int64) (string, string, error) {
	apiURL := "https://slack.com/api/files.getUploadURLExternal"

	formData := url.Values{}
	formData.Set("filename", filename)
	formData.Set("length", fmt.Sprintf("%d", length))
	formData.Set("token", token)

	resp, err := http.PostForm(apiURL, formData)
	if err != nil {
		return "", "", fmt.Errorf("failed to call API: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		OK        bool   `json:"ok"`
		Error     string `json:"error,omitempty"`
		UploadURL string `json:"upload_url,omitempty"`
		FileID    string `json:"file_id,omitempty"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.OK {
		return "", "", fmt.Errorf("API error: %s", result.Error)
	}

	if result.UploadURL == "" || result.FileID == "" {
		return "", "", fmt.Errorf("missing upload_url or file_id in response")
	}

	return result.UploadURL, result.FileID, nil
}

// uploadFileToURL uploads file content to the provided URL using HTTP PUT
func uploadFileToURL(uploadURL string, content []byte) error {
	req, err := http.NewRequest("PUT", uploadURL, bytes.NewReader(content))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = int64(len(content))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// FileInfo represents the file information returned from completeUploadExternal
type FileInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	URLPrivate string `json:"url_private"`
	Permalink  string `json:"permalink"`
}

// shareFileByUpdatingMessage updates the original message to include the file link
func shareFileByUpdatingMessage(slack_client *slack.Client, channelID, messageTs, filePermalink string) error {
	// Update the original message to include the file
	// This is a workaround since files.completeUploadExternal with channel_id doesn't share the file
	_, _, _, err := slack_client.UpdateMessage(channelID, messageTs,
		slack.MsgOptionText(fmt.Sprintf("Testing file upload\n\nFile: %s", filePermalink), false))
	return err
}

// shareFileViaMessage shares a file by posting a message with the file in a channel
func shareFileViaMessage(token, channelID, fileID, threadTs string) error {
	// Use chat.postMessage API directly with file parameter
	apiURL := "https://slack.com/api/chat.postMessage"

	formData := url.Values{}
	formData.Set("channel", channelID)
	formData.Set("token", token)

	// Add file as an attachment - try using blocks or file parameter
	// Actually, let's try posting the file permalink or using files.info to get shareable URL
	// For now, let's use a simpler approach: post message with file reference

	// Try using the file ID in the message
	// Slack might auto-link file IDs in messages
	messageText := fmt.Sprintf("File uploaded: <https://apptweak.slack.com/files/%s|View File>", fileID)
	formData.Set("text", messageText)

	if threadTs != "" {
		formData.Set("thread_ts", threadTs)
	}

	resp, err := http.PostForm(apiURL, formData)
	if err != nil {
		return fmt.Errorf("failed to call API: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Fprintf(os.Stderr, "chat.postMessage (file share) API response: %s\n", string(body))

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("API error: %s", result.Error)
	}

	return nil
}

// completeUploadExternal calls files.completeUploadExternal API
func completeUploadExternal(token, fileID, title, channelID, threadTs, initialComment string) (*FileInfo, error) {
	apiURL := "https://slack.com/api/files.completeUploadExternal"

	// Build files array
	filesArray := []map[string]string{
		{
			"id":    fileID,
			"title": title,
		},
	}

	formData := url.Values{}
	filesJSON, err := json.Marshal(filesArray)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal files array: %w", err)
	}
	formData.Set("files", string(filesJSON))
	formData.Set("token", token)

	// Add channel_id to share the file to the channel
	if channelID != "" {
		formData.Set("channel_id", channelID)
	}

	// Add initial_comment - this might be required to share the file
	if initialComment != "" {
		formData.Set("initial_comment", initialComment)
	}

	// Add thread_ts if provided to attach file to a thread
	if threadTs != "" {
		formData.Set("thread_ts", threadTs)
	}

	// Debug: log the request parameters (without token)
	fmt.Fprintf(os.Stderr, "completeUploadExternal request - channel_id: %s, thread_ts: %s, files: %s\n", channelID, threadTs, string(filesJSON))

	resp, err := http.PostForm(apiURL, formData)
	if err != nil {
		return nil, fmt.Errorf("failed to call API: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Debug: log the raw response
	fmt.Fprintf(os.Stderr, "completeUploadExternal API response: %s\n", string(body))

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
		Files []struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			URLPrivate string `json:"url_private"`
			Permalink  string `json:"permalink"`
		} `json:"files,omitempty"`
		File struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			URLPrivate string `json:"url_private"`
			Permalink  string `json:"permalink"`
		} `json:"file,omitempty"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("API error: %s", result.Error)
	}

	// Handle both files array and single file response formats
	var finalFileID, finalFileName, finalFileURL, finalPermalink string

	if len(result.Files) > 0 {
		finalFileID = result.Files[0].ID
		finalFileName = result.Files[0].Name
		finalFileURL = result.Files[0].URLPrivate
		finalPermalink = result.Files[0].Permalink
	} else if result.File.ID != "" {
		finalFileID = result.File.ID
		finalFileName = result.File.Name
		finalFileURL = result.File.URLPrivate
		finalPermalink = result.File.Permalink
	} else {
		return nil, fmt.Errorf("no file information in response")
	}

	return &FileInfo{
		ID:         finalFileID,
		Name:       finalFileName,
		URLPrivate: finalFileURL,
		Permalink:  finalPermalink,
	}, nil
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
