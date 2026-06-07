package agent

import (
	"context"
	"fmt"
	"go-situation-puzzle/model"
	"math/rand"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"gorm.io/gorm"
)

// 1. 添加一个占位参数，满足大模型对 parameters 不能为空的校验
type PuzzleParams struct {
	// 大模型调用时会传这个字段，我们直接忽略它
	Trigger string `json:"trigger" jsonschema:"description=触发随机查询的指令，随便传个字符串即可"`
}

func GetPuzzleFunc(ctx context.Context, db *gorm.DB) func(ctx context.Context, params *PuzzleParams) (string, error) {
	return func(ctx context.Context, params *PuzzleParams) (string, error) {
		// 2. 忽略 params，直接执行你的随机查询逻辑
		var count int64
		if err := db.Model(&model.Puzzle{}).Count(&count).Error; err != nil {
			return "", fmt.Errorf("获取行数失败: %v", err)
		}

		if count == 0 {
			return "", fmt.Errorf("数据库为空，无法随机查询")
		}
		randomOffset := rand.Intn(int(count))

		var randomPuzzle model.Puzzle
		result := db.Offset(randomOffset).Limit(1).Find(&randomPuzzle)
		if result.Error != nil {
			return "", result.Error
		}
		return randomPuzzle.String(), nil
	}
}

func NewPuzzleDBTool(ctx context.Context, db *gorm.DB) (tool.InvokableTool, error) {
	// 3. 恢复使用 utils.InferTool，它会自动把 PuzzleParams 转成合法的 JSON Schema
	puzzleDBTool, err := utils.InferTool(
		"puzzle_db",
		"PuzzleDBTool is used to search a random puzzle in database",
		GetPuzzleFunc(ctx, db))
	if err != nil {
		return nil, err
	}
	return puzzleDBTool, nil
}
