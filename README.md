智能命令行AI助手 - Commander
===

可以帮你执行简单操作的命令行助手。具备以下特性:  
- 基于AI的操作安全性审查
- 自行规划任务并执行
- 简单易懂

> ⚠️**注意: 该项目作为AI Agent入门原理示例, 尽量以最简单的方式向读者展示Agent的核心思想,并且最小化对框架的依赖程度,因此代码量和实现方式不一定优雅.**

## 配套文章
[探索AI工具幕后，打造你自己的Agent应用](https://tech.qimao.com/build-your-command-agent-tool/) 

## 效果演示
![example1](./example1.png)
![example2](./example2.png)

(图上每个横线分隔的就是一轮对话)

## 使用方式

### 配置
1. 创建 `config.yml` 并填写相关配置信息(可参考`config.example.yml`文件):
   - `reasoning_model`: 用于理解用户需求和决策的模型
   - `structured_model`: 用于生成结构化输出的模型
   - `security_model`: 用于安全检查的模型
   - `actor_model`: 用于最终回答的模型

每个模型配置需要包含:
- `model_id`: 模型ID
- `base_url`: API基础URL
- `token`: API令牌

### 运行
有多种方式指定配置文件:
1. 默认情况下，程序会查找当前目录下的 `config.yml`
2. 通过命令行参数指定: `./commander --config=/path/to/config.yml`
3. 通过环境变量指定: `COMMANDER_CONFIG=/path/to/config.yml ./commander`

> ⚠️警告: 代码仅作为示例用途，使用时请注意操作安全
