/*
 * Copyright 2025 CloudWeGo Authors
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

package react

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/mrh997/eino/components/model"
	"github.com/mrh997/eino/components/tool"
	"github.com/mrh997/eino/compose"
	mockModel "github.com/mrh997/eino/internal/mock/components/model"
	"github.com/mrh997/eino/schema"
)

func TestWithMessageFuture(t *testing.T) {
	ctx := context.Background()

	// Test with tool calls
	t.Run("test generate with tool calls", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		cm := mockModel.NewMockToolCallingChatModel(ctrl)
		fakeTool := &fakeToolGreetForTest{}

		info, err := fakeTool.Info(ctx)
		assert.NoError(t, err)

		// Mock model response with tool call
		cm.EXPECT().Generate(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(schema.AssistantMessage("",
				[]schema.ToolCall{
					{
						ID: "tool-call-1",
						Function: schema.FunctionCall{
							Name:      info.Name,
							Arguments: `{"name": "test user"}`,
						},
					},
				}), nil).
			Times(1)
		cm.EXPECT().Generate(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(schema.AssistantMessage("final response", nil), nil).
			Times(1)
		cm.EXPECT().WithTools(gomock.Any()).Return(cm, nil).AnyTimes()

		// Create agent with MessageFuture
		option, future := WithMessageFuture()
		a, err := NewAgent(ctx, &AgentConfig{
			ToolCallingModel: cm,
			ToolsConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{fakeTool},
			},
			MaxStep: 3,
		})
		assert.Nil(t, err)

		// Generate response
		response, err := a.Generate(ctx, []*schema.Message{
			schema.UserMessage("use the greet tool"),
		}, option)
		assert.Nil(t, err)
		assert.Equal(t, "final response", response.Content)

		sIter := future.GetMessageStreams()
		// Should be no messages
		_, hasNext, err := sIter.Next()
		assert.Nil(t, err)
		assert.False(t, hasNext)

		iter := future.GetMessages()
		// First message should be the assistant message for tool calling
		msg1, hasNext, err := iter.Next()
		assert.Nil(t, err)
		assert.True(t, hasNext)
		assert.Equal(t, schema.Assistant, msg1.Role)
		assert.Equal(t, 1, len(msg1.ToolCalls))

		// Second message should be the tool response
		msg2, hasNext, err := iter.Next()
		assert.Nil(t, err)
		assert.True(t, hasNext)
		assert.Equal(t, schema.Tool, msg2.Role)

		// Third message should be the final response
		msg3, hasNext, err := iter.Next()
		assert.Nil(t, err)
		assert.True(t, hasNext)
		assert.Equal(t, "final response", msg3.Content)

		// Should be no more messages
		_, hasNext, err = iter.Next()
		assert.Nil(t, err)
		assert.False(t, hasNext)
	})
	// Test with streaming tool calls
	t.Run("test generate with streaming tool calls", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		cm := mockModel.NewMockToolCallingChatModel(ctrl)
		fakeTool := &fakeStreamToolGreetForTest{}

		info, err := fakeTool.Info(ctx)
		assert.NoError(t, err)

		// Mock model response with tool call
		cm.EXPECT().Generate(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(schema.AssistantMessage("",
				[]schema.ToolCall{
					{
						ID: "tool-call-1",
						Function: schema.FunctionCall{
							Name:      info.Name,
							Arguments: `{"name": "test user"}`,
						},
					},
				}), nil).
			Times(1)
		cm.EXPECT().Generate(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(schema.AssistantMessage("final response", nil), nil).
			Times(1)
		cm.EXPECT().WithTools(gomock.Any()).Return(cm, nil).AnyTimes()

		// Create agent with MessageFuture
		option, future := WithMessageFuture()
		a, err := NewAgent(ctx, &AgentConfig{
			ToolCallingModel: cm,
			ToolsConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{fakeTool},
			},
			MaxStep: 3,
		})
		assert.Nil(t, err)

		// Generate response
		response, err := a.Generate(ctx, []*schema.Message{
			schema.UserMessage("use the greet tool"),
		}, option)
		assert.Nil(t, err)
		assert.Equal(t, "final response", response.Content)

		// Get messages from future
		iter := future.GetMessages()

		// First message should be the assistant message for tool calling
		msg1, hasNext, err := iter.Next()
		assert.Nil(t, err)
		assert.True(t, hasNext)
		assert.Equal(t, schema.Assistant, msg1.Role)
		assert.Equal(t, 1, len(msg1.ToolCalls))

		// Second message should be the tool response
		msg2, hasNext, err := iter.Next()
		assert.Nil(t, err)
		assert.True(t, hasNext)
		assert.Equal(t, schema.Tool, msg2.Role)

		// Third message should be the final response
		msg3, hasNext, err := iter.Next()
		assert.Nil(t, err)
		assert.True(t, hasNext)
		assert.Equal(t, "final response", msg3.Content)

		// Should be no more messages
		_, hasNext, err = iter.Next()
		assert.Nil(t, err)
		assert.False(t, hasNext)
	})

	// Test with non-streaming tool but using agent's Stream interface
	t.Run("test stream with tool calls", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		cm := mockModel.NewMockToolCallingChatModel(ctrl)
		fakeTool := &fakeToolGreetForTest{}

		info, err := fakeTool.Info(ctx)
		assert.NoError(t, err)

		// Mock model response with tool call
		cm.EXPECT().Stream(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(schema.StreamReaderFromArray([]*schema.Message{schema.AssistantMessage("",
				[]schema.ToolCall{
					{
						ID: "tool-call-1",
						Function: schema.FunctionCall{
							Name:      info.Name,
							Arguments: `{"name": "test user"}`,
						},
					},
				})}), nil).
			Times(1)
		cm.EXPECT().Stream(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(schema.StreamReaderFromArray([]*schema.Message{schema.AssistantMessage("final response", nil)}), nil).
			Times(1)
		cm.EXPECT().WithTools(gomock.Any()).Return(cm, nil).AnyTimes()

		// Create agent with MessageFuture
		option, future := WithMessageFuture()
		a, err := NewAgent(ctx, &AgentConfig{
			ToolCallingModel: cm,
			ToolsConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{fakeTool},
			},
			MaxStep: 3,
		})
		assert.Nil(t, err)

		// Use Stream interface
		stream, err := a.Stream(ctx, []*schema.Message{
			schema.UserMessage("use the greet tool"),
		}, option)
		assert.Nil(t, err)

		// Collect all chunks from stream
		finalResponse, err := schema.ConcatMessageStream(stream)
		assert.Nil(t, err)
		assert.Equal(t, "final response", finalResponse.Content)

		iter := future.GetMessages()
		// Should be no messages
		_, hasNext, err := iter.Next()
		assert.Nil(t, err)
		assert.False(t, hasNext)

		// Get message streams from future
		sIter := future.GetMessageStreams()

		// First message should be the assistant message for tool calling
		stream1, hasNext, err := sIter.Next()
		assert.Nil(t, err)
		assert.True(t, hasNext)
		assert.NotNil(t, stream1)
		msg1, err := schema.ConcatMessageStream(stream1)
		assert.Nil(t, err)
		assert.Equal(t, schema.Assistant, msg1.Role)
		assert.Equal(t, 1, len(msg1.ToolCalls))

		// Second message should be the tool response
		stream2, hasNext, err := sIter.Next()
		assert.Nil(t, err)
		assert.True(t, hasNext)
		assert.NotNil(t, stream2)
		msg2, err := schema.ConcatMessageStream(stream2)
		assert.Nil(t, err)
		assert.Equal(t, schema.Tool, msg2.Role)

		// Third message should be the final response
		stream3, hasNext, err := sIter.Next()
		assert.Nil(t, err)
		assert.True(t, hasNext)
		assert.NotNil(t, stream3)
		msg3, err := schema.ConcatMessageStream(stream3)
		assert.Nil(t, err)
		assert.Equal(t, "final response", msg3.Content)

		// Should be no more messages
		_, hasNext, err = sIter.Next()
		assert.Nil(t, err)
		assert.False(t, hasNext)
	})

	t.Run("test stream with streaming tool calls and with concurrent goroutines", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		cm := mockModel.NewMockToolCallingChatModel(ctrl)
		fakeTool := &fakeStreamToolGreetForTest{}

		info, err := fakeTool.Info(ctx)
		assert.NoError(t, err)

		// Mock model response with tool call
		cm.EXPECT().Stream(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(schema.StreamReaderFromArray([]*schema.Message{schema.AssistantMessage("",
				[]schema.ToolCall{
					{
						ID: "tool-call-1",
						Function: schema.FunctionCall{
							Name:      info.Name,
							Arguments: `{"name": "test user"}`,
						},
					},
				})}), nil).
			Times(1)
		cm.EXPECT().Stream(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(schema.StreamReaderFromArray([]*schema.Message{schema.AssistantMessage("final response", nil)}), nil).
			Times(1)
		cm.EXPECT().WithTools(gomock.Any()).Return(cm, nil).AnyTimes()

		// Create agent with MessageFuture
		option, future := WithMessageFuture()
		a, err := NewAgent(ctx, &AgentConfig{
			ToolCallingModel: cm,
			ToolsConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{fakeTool},
			},
			MaxStep: 3,
		})
		assert.Nil(t, err)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Get message streams from future
			sIter := future.GetMessageStreams()

			// First message should be the assistant message for tool calling
			stream1, hasNext, err_ := sIter.Next()
			assert.Nil(t, err_)
			assert.True(t, hasNext)
			assert.NotNil(t, stream1)
			msg1, err_ := schema.ConcatMessageStream(stream1)
			assert.Nil(t, err_)
			assert.Equal(t, schema.Assistant, msg1.Role)
			assert.Equal(t, 1, len(msg1.ToolCalls))

			// Second message should be the tool response
			stream2, hasNext, err_ := sIter.Next()
			assert.Nil(t, err_)
			assert.True(t, hasNext)
			assert.NotNil(t, stream2)
			msg2, err_ := schema.ConcatMessageStream(stream2)
			assert.Nil(t, err_)
			assert.Equal(t, schema.Tool, msg2.Role)

			// Third message should be the final response
			stream3, hasNext, err_ := sIter.Next()
			assert.Nil(t, err_)
			assert.True(t, hasNext)
			assert.NotNil(t, stream3)
			msg3, err_ := schema.ConcatMessageStream(stream3)
			assert.Nil(t, err_)
			assert.Equal(t, "final response", msg3.Content)

			// Should be no more messages
			_, hasNext, err_ = sIter.Next()
			assert.Nil(t, err_)
			assert.False(t, hasNext)
		}()

		// Use Stream interface
		stream, err := a.Stream(ctx, []*schema.Message{
			schema.UserMessage("use the greet tool"),
		}, option)
		assert.Nil(t, err)

		// Collect all chunks from stream
		finalResponse, err := schema.ConcatMessageStream(stream)
		assert.Nil(t, err)
		assert.Equal(t, "final response", finalResponse.Content)

		wg.Wait()
	})
}

func TestWithToolOptions(t *testing.T) {
	type dummyOpt struct{ val string }
	opt := tool.WrapImplSpecificOptFn(func(o *dummyOpt) { o.val = "mock" })
	agentOpt := WithToolOptions(opt)
	assert.NotNil(t, agentOpt)
	// The returned value should be an agent.AgentOption (function)
	assert.IsType(t, agentOpt, agentOpt)
}

func TestWithChatModelOptions(t *testing.T) {
	opt := model.WithModel("mock-model")
	agentOpt := WithChatModelOptions(opt)
	assert.NotNil(t, agentOpt)
	assert.IsType(t, agentOpt, agentOpt)
}

func TestWithToolList(t *testing.T) {
	dummyTool := &dummyBaseTool{}
	agentOpt := WithToolList(dummyTool)
	assert.NotNil(t, agentOpt)
	assert.IsType(t, agentOpt, agentOpt)
}

// dummyBaseTool is a minimal implementation of tool.BaseTool for testing.
type dummyBaseTool struct{}

func (d *dummyBaseTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: "dummy"}, nil
}

func (d *dummyBaseTool) InvokableRun(ctx context.Context, _ string, _ ...tool.Option) (string, error) {
	return "dummy-response", nil
}

type assertTool struct {
	toolOptVal      string
	receivedToolOpt bool
}
type toolOpt struct{ val string }

func (a *assertTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: "assert_tool"}, nil
}
func (a *assertTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	opt := tool.GetImplSpecificOptions(&toolOpt{}, opts...)
	if opt.val == a.toolOptVal {
		a.receivedToolOpt = true
	}
	return "tool-response", nil
}

func TestAgentWithAllOptions(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	// Prepare a tool that asserts it receives the tool option
	toolOptVal := "tool-opt-value"
	to := tool.WrapImplSpecificOptFn(func(o *toolOpt) { o.val = toolOptVal })
	at := &assertTool{toolOptVal: toolOptVal}

	// Prepare a mock chat model that asserts it receives the model option
	cm := mockModel.NewMockToolCallingChatModel(ctrl)
	modelOpt := model.WithModel("test-model")
	modelOptReceived := false
	times := 0
	cm.EXPECT().Generate(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ []*schema.Message, opts ...model.Option) (*schema.Message, error) {
			times++
			if times == 1 {
				for _, o := range opts {
					opt := model.GetCommonOptions(&model.Options{}, o)
					if opt.Model != nil && *opt.Model == "test-model" {
						modelOptReceived = true
					}
				}

				info, _ := at.Info(ctx)
				return schema.AssistantMessage("hello max",
						[]schema.ToolCall{
							{
								ID: randStr(),
								Function: schema.FunctionCall{
									Name:      info.Name,
									Arguments: "",
								},
							},
						}),
					nil
			}

			return schema.AssistantMessage("ok", nil), nil
		},
	).AnyTimes()
	cm.EXPECT().WithTools(gomock.Any()).Return(cm, nil).AnyTimes()

	agentOpt := WithToolOptions(to)
	agentOpt2 := WithChatModelOptions(modelOpt)
	agentOpt3 := WithToolList(at)

	a, err := NewAgent(ctx, &AgentConfig{
		ToolCallingModel: cm,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: []tool.BaseTool{&dummyBaseTool{}},
		},
		MaxStep: 20,
	})
	assert.NoError(t, err)

	_, err = a.Generate(ctx, []*schema.Message{
		schema.UserMessage("call the tool"),
	}, agentOpt, agentOpt2, agentOpt3)
	assert.NoError(t, err)
	assert.True(t, modelOptReceived, "model option should be received by chat model")
	assert.True(t, at.receivedToolOpt, "tool option should be received by tool")
}
