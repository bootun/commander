package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/prompts"
)

func main() {
	// 开启下面这行注释以禁用日志输出
	//log.SetOutput(io.Discard)
	key := os.Getenv("OPENAI_API_KEY")
	// 1.初始化推理模型
	reasonModel, err := openai.New(
		openai.WithBaseURL("https://api.siliconflow.cn/v1"),
		openai.WithToken(key),
		openai.WithModel("Qwen/Qwen2.5-Coder-32B-Instruct"),
	)
	if err != nil {
		log.Fatalf("决策模型初始化失败: %v", err)
	}

	// 2.构造推理决策使用的prompt
	// 读取用户输入
	userQuestion := readInputFromStdin()
	reasonPrompt, err := prompts.NewChatPromptTemplate([]prompts.MessageFormatter{
		prompts.NewSystemMessagePromptTemplate(
			`你是一个善于执行任务的AI助手, 你需要根据用户的需求，以“我”为主语，从用户的视角来判断接下来在命令行内执行什么命令有助于解决该问题。如果有多个选择，你只能选择你认为当前对你最有帮助的一条命令。当前的操作系统是{{.os_name}}`,
			[]string{"os_name"},
		),
		prompts.NewHumanMessagePromptTemplate(
			//`帮我看下当前目录下有没有main.go这个文件?`,
			//`在当前目录下创建一个helloworld的目录`,
			//`请求www.baidu.com的首页`,
			//`列出当前目录下的文件`,
			userQuestion,
			nil,
		),
	}).Format(map[string]any{
		"os_name": runtime.GOOS,
	})
	if err != nil {
		log.Fatalf("用户问题Prompt初始化失败: %v", err)
	}

	ctx := context.Background()
	// 3.让AI进行决策，要执行什么命令
	completion, err := llms.GenerateFromSinglePrompt(ctx, reasonModel, reasonPrompt)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("推理模型返回的决策内容: %v", completion)

	// 4.构造总结JSON输出的模型，使用结构化输出能力比较强的模型
	commandModel, err := openai.New(
		openai.WithBaseURL("https://api.siliconflow.cn/v1"),
		openai.WithToken(key),
		openai.WithModel("Qwen/Qwen2.5-32B-Instruct"),
	)
	if err != nil {
		log.Fatalf("JSON输出模型初始化失败: %v", err)
	}

	// 5.构造JSON模式的prompt
	jsonMode, err := prompts.NewChatPromptTemplate([]prompts.MessageFormatter{
		prompts.NewSystemMessagePromptTemplate(
			`你是一个JSON构造器，能够从用户的输入里解析命令，并以JSON格式输出，格式如下:
{"command": 要执行的命令, "args": [参数1,参数2,..., 参数n]}
无论用户说了什么，你始终要以这种格式进行返回，没有任何例外
	`, nil),
		prompts.NewHumanMessagePromptTemplate(
			completion, // 将模型的决策作为用户输入放进prompt里
			nil,
		),
	}).Format(nil)
	if err != nil {
		log.Fatalf("JSON模型Prompt初始化失败: %v", err)
	}

	cmdJson, err := llms.GenerateFromSinglePrompt(ctx, commandModel, jsonMode, llms.WithJSONMode())
	if err != nil {
		log.Fatalf("模型在输出JSON格式的回答时发生了错误: %v", err)
	}
	log.Printf("模型提取出的JSON输出: %v", cmdJson)

	var cmdMsg Command
	// 6.解析出JSON命令
	if err := json.Unmarshal([]byte(cmdJson), &cmdMsg); err != nil {
		log.Fatalf("解析JSON命令时出现了错误: %v", err)
	}
	// 7.执行AI想要执行的命令
	command := exec.Command(cmdMsg.Command, cmdMsg.Args...)
	output, err := command.Output()
	if err != nil {
		log.Fatalf("执行命令%s时发生了错误: %v", cmdMsg, err)
	}
	log.Printf("命令[%s]的执行结果: %v", cmdMsg, output)

	// 8.构造结果prompt
	resultPrompt := prompts.NewChatPromptTemplate([]prompts.MessageFormatter{
		prompts.NewSystemMessagePromptTemplate(
			`你是一个善于执行IT任务的AI助手, 你需要根据用户的需求来一步一步的操作计算机，观察结果并判断是否要继续执行命令。
用户的问题是:{{.user_question}}
下面是你已经执行过的命令和输出:
{{.command_logs}}
请根据这些内容，对用户进行回复，告诉用户你做了什么，结果是什么`,
			[]string{"user_question", "command_logs"},
		),
	})
	history := NewCommandHistory(cmdMsg, string(output))

	resultPromptString, err := resultPrompt.Format(map[string]any{
		"user_question": userQuestion,
		"command_logs":  history.String(),
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("最终推理prompt: %v", resultPromptString)

	resultStr, err := llms.GenerateFromSinglePrompt(ctx, reasonModel, resultPromptString)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Commander: %v", resultStr)
}

func readInputFromStdin() string {
	fmt.Printf("> ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return scanner.Text()
}

type Command struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

func (c Command) String() string {
	argsBuf := strings.Builder{}
	argsBuf.Grow(len(c.Args) * 5)
	for _, arg := range c.Args {
		argsBuf.WriteString(arg)
		argsBuf.WriteString(" ")
	}
	return fmt.Sprintf("%v %s", c.Command, argsBuf.String())
}

type CommandHistory struct {
	Command Command
	Output  string
}

func NewCommandHistory(cmd Command, output string) CommandHistory {
	return CommandHistory{
		Command: cmd,
		Output:  output,
	}
}

func (c CommandHistory) String() string {
	return fmt.Sprintf("你执行了命令: %v\n执行结果: %v\n", c.Command.String(), c.Output)
}
