/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package prompt

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mrh997/eino/schema"
)

func TestFormat(t *testing.T) {
	pyFmtTestTemplate := []schema.MessagesTemplate{
		schema.SystemMessage(
			"you are a helpful assistant.\n" +
				"here is the context: {context}"),
		schema.MessagesPlaceholder("chat_history", true),
		schema.UserMessage("question: {question}"),
	}
	jinja2TestTemplate := []schema.MessagesTemplate{
		schema.SystemMessage(
			"you are a helpful assistant.\n" +
				"here is the context: {{context}}"),
		schema.MessagesPlaceholder("chat_history", true),
		schema.UserMessage("question: {{question}}"),
	}
	goFmtTestTemplate := []schema.MessagesTemplate{
		schema.SystemMessage(
			"you are a helpful assistant.\n" +
				"here is the context: {{.context}}"),
		schema.MessagesPlaceholder("chat_history", true),
		schema.UserMessage("question: {{.question}}"),
	}
	testValues := map[string]any{
		"context":  "it's beautiful day",
		"question": "how is the day today",
		"chat_history": []*schema.Message{
			schema.UserMessage("who are you"),
			schema.AssistantMessage("I'm a helpful assistant", nil),
		},
	}
	expected := []*schema.Message{
		schema.SystemMessage(
			"you are a helpful assistant.\n" +
				"here is the context: it's beautiful day"),
		schema.UserMessage("who are you"),
		schema.AssistantMessage("I'm a helpful assistant", nil),
		schema.UserMessage("question: how is the day today"),
	}

	// FString
	chatTemplate := FromMessages(schema.FString, pyFmtTestTemplate...)
	msgs, err := chatTemplate.Format(context.Background(), testValues)
	assert.Nil(t, err)
	assert.Equal(t, expected, msgs)

	// Jinja2
	chatTemplate = FromMessages(schema.Jinja2, jinja2TestTemplate...)
	msgs, err = chatTemplate.Format(context.Background(), testValues)
	assert.Nil(t, err)
	assert.Equal(t, expected, msgs)

	// GoTemplate
	chatTemplate = FromMessages(schema.GoTemplate, goFmtTestTemplate...)
	msgs, err = chatTemplate.Format(context.Background(), testValues)
	assert.Nil(t, err)
	assert.Equal(t, expected, msgs)
}

func TestDocumentFormat(t *testing.T) {
	docs := []*schema.Document{
		{
			ID:      "1",
			Content: "qwe",
			MetaData: map[string]any{
				"hello": 888,
			},
		},
		{
			ID:      "2",
			Content: "asd",
			MetaData: map[string]any{
				"bye": 111,
			},
		},
	}

	template := FromMessages(schema.FString,
		schema.SystemMessage("all:{all_docs}\nsingle:{single_doc}"),
	)

	msgs, err := template.Format(context.Background(), map[string]any{
		"all_docs":   docs,
		"single_doc": docs[0],
	})

	assert.Nil(t, err)
	t.Log(msgs)
}

func TestMultiContentFormat(t *testing.T) {
	mtpl := []schema.MessagesTemplate{
		&schema.Message{
			Content: "{a}",
			MultiContent: []schema.ChatMessagePart{
				{
					Text: "{b}",
					ImageURL: &schema.ChatMessageImageURL{
						URL: "{c}",
						URI: "{d}",
					},
					AudioURL: &schema.ChatMessageAudioURL{
						URL: "{e}",
						URI: "{f}",
					},
					VideoURL: &schema.ChatMessageVideoURL{
						URL: "{g}",
						URI: "{h}",
					},
					FileURL: &schema.ChatMessageFileURL{
						URL: "{i}",
						URI: "{j}",
					},
				},
			},
		},
	}
	input := map[string]any{
		"a": "content",
		"b": "text",
		"c": "image url",
		"d": "image uri",
		"e": "audio url",
		"f": "audio uri",
		"g": "video url",
		"h": "video uri",
		"i": "file url",
		"j": "json url",
	}
	expected := []*schema.Message{
		{
			Content: "content",
			MultiContent: []schema.ChatMessagePart{
				{
					Text: "text",
					ImageURL: &schema.ChatMessageImageURL{
						URL: "{c}",
						URI: "{d}",
					},
					AudioURL: &schema.ChatMessageAudioURL{
						URL: "{e}",
						URI: "{f}",
					},
					VideoURL: &schema.ChatMessageVideoURL{
						URL: "{g}",
						URI: "{h}",
					},
					FileURL: &schema.ChatMessageFileURL{
						URL: "{i}",
						URI: "{j}",
					},
				},
			},
		},
	}

	tpl := FromMessages(schema.FString, mtpl...)
	result, err := tpl.Format(context.Background(), input)
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
}
