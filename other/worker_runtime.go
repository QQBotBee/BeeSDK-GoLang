//go:build bee_worker_runtime

package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// decodeArgBytes 将 C 壳通过 IPC 传来的 Base64 参数还原为原始字节。
func decodeArgBytes(value string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(value)
}

// decodeBeeBytes 将 Bee 使用的 GBK 字节转换成 Go 内部使用的 UTF-8 字符串。
func decodeBeeBytes(value []byte) (string, error) {
	decoded, _, err := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), value)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// decodedArgs 批量解码事件参数：先解 Base64，再把 GBK 文本转换为 UTF-8。
func decodedArgs(msg IPCMessage) ([][]byte, error) {
	args := make([][]byte, len(msg.ArgsB64))
	for i, encoded := range msg.ArgsB64 {
		value, err := decodeArgBytes(encoded)
		if err != nil {
			return nil, fmt.Errorf("argument %d: %w", i, err)
		}
		utf8Value, err := decodeBeeBytes(value)
		if err != nil {
			return nil, fmt.Errorf("argument %d GBK: %w", i, err)
		}
		args[i] = []byte(utf8Value)
	}
	return args, nil
}

// dispatchEvent 根据事件名称调用对应的插件回调，并返回处理结果及是否退出工作进程。
func dispatchEvent(msg IPCMessage) (shutdown bool, result IPCMessage) {
	result = IPCMessage{Type: "event_result", ID: msg.ID, Result: MessageContinue}
	if msg.Event == "stop" {
		closeSettingsWindow()
		return true, result
	}
	args, err := decodedArgs(msg)
	if err != nil {
		result.Error = err.Error()
		return false, result
	}

	switch msg.Event {
	case "initialize":
		onInitialize(args)
	case "enable":
		onEnable(args)
	case "disable":
		onDisable(args)
	case "unload":
		onUnload(args)
		return true, result
	case "settings":
		onSettings(args)
	case "channel_private":
		if len(args) < 6 {
			result.Error = "channel_private 参数不足"
			break
		}
		result.Result = onChannelPrivate(
			string(args[0]),
			string(args[1]),
			string(args[2]),
			string(args[3]),
			string(args[4]),
			string(args[5]),
		)
	case "channel_message":
		if len(args) < 6 {
			result.Error = "channel_message 参数不足"
			break
		}
		result.Result = onChannelMessage(
			string(args[0]),
			string(args[1]),
			string(args[2]),
			string(args[3]),
			string(args[4]),
			string(args[5]),
		)
	case "channel_event":
		if len(args) < 7 {
			result.Error = "channel_event 参数不足"
			break
		}
		result.Result = onChannelEvent(
			string(args[0]),
			string(args[1]),
			string(args[2]),
			string(args[3]),
			string(args[4]),
			string(args[5]),
			string(args[6]),
		)
	case "private_message":
		if len(args) < 4 {
			result.Error = "private_message 参数不足"
			break
		}
		result.Result = onPrivateMessage(
			string(args[0]),
			string(args[1]),
			string(args[2]),
			string(args[3]),
		)
	case "group_message":
		if len(args) < 5 {
			result.Error = "group_message 参数不足"
			break
		}
		result.Result = onGroupMessage(
			string(args[0]),
			string(args[1]),
			string(args[2]),
			string(args[3]),
			string(args[4]),
		)
	case "common_event":
		if len(args) < 6 {
			result.Error = "common_event 参数不足"
			break
		}
		result.Result = onCommonEvent(
			string(args[0]),
			string(args[1]),
			string(args[2]),
			string(args[3]),
			string(args[4]),
			string(args[5]),
		)
	default:
		result.Error = "unknown event: " + msg.Event
	}
	return false, result
}

// runWorker 运行 IPC 主循环，从 C 壳读取事件、调用插件回调并写回处理结果。
func runWorker(input io.Reader, output io.Writer) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), 8*1024*1024)
	encoder := json.NewEncoder(output)
	writeMessage := func(message IPCMessage) error { return encoder.Encode(message) }
	readMessage := func() (IPCMessage, error) {
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return IPCMessage{}, err
			}
			return IPCMessage{}, io.EOF
		}
		var message IPCMessage
		if err := json.Unmarshal(scanner.Bytes(), &message); err != nil {
			return IPCMessage{}, err
		}
		return message, nil
	}
	setCurrentAPITransport(newIPCClientForTest(writeMessage, readMessage))
	defer setCurrentAPITransport(nil)

	for {
		msg, err := readMessage()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			_ = encoder.Encode(IPCMessage{Type: "event_result", Error: err.Error()})
			continue
		}
		if msg.Type != "event" {
			_ = encoder.Encode(IPCMessage{Type: "event_result", ID: msg.ID, Error: "unsupported message type"})
			continue
		}
		shutdown, result := dispatchEvent(msg)
		if err := encoder.Encode(result); err != nil {
			return err
		}
		if shutdown {
			return nil
		}
	}
}

// parseHostPID 从命令行参数中读取 Bee 宿主进程 PID，用于监控宿主是否退出。
func parseHostPID(args []string) (uint32, error) {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "--host-pid" {
			value, err := strconv.ParseUint(args[i+1], 10, 32)
			return uint32(value), err
		}
	}
	return 0, errors.New("missing --host-pid")
}

// main 是 Go 工作进程入口：启动宿主监控，并通过标准输入输出运行 IPC 循环。
func main() {
	hostPID, err := parseHostPID(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	startHostWatcher(hostPID)
	if err := runWorker(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
