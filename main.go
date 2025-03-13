package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/bootun/commander/config"
	"github.com/bootun/commander/model"
	"github.com/bootun/commander/prompt"
	"github.com/chzyer/readline"
	"github.com/cloudwego/eino/schema"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m" // 红色
	colorGreen  = "\033[32m" // 绿色
	colorYellow = "\033[33m" // 黄色
	colorBlue   = "\033[34m" // 蓝色
	colorPurple = "\033[35m" // 紫色
	colorCyan   = "\033[36m" // 青色
)

var (
	configPath = flag.String("config", os.Getenv("COMMANDER_CONFIG"), "配置文件路径，默认为当前目录下的config.yml")
)

func main() {
	// 解析命令行参数
	flag.Parse()

	// 初始化日志文件
	logFile, err := os.Create("commander.log")
	if err != nil {
		log.Fatalf("创建日志文件失败: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	// 加载配置文件
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 初始化模型
	ctx := context.Background()
	team, err := model.NewTeam(ctx, cfg)
	if err != nil {
		log.Fatalf("模型初始化失败: %v", err)
	}
	// 获取用户输入
	PrintAssistantMessage("有什么我可以帮助你的?")
	userQuestion := readInputFromStdin(fmt.Sprintf("%s>%s ", colorYellow, colorReset))

	// 整合用户问题, 初始化聊天历史
	chatHistory := []*schema.Message{
		schema.SystemMessage(prompt.GetInitialReasoningPrompt(runtime.GOOS)),
		schema.UserMessage(userQuestion),
	}
	// 统计轮次
	round := 0
	PrintAssistantMessage("正在理解问题...")
	for {
		// 开始进行推理
		reasoningPrompt := append(chatHistory, schema.SystemMessage(prompt.GetFinishReasoningPrompt()))
		result, err := team.Reasoning.Generate(ctx, reasoningPrompt)
		if err != nil {
			log.Printf("推理模型生成失败: %v", err)
			PrintWarningMessage("推理模型生成失败, 请重新尝试")
			break
		}
		if strings.Contains(result.Content, "[finish]") {
			break
		}
		// 打印推理结果
		PrintAssistantMessage(result.Content)

		// 结构化模型提取命令
		structuredResult, err := team.Structured.Generate(ctx, []*schema.Message{
			schema.SystemMessage(prompt.GetJSONStructuredPrompt()),
			schema.UserMessage(result.Content),
		})
		if err != nil {
			log.Printf("结构化模型生成失败: %v", err)
			PrintWarningMessage("结构化模型提取命令失败")
			break
		}

		var cmdMsg Command
		if err := json.Unmarshal([]byte(structuredResult.Content), &cmdMsg); err != nil {
			log.Printf("解析结构化模型JSON命令时出现了错误: %v, 回答内容: %s", err, structuredResult.Content)
			PrintWarningMessage("解析结构化模型JSON命令时出现了错误")
			break
		}

		userDir, err := os.Getwd()
		if err != nil {
			log.Fatalf("获取当前目录失败: %v", err)
		}

		// 判断当前命令是否安全
		securityResult, err := team.Security.Generate(ctx, []*schema.Message{
			schema.SystemMessage(prompt.GetSecurityPrompt(runtime.GOOS, userDir, cmdMsg.Command)),
		})
		if err != nil {
			log.Printf("安全模型生成失败: %v", err)
			PrintWarningMessage("安全模型生成失败")
			break
		}

		var securityMsg SecurityMsg
		if err := json.Unmarshal([]byte(securityResult.Content), &securityMsg); err != nil {
			log.Fatalf("解析安全模型JSON命令时出现了错误: %v, 回答内容: %s", err, securityResult.Content)
		}
		if !securityMsg.Safe {
			PrintAssistantMessage(fmt.Sprintf("我希望在[%s]目录下执行命令[%v]", userDir, cmdMsg.Command))
			PrintWarningMessage(fmt.Sprintf("是否允许执行[%s]命令? %s", cmdMsg.Command, securityMsg.Reason))
			for {
				res := readInputFromStdin("(y/n):")
				if res == "y" {
					break
				}
				if res == "n" {
					log.Printf("用户终止了[%s]命令的执行", cmdMsg.Command)
					PrintWarningMessage("任务已终止")
					chatHistory = append(chatHistory, schema.UserMessage(fmt.Sprintf("我终止了[%s]命令的执行", cmdMsg.Command)))
					break
				}
			}
			log.Printf("用户允许了[%s]命令的执行", cmdMsg.Command)
		}
		PrintAssistantMessage("正在执行命令...")
		// 执行AI想要执行的命令
		command := exec.Command("sh", "-c", cmdMsg.Command)
		output, err := command.CombinedOutput() // 使用CombinedOutput同时捕获stdout和stderr
		outputStr := string(output)
		if err != nil {
			log.Printf("执行命令%s时发生了错误: %v", cmdMsg, err)
			// 即使命令失败，也记录错误输出
			outputStr += fmt.Sprintf("\n执行%s命令时发生错误: %v", cmdMsg.Command, err)
		}
		log.Printf("命令[%s]的执行结果: %v", cmdMsg.Command, outputStr)
		PrintAssistantMessage("正在分析命令执行结果...")
		chatHistory = append(chatHistory, schema.AssistantMessage(fmt.Sprintf("我执行了%s命令, 结果是%s", cmdMsg.Command, outputStr), []schema.ToolCall{}))
		round++
		fmt.Printf("------------------------------------------------------\n")
		if round >= cfg.MaxRounds {
			chatHistory = append(chatHistory, schema.SystemMessage("已达到用户设置的最大尝试次数，本次任务已终止"))
			PrintWarningMessage("已达到最大尝试次数，任务已终止。")
			log.Printf("达到最大轮次，任务已终止。")
			break
		}
	}
	actorResult, err := team.Actor.Generate(ctx, append(chatHistory, schema.SystemMessage("请根据用户的问题以及你进行的操作和结果来回答用户的问题")))
	if err != nil {
		log.Printf("回答模型生成失败: %v", err)
		PrintWarningMessage("回答模型生成失败")
		return
	}
	PrintAssistantMessage(actorResult.Content)
	PrintAssistantMessage(fmt.Sprintf("本次回答共进行了%d轮尝试", round+1))
}

// readInputFromStdin 读取用户输入
func readInputFromStdin(prompt string) string {
	rl, err := readline.New(prompt)
	if err != nil {
		log.Fatalf("初始化readline失败: %v", err)
	}
	defer rl.Close()
	line, err := rl.Readline()
	if err != nil {
		log.Fatalf("读取输入失败: %v", err)
	}
	return line
}

type Command struct {
	Command string `json:"command"`
}

func (c Command) String() string {
	return fmt.Sprintf("%v", c.Command)
}

type SecurityMsg struct {
	Safe   bool   `json:"safe"`
	Reason string `json:"reason"`
}

// PrintAssistantMessage 打印AI的回答
func PrintAssistantMessage(message string) {
	fmt.Printf("%s[Commander]%s%s[%s]%s: %s%s%s\n", colorYellow, colorReset, colorBlue, time.Now().Format("2006-01-02 15:04:05"), colorReset, colorCyan, message, colorReset)
}

// PrintWarningMessage 打印警告信息
func PrintWarningMessage(message string) {
	fmt.Printf("%s%s%s\n", colorRed, message, colorReset)
}
