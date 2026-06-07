package server

import (
	"context"
	"encoding/json"
	"go-situation-puzzle/agent"
	"log"
	"net/http"
	"sync"

	"github.com/coder/websocket"
)

type client struct {
	Username string
	Addr     string
	sendChan chan []byte
}

type msgServer struct {
	rooms       map[string]map[*websocket.Conn]*client
	soups       map[string]string
	multiplexer *http.ServeMux
	mutex       sync.RWMutex
	puzzleAgent *agent.PuzzleAgent
}

func NewMsgServer(puzzleAgent *agent.PuzzleAgent) *msgServer {
	ms := &msgServer{
		rooms:       make(map[string]map[*websocket.Conn]*client),
		soups:       make(map[string]string),
		multiplexer: http.NewServeMux(),
		puzzleAgent: puzzleAgent,
	}
	ms.multiplexer.HandleFunc("/ws", ms.handleBroadcast)
	ms.multiplexer.Handle("/", http.FileServer(http.Dir("./html")))
	return ms
}

func (ms *msgServer) GetMultiplexer() *http.ServeMux {
	return ms.multiplexer
}

func (ms *msgServer) addClient(roomID string, conn *websocket.Conn, cli *client) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	cli.sendChan = make(chan []byte, 64)
	if _, ok := ms.rooms[roomID]; !ok {
		ms.rooms[roomID] = make(map[*websocket.Conn]*client)
	}
	ms.rooms[roomID][conn] = cli

	go ms.writePump(conn, cli.sendChan, roomID)
}

func (ms *msgServer) writePump(conn *websocket.Conn, sendChan <-chan []byte, roomID string) {
	log.Println("writePump started for room:", roomID)
	for msg := range sendChan {
		log.Printf("writePump writing %d bytes to room=%s", len(msg), roomID)
		err := conn.Write(context.Background(), websocket.MessageText, msg)
		if err != nil {
			log.Println("单连接发送失败:", err)
			_ = conn.Close(websocket.StatusInternalError, "连接异常")
			ms.removeClient(roomID, conn)
			return
		}
		log.Println("writePump write OK")
	}
	log.Println("writePump ended for room:", roomID)
}

func (ms *msgServer) removeClient(roomID string, conn *websocket.Conn) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	room, ok := ms.rooms[roomID]
	if !ok {
		return
	}

	cli, exists := room[conn]
	if exists {
		close(cli.sendChan)
		delete(room, conn)
	}

	if len(room) == 0 {
		delete(ms.rooms, roomID)
		delete(ms.soups, roomID)
	}
}

func (ms *msgServer) sendSoup(roomID string, conn *websocket.Conn) {
	ms.mutex.RLock()
	soup := ms.soups[roomID]
	ms.mutex.RUnlock()

	msg, err := json.Marshal(Message{
		Role:    "server",
		Type:    "soup",
		Content: soup,
	})
	if err != nil {
		log.Println("序列化汤面失败:", err)
		return
	}

	if conn != nil {
		ms.mutex.RLock()
		room, exists := ms.rooms[roomID]
		if !exists {
			ms.mutex.RUnlock()
			return
		}
		cli, exists := room[conn]
		ms.mutex.RUnlock()
		if !exists {
			return
		}
		select {
		case cli.sendChan <- msg:
		default:
			log.Println("发送汤面通道阻塞")
		}
	} else {
		ms.broadcast(roomID, websocket.MessageText, msg)
	}
}

func (ms *msgServer) broadcast(roomID string, msgType websocket.MessageType, message []byte) {
	ms.mutex.RLock()
	roomConns, ok := ms.rooms[roomID]
	if !ok {
		ms.mutex.RUnlock()
		log.Printf("broadcast: room=%s not found", roomID)
		return
	}

	conns := make([]*client, 0, len(roomConns))
	for _, cli := range roomConns {
		conns = append(conns, cli)
	}
	ms.mutex.RUnlock()

	log.Printf("broadcast: room=%s, clients=%d", roomID, len(conns))

	var dropLog string
	if msgType == websocket.MessageBinary {
		dropLog = "二进制消息发送通道满，消息丢弃"
	} else {
		dropLog = "发送通道阻塞，可能客户端异常"
	}

	for _, cli := range conns {
		select {
		case cli.sendChan <- message:
		default:
			log.Println(dropLog)
		}
	}
}

func (ms *msgServer) getOnlineUsers(roomID string) []string {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	var users []string
	room, ok := ms.rooms[roomID]
	if !ok {
		return users
	}
	for _, cli := range room {
		users = append(users, cli.Username)
	}
	return users
}

type Message struct {
	Role    string `json:"role"`
	Type    string `json:"type"`
	Content string `json:"content"`
}

func (ms *msgServer) broadcastUserList(roomID string) error {
	users := ms.getOnlineUsers(roomID)
	msg, err := json.Marshal(Message{
		Role:    "server",
		Type:    "userlist",
		Content: joinStrings(users, ", "),
	})
	if err != nil {
		return err
	}
	ms.broadcast(roomID, websocket.MessageText, msg)
	return nil
}

func (ms *msgServer) sendTypingStart(roomID string) {
	msg, err := json.Marshal(Message{
		Role:    "server",
		Type:    "typing_start",
		Content: "",
	})
	if err != nil {
		log.Println("序列化失败:", err)
		return
	}
	ms.broadcast(roomID, websocket.MessageText, msg)
}

func (ms *msgServer) sendTypingEnd(roomID string) {
	msg, err := json.Marshal(Message{
		Role:    "server",
		Type:    "typing_end",
		Content: "",
	})
	if err != nil {
		log.Println("序列化失败:", err)
		return
	}
	ms.broadcast(roomID, websocket.MessageText, msg)
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	res := strs[0]
	for i := 1; i < len(strs); i++ {
		res += sep + strs[i]
	}
	return res
}

func (ms *msgServer) handleBroadcast(w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Query().Get("id")
	if roomID == "" {
		log.Println("房间ID参数缺失")
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		username = r.RemoteAddr
	}

	opts := &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	}

	conn, err := websocket.Accept(w, r, opts)
	if err != nil {
		log.Println("websocket 升级失败:", err)
		return
	}

	cli := &client{Username: username, Addr: r.RemoteAddr}
	ms.addClient(roomID, conn, cli)
	log.Printf("房间[%s] 客户端 %s 已接入", roomID, username)

	defer func() {
		ms.removeClient(roomID, conn)
		log.Printf("房间[%s] 客户端 %s 断开", roomID, username)
		_ = ms.broadcastUserList(roomID)
		_ = conn.Close(websocket.StatusNormalClosure, "")
	}()

	_ = ms.broadcastUserList(roomID)
	ms.sendSoup(roomID, conn)

	for {
		msgType, message, err := conn.Read(context.Background())
		if err != nil {
			log.Printf("客户端 %s 读取断开", username)
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Println("消息解析失败:", err)
			continue
		}
		log.Printf("收到 [%s]: %s", msg.Role, msg.Content)

		switch msg.Type {
		case "init":
			log.Println("收到 init 消息, 房间ID:", roomID)
			if err := ms.puzzleAgent.DelSess(context.Background(), roomID); err != nil {
				log.Println("删除会话失败:", err)
				continue
			}

			ms.mutex.Lock()
			ms.soups[roomID] = ""
			ms.mutex.Unlock()

			ms.sendSoup(roomID, nil)
			ms.sendTypingStart(roomID)

			go func() {
				log.Println("开始获取新谜题...")

				ms.puzzleAgent.OutputMessage(context.Background(), roomID, "请给我一个新的海龟汤谜题", func(chunk string) {
					log.Printf("收到AI响应片段: %s", chunk)
					ms.mutex.Lock()
					ms.soups[roomID] += chunk
					ms.mutex.Unlock()

					jsonChunk, err := json.Marshal(Message{
						Role:    "server",
						Type:    "init",
						Content: chunk,
					})
					if err != nil {
						log.Println("序列化失败:", err)
						return
					}
					log.Printf("发送init消息: %s", string(jsonChunk))
					ms.broadcast(roomID, websocket.MessageText, jsonChunk)
				})
				ms.sendTypingEnd(roomID)
				log.Println("获取谜题完成")
			}()

		case "question":
			ms.broadcast(roomID, msgType, message)
			ms.sendTypingStart(roomID)

			go func() {
				ms.puzzleAgent.OutputMessage(context.Background(), roomID, msg.Content, func(chunk string) {
					jsonChunk, err := json.Marshal(Message{
						Role:    "server",
						Type:    "answer",
						Content: chunk,
					})
					if err != nil {
						log.Println("序列化失败:", err)
						return
					}
					ms.broadcast(roomID, websocket.MessageText, jsonChunk)
				})
				ms.sendTypingEnd(roomID)
			}()

		case "chat":
			ms.broadcast(roomID, msgType, message)

		default:
			log.Println("未知消息类型:", msg.Type)
		}
	}
}
