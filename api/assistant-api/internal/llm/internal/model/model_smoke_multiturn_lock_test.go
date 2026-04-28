package internal_model

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
)

// Smoke tests for multi-turn concurrency and lock-risk paths (context switches,
// late tool results, and concurrent packet processing) to ensure no stuck/lock behavior.

func TestModel_MultiTurn_ContextSwitchWithLateToolResult_NoStuck(t *testing.T) {
	// MT2: context switch with late old-tool result should not block newer turn completion.
	e, comm, stream, _ := newModelTestEnv(t)

	require.NoError(t, e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "ctx-1", Text: "first",
	}))
	require.Len(t, stream.sendCalls, 1)

	e.handleResponse(context.Background(), comm, &protos.ChatResponse{
		RequestId:    "ctx-1",
		Success:      true,
		FinishReason: "tool_calls",
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					Contents: []string{"working"},
					ToolCalls: []*protos.ToolCall{
						{Id: "t1", Type: "function", Function: &protos.FunctionCall{Name: "lookup", Arguments: `{"q":"1"}`}},
					},
				},
			},
		},
		Metrics: []*protos.Metric{{Name: "token_count", Value: "3"}},
	})

	require.NoError(t, e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "ctx-2", Text: "second",
	}))
	require.Len(t, stream.sendCalls, 2)

	// Late result from old context should not trigger follow-up send and should not block ctx-2.
	require.NoError(t, e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ContextID: "ctx-1", ToolID: "t1", Name: "lookup", Result: map[string]string{"ok": "1"},
	}))
	require.Len(t, stream.sendCalls, 2)

	e.handleResponse(context.Background(), comm, &protos.ChatResponse{
		RequestId: "ctx-2",
		Success:   true,
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{Contents: []string{"second done"}},
			},
		},
		Metrics: []*protos.Metric{{Name: "token_count", Value: "4"}},
	})

	dones := findPackets[internal_type.LLMResponseDonePacket](comm.pkts)
	require.NotEmpty(t, dones)
	require.Equal(t, "ctx-2", dones[len(dones)-1].ContextID)
}

func TestModel_MultiTurn_ConcurrentUserAndResponse_NoLock(t *testing.T) {
	// MT3: rapid concurrent user packets and responses must not deadlock.
	e, comm, _, _ := newModelTestEnv(t)
	const n = 80

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < n; i++ {
			_ = e.Execute(context.Background(), comm, internal_type.UserInputPacket{
				ContextID: fmt.Sprintf("u-%d", i),
				Text:      "hello",
			})
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < n; i++ {
			e.handleResponse(context.Background(), comm, &protos.ChatResponse{
				RequestId: fmt.Sprintf("u-%d", i),
				Success:   true,
				Data: &protos.Message{
					Role: "assistant",
					Message: &protos.Message_Assistant{
						Assistant: &protos.AssistantMessage{Contents: []string{"ok"}},
					},
				},
				Metrics: []*protos.Metric{{Name: "token_count", Value: "1"}},
			})
		}
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("possible lock: concurrent user+response processing timed out")
	}
}

func TestModel_MultiTurn_ConcurrentToolResultAndInterrupt_NoLock(t *testing.T) {
	// MT4: tool-result and interrupt race must not deadlock.
	e, comm, _, _ := newModelTestEnv(t)
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-race"}
	e.history.AppendAssistant("ctx-race", testToolAssistantMessage("t1", "t2", "t3", "t4"))

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 1; i <= 4; i++ {
			_ = e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
				ContextID: "ctx-race",
				ToolID:    fmt.Sprintf("t%d", i),
				Name:      "fn",
				Result:    map[string]string{"i": fmt.Sprintf("%d", i)},
			})
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			_ = e.Execute(context.Background(), comm, internal_type.LLMInterruptPacket{ContextID: "ctx-race"})
		}
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("possible lock: tool results + interrupt race timed out")
	}
}
