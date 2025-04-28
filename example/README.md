# Go-Queue 示例

本目录包含了 Go-Queue 库的使用示例，展示了不同的使用场景和功能。

## 目录结构

每个示例都在独立的目录中，包含自己的 `main.go` 文件：

- `basic/` - 基础队列操作示例
- `retry/` - 任务重试机制示例
- `multi_queues/` - 多队列并行处理示例

## 基础示例

`basic/main.go` 演示了最基本的队列使用方法，包括：

- 创建队列服务
- 注册内存队列驱动
- 配置并注册工作器
- 添加即时任务和延迟任务
- 启动并处理队列

运行示例：

```bash
cd basic
go run main.go
```

## 重试机制示例

`retry/main.go` 演示了任务失败重试的功能，包括：

- 配置重试次数和重试间隔
- 处理失败任务并进行重试
- 使用回调函数监控任务执行状态

运行示例：

```bash
cd retry
go run main.go
```

## 多队列示例

`multi_queues/main.go` 演示了多队列并行处理的功能，包括：

- 创建和配置多个队列
- 为不同优先级的任务分配不同的资源
- 并行处理多个队列的任务

运行示例：

```bash
cd multi_queues
go run main.go
```

## 运行所有示例

以下脚本可以顺序运行所有示例：

```bash
for dir in basic retry multi_queues; do
    echo "Running $dir example..."
    cd $dir
    go run main.go
    cd ..
    echo "Done."
    echo "------------------------------"
    sleep 1
done
```

## 注意事项

- 示例中使用的是内存队列，数据不会持久化
- 示例代码中包含了详细的注释，帮助理解每一步的功能