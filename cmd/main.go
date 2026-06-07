package main

import (
	"context"
	"fmt"
	"go-situation-puzzle/agent"
	"go-situation-puzzle/config"
	"go-situation-puzzle/model"
	"go-situation-puzzle/server"
	"log"
	"net/http"
	"time"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/schema"
	//"gorm.io/driver/sqlite"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
	store, err := agent.NewStore[*schema.Message]("./sessions")
	if err != nil {
		log.Fatal("创建会话存储失败:", err)
	}

	ctx := context.Background()
	config, err := config.InitConfig()
	if err != nil {
		fmt.Printf("Error initializing config: %v", err)
		return
	}
	chatModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey: config.Deepseek.APIKey,
		Model:  "deepseek-v4-pro",
	})

	db, err := gorm.Open(sqlite.Open("./puzzle.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// ⚠️ SQLite 专属优化：开启 WAL (Write-Ahead Logging) 模式
	// 这能极大提升 SQLite 的并发读写性能，避免 "database is locked" 错误
	db.Exec("PRAGMA journal_mode = WAL;")

	// 3. 自动迁移 (AutoMigrate)
	// 自动创建表、缺失的列和索引 (注意：它不会删除未使用的列，以保护你的数据)
	err = db.AutoMigrate(&model.Puzzle{})
	if err != nil {
		log.Fatalf("迁移失败: %v", err)
	}
	fmt.Println("✅ 数据库连接成功，表已创建/更新！")

	err = db.Create(&model.Puzzle{
		Title:    "刷牙",
		Question: "男子清晨起来刷牙，发现自己牙齿是绿色的，他吓疯了过去。",
		Answer:   "男子是一个守太平间的看管人，最近天平间有尸体丢失，他报警后警察在每具尸体上涂抹了一种荧光粉，在夜间会显现绿色。后来尸体再次消失，警察依然没有线索。男子早上起来刷牙时，发现自己牙齿是绿色的（他有梦游异食癖，半夜梦游起来吃尸体）。",
	}).Error
	if err != nil {
		log.Fatalf("创建谜题失败: %v", err)
	}

	err = db.Create(&model.Puzzle{
		Title:    "女明星的女儿",
		Question: "女明星养育了一个女儿，她年轻时总是时不时给女儿戴上帽子。有一天，她带女儿去医院，女儿在医院得知真相后自杀了。为什么？",
		Answer:   "这位女演员毁容了，她抚养女儿并接受面部移植到自己的脸上。那帽子是她自己的，她时不时给女儿戴上，测量女儿头部的大小，一旦刚刚好，她就带女儿去医院做面部移植。",
	}).Error
	if err != nil {
		log.Fatalf("创建谜题失败: %v", err)
	}

	puzzleAgent := agent.NewDefaultAgent(ctx, chatModel, db, store)

	ms := server.NewMsgServer(puzzleAgent)

	s := &http.Server{
		Handler:      ms.GetMultiplexer(),
		Addr:         ":8080",
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
	}

	log.Println("WebSocket 服务启动 :8080")
	if err := s.ListenAndServe(); err != nil {
		log.Fatal("服务启动失败:", err)
	}
}
