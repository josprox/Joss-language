package core

import (
	"fmt"

	"github.com/jossecurity/joss/pkg/parser"
)

// WebSocket Implementation
func (r *Runtime) executeWebSocketMethod(instance *Instance, method string, args []interface{}) interface{} {
	switch method {
	case "broadcast":
		if len(args) > 0 {
			msg := args[0]
			if BroadcastFunc != nil {
				BroadcastFunc(msg)
				return true
			} else {
				fmt.Println("[WebSocket] Error: BroadcastFunc not initialized")
			}
		}
		return false

	case "send":
		// $ws->send(msg)
		if len(args) > 0 {
			msg := args[0]
			if connVal, ok := instance.Fields["_sender"]; ok {
				if sender, ok := connVal.(func(interface{}) error); ok {
					sender(msg)
					return true
				}
			}
		}
		return false

	case "onMessage":
		// $ws->onMessage(func($msg) { ... })
		if len(args) > 0 {
			// BoundMethod (e.g. $this.handle)
			if fn, ok := args[0].(*BoundMethod); ok {
				instance.Fields["_on_message"] = fn
				return true
			}
			// FunctionLiteral (Anonymous function)
			if fn, ok := args[0].(*parser.FunctionLiteral); ok {
				instance.Fields["_on_message"] = fn
				return true
			}
		}
		return false

	case "close":
		if closer, ok := instance.Fields["_closer"].(func() error); ok {
			if err := closer(); err != nil {
				fmt.Printf("[WebSocket] Error closing connection: %v\n", err)
				return false
			}
			return true
		}
		return false
	}
	return nil
}
