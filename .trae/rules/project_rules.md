
# MCP Interactive Feedback Rules
- follow mcp-feedback-enhanced instructions
- 使用`context7-mcp`查询相关技术文档和最佳实践
- 使用`sequential-thinking`分析复杂需求的技术可行性
- 使用`to-dos`计划整个任务流程

## MCP服务优先级
1. `mcp-feedback-enhanced` - 用户交互和确认
2. `to-dos`计划整个任务流程
2. `sequential-thinking` - 复杂问题分析和深度思考
3. `context7-mcp` - 查询最新库文档和示例

## 工具使用指南

### Sequential Thinking
- **用途**：复杂问题的逐步分析
- **适用场景**：需求分析、方案设计、问题排查
- **使用时机**：遇到复杂逻辑或多步骤问题时

### Context 7
- **用途**：查询最新的技术文档、API参考和代码示例
- **适用场景**：技术调研、最佳实践获取
- **使用时机**：需要了解新技术或验证实现方案时

## 注意事项
- 后端：每次在文件进行更改时，必须进行编译测试，编译后端文件到`/Users/danyhug/GolandProjects/DH-Blog/blog-deploy/backend` 中，每完成一个任务必须编译一次查看是否成功，所有任务完成后删除编译文件；编译成功后不要运行！
- 数据库在 `/Users/danyhug/GolandProjects/DH-Blog/blog-deploy/backend/data/dhblog.db`
- 前端：更改了文件后进行`pnpm run dev`进行测试，编译成功则退出终端；除非我要求，否则不要改任何样式；`npm`命令统一改为使用`pnpm`