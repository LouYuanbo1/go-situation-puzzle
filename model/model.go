package model

import (
	"fmt"
)

type Puzzle struct {
	ID       int    `gorm:"primaryKey"`
	Title    string `gorm:"column:title"`
	Question string `gorm:"column:question"`
	Answer   string `gorm:"column:answer"`
}

func (p Puzzle) String() string {
	return fmt.Sprintf("ID: %d, Question: %s, Answer: %s", p.ID, p.Question, p.Answer)
}
