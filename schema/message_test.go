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

package schema

import (
	"context"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mrh997/eino/internal/generic"
)

func TestMessageTemplate(t *testing.T) {
	pyFmtMessage := UserMessage("input: {question}")
	jinja2Message := UserMessage("input: {{question}}")
	goTemplateMessage := UserMessage("input: {{.question}}")
	ctx := context.Background()
	question := "what's the weather today"
	expected := []*Message{UserMessage("input: " + question)}

	ms, err := pyFmtMessage.Format(ctx, map[string]any{"question": question}, FString)
	assert.Nil(t, err)
	assert.True(t, reflect.DeepEqual(expected, ms))
	ms, err = jinja2Message.Format(ctx, map[string]any{"question": question}, Jinja2)
	assert.Nil(t, err)
	assert.True(t, reflect.DeepEqual(expected, ms))
	ms, err = goTemplateMessage.Format(ctx, map[string]any{"question": question}, GoTemplate)
	assert.Nil(t, err)
	assert.True(t, reflect.DeepEqual(expected, ms))

	mp := MessagesPlaceholder("chat_history", false)
	m1 := UserMessage("how are you?")
	m2 := AssistantMessage("I'm good. how about you?", nil)
	ms, err = mp.Format(ctx, map[string]any{"chat_history": []*Message{m1, m2}}, FString)
	assert.Nil(t, err)

	// len(ms) == 2
	assert.Equal(t, 2, len(ms))
	assert.Equal(t, ms[0], m1)
	assert.Equal(t, ms[1], m2)
}

func TestConcatMessage(t *testing.T) {
	t.Run("tool_call_normal_append", func(t *testing.T) {
		expectMsg := &Message{
			Role:    "assistant",
			Content: "",
			ToolCalls: []ToolCall{
				{
					Index: generic.PtrOf(0),
					ID:    "i_am_a_too_call_id",
					Type:  "function",
					Function: FunctionCall{
						Name:      "i_am_a_tool_name",
						Arguments: "{}",
					},
				},
			},
		}
		givenMsgList := []*Message{
			{
				Role:    "",
				Content: "",
				ToolCalls: []ToolCall{
					{
						Index: generic.PtrOf(0),
						ID:    "",
						Type:  "",
						Function: FunctionCall{
							Name: "",
						},
					},
				},
			},
			{
				Role:    "assistant",
				Content: "",
				ToolCalls: []ToolCall{
					{
						Index: generic.PtrOf(0),
						ID:    "i_am_a_too_call_id",
						Type:  "function",
						Function: FunctionCall{
							Name: "i_am_a_tool_name",
						},
					},
				},
			},
			{
				Role:    "",
				Content: "",
				ToolCalls: []ToolCall{
					{
						Index: generic.PtrOf(0),
						ID:    "",
						Type:  "",
						Function: FunctionCall{
							Name:      "",
							Arguments: "{}",
						},
					},
				},
			},
		}

		msg, err := ConcatMessages(givenMsgList)
		assert.NoError(t, err)
		assert.EqualValues(t, expectMsg, msg)
	})

	t.Run("exist_nil_message", func(t *testing.T) {
		givenMsgList := []*Message{
			nil,
			{
				Role:    "assistant",
				Content: "",
				ToolCalls: []ToolCall{
					{
						Index: generic.PtrOf(0),
						ID:    "i_am_a_too_call_id",
						Type:  "function",
						Function: FunctionCall{
							Name: "i_am_a_tool_name",
						},
					},
				},
			},
		}

		_, err := ConcatMessages(givenMsgList)
		assert.ErrorContains(t, err, "unexpected nil chunk in message stream")
	})

	t.Run("response_meta", func(t *testing.T) {
		expectedMsg := &Message{
			Role: "assistant",
			ResponseMeta: &ResponseMeta{
				FinishReason: "stop",
				Usage: &TokenUsage{
					CompletionTokens: 15,
					PromptTokens:     30,
					PromptTokenDetails: PromptTokenDetails{
						CachedTokens: 15,
					},
					TotalTokens: 45,
				},
			},
		}

		givenMsgList := []*Message{
			{
				Role: "assistant",
			},
			{
				Role: "assistant",
				ResponseMeta: &ResponseMeta{
					FinishReason: "",
					Usage: &TokenUsage{
						CompletionTokens: 10,
						PromptTokens:     20,
						PromptTokenDetails: PromptTokenDetails{
							CachedTokens: 10,
						},
						TotalTokens: 30,
					},
				},
			},
			{
				Role: "assistant",
				ResponseMeta: &ResponseMeta{
					FinishReason: "stop",
				},
			},
			{
				Role: "assistant",
				ResponseMeta: &ResponseMeta{
					Usage: &TokenUsage{
						CompletionTokens: 15,
						PromptTokens:     30,
						PromptTokenDetails: PromptTokenDetails{
							CachedTokens: 15,
						},
						TotalTokens: 45,
					},
				},
			},
		}

		msg, err := ConcatMessages(givenMsgList)
		assert.NoError(t, err)
		assert.Equal(t, expectedMsg, msg)

		givenMsgList = append(givenMsgList, &Message{
			Role: "assistant",
			ResponseMeta: &ResponseMeta{
				FinishReason: "tool_calls",
			},
		})
		msg, err = ConcatMessages(givenMsgList)
		assert.NoError(t, err)
		expectedMsg.ResponseMeta.FinishReason = "tool_calls"
		assert.Equal(t, expectedMsg, msg)

	})

	t.Run("err: different roles", func(t *testing.T) {
		msgs := []*Message{
			{Role: User},
			{Role: Assistant},
		}

		msg, err := ConcatMessages(msgs)
		if assert.Error(t, err) {
			assert.ErrorContains(t, err, "cannot concat messages with different roles")
			assert.Nil(t, msg)
		}
	})

	t.Run("err: different name", func(t *testing.T) {
		msgs := []*Message{
			{Role: Assistant, Name: "n", Content: "1"},
			{Role: Assistant, Name: "a", Content: "2"},
		}

		msg, err := ConcatMessages(msgs)
		if assert.Error(t, err) {
			assert.ErrorContains(t, err, "cannot concat messages with different names")
			assert.Nil(t, msg)
		}
	})

	t.Run("err: different tool name", func(t *testing.T) {
		msgs := []*Message{
			{
				Role:       "",
				Content:    "",
				ToolCallID: "123",
				ToolCalls: []ToolCall{
					{
						Index: generic.PtrOf(0),
						ID:    "abc",
						Type:  "",
						Function: FunctionCall{
							Name: "",
						},
					},
				},
			},
			{
				Role:       "assistant",
				Content:    "",
				ToolCallID: "321",
				ToolCalls: []ToolCall{
					{
						Index: generic.PtrOf(0),
						ID:    "abc",
						Type:  "function",
						Function: FunctionCall{
							Name: "i_am_a_tool_name",
						},
					},
				},
			},
		}

		msg, err := ConcatMessages(msgs)
		if assert.Error(t, err) {
			assert.ErrorContains(t, err, "cannot concat messages with different toolCallIDs")
			assert.Nil(t, msg)
		}
	})

	t.Run("first response meta usage is nil", func(t *testing.T) {
		exp := &Message{
			Role: "assistant",
			ResponseMeta: &ResponseMeta{
				FinishReason: "stop",
				Usage: &TokenUsage{
					CompletionTokens: 15,
					PromptTokens:     30,
					TotalTokens:      45,
				},
			},
		}

		msgs := []*Message{
			{
				Role: "assistant",
				ResponseMeta: &ResponseMeta{
					FinishReason: "",
					Usage:        nil,
				},
			},
			{
				Role: "assistant",
				ResponseMeta: &ResponseMeta{
					FinishReason: "stop",
				},
			},
			{
				Role: "assistant",
				ResponseMeta: &ResponseMeta{
					Usage: &TokenUsage{
						CompletionTokens: 15,
						PromptTokens:     30,
						TotalTokens:      45,
					},
				},
			},
		}

		msg, err := ConcatMessages(msgs)
		assert.NoError(t, err)
		assert.Equal(t, exp, msg)
	})

	t.Run("concurrent concat", func(t *testing.T) {
		content := "i_am_a_good_concat_message"
		exp := &Message{Role: Assistant, Content: content}
		var msgs []*Message
		for i := 0; i < len(content); i++ {
			msgs = append(msgs, &Message{Role: Assistant, Content: content[i : i+1]})
		}

		wg := sync.WaitGroup{}
		size := 100
		wg.Add(size)
		for i := 0; i < size; i++ {
			go func() {
				defer wg.Done()
				msg, err := ConcatMessages(msgs)
				assert.NoError(t, err)
				assert.Equal(t, exp, msg)
			}()
		}

		wg.Wait()
	})

	t.Run("concat logprobs", func(t *testing.T) {
		msgs := []*Message{
			{
				Role:    Assistant,
				Content: "🚀",
				ResponseMeta: &ResponseMeta{
					LogProbs: &LogProbs{
						Content: []LogProb{
							{
								Token:   "\\xf0\\x9f\\x9a",
								LogProb: -0.0000073458323,
								Bytes:   []int64{240, 159, 154},
							},
							{
								Token:   "\\x80",
								LogProb: 0,
								Bytes:   []int64{128},
							},
						},
					},
				},
			},
			{
				Role:    "",
				Content: "❤️",
				ResponseMeta: &ResponseMeta{
					LogProbs: &LogProbs{
						Content: []LogProb{
							{
								Token:   "❤️",
								LogProb: -0.0011431955,
								Bytes:   []int64{226, 157, 164, 239, 184, 143},
							},
						},
					},
				},
			},
			{
				Role: "",
				ResponseMeta: &ResponseMeta{
					FinishReason: "stop",
					Usage: &TokenUsage{
						PromptTokens:     7,
						CompletionTokens: 3,
						TotalTokens:      10,
					},
				},
			},
		}

		msg, err := ConcatMessages(msgs)
		assert.NoError(t, err)
		assert.Equal(t, 3, len(msg.ResponseMeta.LogProbs.Content))
		assert.Equal(t, msgs[0].ResponseMeta.LogProbs.Content[0], msg.ResponseMeta.LogProbs.Content[0])
		assert.Equal(t, msgs[0].ResponseMeta.LogProbs.Content[1], msg.ResponseMeta.LogProbs.Content[1])
		assert.Equal(t, msgs[1].ResponseMeta.LogProbs.Content[0], msg.ResponseMeta.LogProbs.Content[2])
	})

	t.Run("fix unexpected setting ResponseMeta of the first element in slice after ConcatMessages", func(t *testing.T) {
		msgs := []*Message{
			{
				Role:    Assistant,
				Content: "🚀",
				//ResponseMeta: &ResponseMeta{},
			},
			{
				Role:         "",
				Content:      "❤️",
				ResponseMeta: &ResponseMeta{},
			},
			{
				Role: "",
				ResponseMeta: &ResponseMeta{
					FinishReason: "stop",
					Usage: &TokenUsage{
						PromptTokens:     7,
						CompletionTokens: 3,
						TotalTokens:      10,
					},
				},
			},
		}

		msg, err := ConcatMessages(msgs)
		assert.NoError(t, err)
		assert.Equal(t, msgs[2].ResponseMeta, msg.ResponseMeta)
		assert.Nil(t, msgs[0].ResponseMeta)
	})
}

func TestConcatToolCalls(t *testing.T) {
	t.Run("atomic_field_in_first_chunk", func(t *testing.T) {
		givenToolCalls := []ToolCall{
			{
				Index: generic.PtrOf(0),
				ID:    "tool_call_id",
				Type:  "function",
				Function: FunctionCall{
					Name: "tool_name",
				},
			},
			{
				Index: generic.PtrOf(0),
				Function: FunctionCall{
					Arguments: "call me please",
				},
			},
		}

		expectedToolCall := ToolCall{
			Index: generic.PtrOf(0),
			ID:    "tool_call_id",
			Type:  "function",
			Function: FunctionCall{
				Name:      "tool_name",
				Arguments: "call me please",
			},
		}

		tc, err := concatToolCalls(givenToolCalls)
		assert.NoError(t, err)
		assert.Len(t, tc, 1)
		assert.EqualValues(t, expectedToolCall, tc[0])
	})

	t.Run("atomic_field_in_every_chunk", func(t *testing.T) {
		givenToolCalls := []ToolCall{
			{
				Index: generic.PtrOf(0),
				ID:    "tool_call_id",
				Type:  "function",
				Function: FunctionCall{
					Name: "tool_name",
				},
			},
			{
				Index: generic.PtrOf(0),
				ID:    "tool_call_id",
				Type:  "function",
				Function: FunctionCall{
					Name:      "tool_name",
					Arguments: "call me please",
				},
			},
		}

		expectedToolCall := ToolCall{
			Index: generic.PtrOf(0),
			ID:    "tool_call_id",
			Type:  "function",
			Function: FunctionCall{
				Name:      "tool_name",
				Arguments: "call me please",
			},
		}

		tc, err := concatToolCalls(givenToolCalls)
		assert.NoError(t, err)
		assert.Len(t, tc, 1)
		assert.EqualValues(t, expectedToolCall, tc[0])
	})

	t.Run("atomic_field_in_interval", func(t *testing.T) {
		givenToolCalls := []ToolCall{
			{
				Index: generic.PtrOf(0),
				ID:    "tool_call_id",
				Type:  "",
				Function: FunctionCall{
					Name: "",
				},
			},
			{
				Index: generic.PtrOf(0),
				ID:    "",
				Type:  "function",
				Function: FunctionCall{
					Name:      "",
					Arguments: "call me please",
				},
			},
			{
				Index: generic.PtrOf(0),
				ID:    "tool_call_id",
				Type:  "",
				Function: FunctionCall{
					Name:      "",
					Arguments: "",
				},
			},
		}

		expectedToolCall := ToolCall{
			Index: generic.PtrOf(0),
			ID:    "tool_call_id",
			Type:  "function",
			Function: FunctionCall{
				Name:      "",
				Arguments: "call me please",
			},
		}

		tc, err := concatToolCalls(givenToolCalls)
		assert.NoError(t, err)
		assert.Len(t, tc, 1)
		assert.EqualValues(t, expectedToolCall, tc[0])
	})

	t.Run("different_tool_id", func(t *testing.T) {
		givenToolCalls := []ToolCall{
			{
				Index: generic.PtrOf(0),
				ID:    "tool_call_id",
				Type:  "function",
				Function: FunctionCall{
					Name: "tool_name",
				},
			},
			{
				Index: generic.PtrOf(0),
				ID:    "tool_call_id_1",
				Type:  "function",
				Function: FunctionCall{
					Name:      "tool_name",
					Arguments: "call me please",
				},
			},
		}

		_, err := concatToolCalls(givenToolCalls)
		assert.ErrorContains(t, err, "cannot concat ToolCalls with different tool id")
	})

	t.Run("different_tool_type", func(t *testing.T) {
		givenToolCalls := []ToolCall{
			{
				Index: generic.PtrOf(0),
				ID:    "tool_call_id",
				Type:  "function",
				Function: FunctionCall{
					Name: "tool_name",
				},
			},
			{
				Index: generic.PtrOf(0),
				ID:    "tool_call_id",
				Type:  "function_1",
				Function: FunctionCall{
					Name:      "tool_name",
					Arguments: "call me please",
				},
			},
		}

		_, err := concatToolCalls(givenToolCalls)
		assert.ErrorContains(t, err, "cannot concat ToolCalls with different tool type")
	})

	t.Run("different_tool_name", func(t *testing.T) {
		givenToolCalls := []ToolCall{
			{
				Index: generic.PtrOf(0),
				ID:    "tool_call_id",
				Type:  "function",
				Function: FunctionCall{
					Name: "tool_name",
				},
			},
			{
				Index: generic.PtrOf(0),
				ID:    "tool_call_id",
				Type:  "function",
				Function: FunctionCall{
					Name:      "tool_name_1",
					Arguments: "call me please",
				},
			},
		}

		_, err := concatToolCalls(givenToolCalls)
		assert.ErrorContains(t, err, "cannot concat ToolCalls with different tool name")
	})

	t.Run("multi_tool_call", func(t *testing.T) {
		givenToolCalls := []ToolCall{
			{
				Index: generic.PtrOf(0),
				ID:    "tool_call_id",
				Type:  "",
				Function: FunctionCall{
					Name: "",
				},
			},
			{
				Index: generic.PtrOf(0),
				ID:    "",
				Type:  "function",
				Function: FunctionCall{
					Name:      "",
					Arguments: "call me please",
				},
			},
			{
				Index: generic.PtrOf(0),
				ID:    "tool_call_id",
				Type:  "",
				Function: FunctionCall{
					Name:      "",
					Arguments: "",
				},
			},
			{
				Index: generic.PtrOf(1),
				ID:    "tool_call_id",
				Type:  "",
				Function: FunctionCall{
					Name: "",
				},
			},
			{
				Index: generic.PtrOf(1),
				ID:    "",
				Type:  "function",
				Function: FunctionCall{
					Name:      "",
					Arguments: "call me please",
				},
			},
			{
				Index: generic.PtrOf(1),
				ID:    "tool_call_id",
				Type:  "",
				Function: FunctionCall{
					Name:      "",
					Arguments: "",
				},
			},
			{
				Index: nil,
				ID:    "22",
				Type:  "",
				Function: FunctionCall{
					Name: "",
				},
			},
			{
				Index: nil,
				ID:    "44",
				Type:  "",
				Function: FunctionCall{
					Name: "",
				},
			},
		}

		expectedToolCall := []ToolCall{
			{
				Index: nil,
				ID:    "22",
				Type:  "",
				Function: FunctionCall{
					Name: "",
				},
			},
			{
				Index: nil,
				ID:    "44",
				Type:  "",
				Function: FunctionCall{
					Name: "",
				},
			},
			{
				Index: generic.PtrOf(0),
				ID:    "tool_call_id",
				Type:  "function",
				Function: FunctionCall{
					Name:      "",
					Arguments: "call me please",
				},
			},
			{
				Index: generic.PtrOf(1),
				ID:    "tool_call_id",
				Type:  "function",
				Function: FunctionCall{
					Name:      "",
					Arguments: "call me please",
				},
			},
		}

		tc, err := concatToolCalls(givenToolCalls)
		assert.NoError(t, err)
		assert.EqualValues(t, expectedToolCall, tc)
	})
}
