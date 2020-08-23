/*
 * Archive Bot
 * Copyright (c) 2020 Nichole Mattera
 * 
 * This program is free software; you can redistribute it and/or
 * modify it under the terms of the GNU General Public License
 * as published by the Free Software Foundation; either version 2
 * of the License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program; if not, write to the Free Software
 * Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.
 */

package main

import(
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/bwmarrin/discordgo"
)

type Attachment struct {
	Filename string `json:"filename"`
	OriginalFilename string `json:"original_filename"`
}

type Message struct {
	Attachments []Attachment `json:"attachments"`
	AuthorID string `json:"author_id"`
	Author string `json:"author"`
	Content string `json:"content"`
	Reactions []Reaction `json:"reactions"`
	Timestamp string `json:"timestamp"`
}

type Reaction struct {
	Name string `json:"name"`
	Count int `json:"count"`
}

func main() {
	var token string
	var channelID string
	var lastMessageID string

	flag.StringVar(&token, "t", "", "The bot token. (Required)")
	flag.StringVar(&channelID, "c", "", "The channel ID. (Required)")
	flag.StringVar(&lastMessageID, "m", "", "The message ID to start archiving from, not including said message. If not specified it will archive all messages in the channel.")

	flag.Parse()

	if token == "" || channelID == "" {
		fmt.Println("Usage:")
		flag.PrintDefaults()
		return
	}

	discord, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		return
	}

	err = discord.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	if _, err := os.Stat("./pfps"); os.IsNotExist(err) {
		os.Mkdir("./pfps", 0700)
	}

	if _, err := os.Stat("./" + channelID); os.IsNotExist(err) {
		os.Mkdir("./" + channelID, 0700)
	}

	archivedMessages := []Message{}
	numberOfMessages := 100
	for numberOfMessages == 100 {
		messages, err := discord.ChannelMessages(channelID, 100, lastMessageID, "", "")
		if err != nil {
			fmt.Println("error getting messages,", err)
			return
		}

		for index, message := range messages {
			// Download PfPs
			if _, err := os.Stat("./pfps/" + message.Author.ID); os.IsNotExist(err) {
				err = DownloadFile("./pfps/" + message.Author.ID, message.Author.AvatarURL(""))
				
				if err != nil {
					fmt.Println("error downloading PfP,", err)
				}
			}

			// Build Attachments
			archivedAttachments := []Attachment{}
			for _, attachment := range message.Attachments {
				if _, err := os.Stat("./" + channelID + "/" + message.ID + "_" + attachment.Filename); os.IsNotExist(err) {
					err = DownloadFile("./" + channelID + "/" + message.ID + "_" + attachment.Filename, attachment.URL)
					
					if err != nil {
						fmt.Println("error downloading attachment,", err)
					}
				}

				archivedAttachment := Attachment {
					message.ID + "_" + attachment.Filename,
					attachment.Filename,
				}
			
				archivedAttachments = append([]Attachment{ archivedAttachment }, archivedAttachments...)
			}

			// Build Reactions
			archivedReactions := []Reaction{}
			for _, reaction := range message.Reactions {
				archivedReaction := Reaction {
					reaction.Emoji.Name,
					reaction.Count,
				}
			
				archivedReactions = append([]Reaction{ archivedReaction }, archivedReactions...)
			}

			archivedMessage := Message {
				archivedAttachments,
				message.Author.ID,
				message.Author.Username + "#" + message.Author.Discriminator,
				message.Content,
				archivedReactions,
				string(message.Timestamp),
			}
			
			archivedMessages = append([]Message{ archivedMessage }, archivedMessages...)

			if index == len(messages) - 1 {
				lastMessageID = message.ID
			}
		}

		numberOfMessages = len(messages)
	}

	res, err := json.Marshal(archivedMessages)
	if err != nil {
		fmt.Println("error encoding messages to json,", err)
		return
	}

	err = ioutil.WriteFile("./" + channelID + ".json", res, 0644)

	discord.Close()
}

func DownloadFile(filepath string, url string) error {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}
