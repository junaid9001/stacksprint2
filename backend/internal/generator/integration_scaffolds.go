package generator

func addInfraBoilerplate(tree *FileTree, req GenerateRequest, root string) {
	if req.Infra.Redis {
		addRedisBoilerplate(tree, req, root)
	}
	if req.Infra.Kafka {
		addKafkaBoilerplate(tree, req, root)
	}
}

func addRedisBoilerplate(tree *FileTree, req GenerateRequest, root string) {
	prefix := root
	if prefix != "" {
		prefix += "/"
	}
	switch req.Language {
	case "go":
		addFile(tree, prefix+"internal/cache/redis.go", "package cache\n\nimport \"os\"\n\ntype RedisCache struct {\n\tAddr string\n}\n\nfunc NewRedisCache() *RedisCache {\n\taddr := os.Getenv(\"REDIS_ADDR\")\n\tif addr == \"\" {\n\t\taddr = \"redis:6379\"\n\t}\n\treturn &RedisCache{Addr: addr}\n}\n\nfunc (r *RedisCache) Ping() string {\n\treturn \"redis configured at \" + r.Addr\n}\n")
	case "node":
		addFile(tree, prefix+"src/cache/redis.js", "export class RedisCache {\n  constructor(addr = process.env.REDIS_ADDR || 'redis:6379') {\n    this.addr = addr;\n  }\n\n  ping() {\n    return `redis configured at ${this.addr}`;\n  }\n}\n")
	case "python":
		addFile(tree, prefix+"app/cache/redis_cache.py", "import os\n\nclass RedisCache:\n    def __init__(self, addr: str | None = None):\n        self.addr = addr or os.getenv('REDIS_ADDR', 'redis:6379')\n\n    def ping(self) -> str:\n        return f'redis configured at {self.addr}'\n")
	}
}

func addKafkaBoilerplate(tree *FileTree, req GenerateRequest, root string) {
	prefix := root
	if prefix != "" {
		prefix += "/"
	}
	switch req.Language {
	case "go":
		addFile(tree, prefix+"internal/messaging/kafka_producer.go", "package messaging\n\nimport \"os\"\n\ntype KafkaProducer struct {\n\tBrokers string\n}\n\nfunc NewKafkaProducer() *KafkaProducer {\n\tb := os.Getenv(\"KAFKA_BROKERS\")\n\tif b == \"\" {\n\t\tb = \"kafka:9092\"\n\t}\n\treturn &KafkaProducer{Brokers: b}\n}\n\nfunc (p *KafkaProducer) Publish(topic, payload string) string {\n\treturn \"publish stub to \" + topic + \" via \" + p.Brokers + \" payload=\" + payload\n}\n")
		addFile(tree, prefix+"internal/messaging/kafka_consumer.go", "package messaging\n\nimport \"os\"\n\ntype KafkaConsumer struct {\n\tBrokers string\n}\n\nfunc NewKafkaConsumer() *KafkaConsumer {\n\tb := os.Getenv(\"KAFKA_BROKERS\")\n\tif b == \"\" {\n\t\tb = \"kafka:9092\"\n\t}\n\treturn &KafkaConsumer{Brokers: b}\n}\n\nfunc (c *KafkaConsumer) Subscribe(topic string) string {\n\treturn \"consumer stub subscribed to \" + topic + \" via \" + c.Brokers\n}\n")
	case "node":
		addFile(tree, prefix+"src/messaging/kafkaProducer.js", "export class KafkaProducer {\n  constructor(brokers = process.env.KAFKA_BROKERS || 'kafka:9092') {\n    this.brokers = brokers;\n  }\n\n  publish(topic, payload) {\n    return `publish stub to ${topic} via ${this.brokers}: ${payload}`;\n  }\n}\n")
		addFile(tree, prefix+"src/messaging/kafkaConsumer.js", "export class KafkaConsumer {\n  constructor(brokers = process.env.KAFKA_BROKERS || 'kafka:9092') {\n    this.brokers = brokers;\n  }\n\n  subscribe(topic) {\n    return `consumer stub subscribed to ${topic} via ${this.brokers}`;\n  }\n}\n")
	case "python":
		addFile(tree, prefix+"app/messaging/kafka_producer.py", "import os\n\nclass KafkaProducer:\n    def __init__(self, brokers: str | None = None):\n        self.brokers = brokers or os.getenv('KAFKA_BROKERS', 'kafka:9092')\n\n    def publish(self, topic: str, payload: str) -> str:\n        return f'publish stub to {topic} via {self.brokers}: {payload}'\n")
		addFile(tree, prefix+"app/messaging/kafka_consumer.py", "import os\n\nclass KafkaConsumer:\n    def __init__(self, brokers: str | None = None):\n        self.brokers = brokers or os.getenv('KAFKA_BROKERS', 'kafka:9092')\n\n    def subscribe(self, topic: str) -> str:\n        return f'consumer stub subscribed to {topic} via {self.brokers}'\n")
	}
}

func addGRPCBoilerplate(tree *FileTree, req GenerateRequest, root string) {
	prefix := root
	if prefix != "" {
		prefix += "/"
	}
	switch req.Language {
	case "go":
		addFile(tree, prefix+"internal/grpc/server/server.go", "package server\n\nimport \"context\"\n\ntype PingRequest struct{ Source string }\ntype PingReply struct{ Message string }\n\ntype Service struct{}\n\nfunc (s *Service) Ping(_ context.Context, req *PingRequest) (*PingReply, error) {\n\treturn &PingReply{Message: \"pong from \" + req.Source}, nil\n}\n")
		addFile(tree, prefix+"internal/grpc/client/client.go", "package client\n\nimport \"fmt\"\n\ntype Client struct{ Address string }\n\nfunc New(address string) *Client {\n\tif address == \"\" {\n\t\taddress = \"127.0.0.1:9090\"\n\t}\n\treturn &Client{Address: address}\n}\n\nfunc (c *Client) Ping() string {\n\treturn fmt.Sprintf(\"ping stub to %s\", c.Address)\n}\n")
	case "node":
		addFile(tree, prefix+"src/grpc/server.js", "export function startGrpcServer() {\n  return 'gRPC server stub started';\n}\n")
		addFile(tree, prefix+"src/grpc/client.js", "export function pingGrpc(target = '127.0.0.1:9090') {\n  return `gRPC client stub pinging ${target}`;\n}\n")
	case "python":
		addFile(tree, prefix+"app/grpc_server.py", "def start_grpc_server() -> str:\n    return 'gRPC server stub started'\n")
		addFile(tree, prefix+"app/grpc_client.py", "def ping_grpc(target: str = '127.0.0.1:9090') -> str:\n    return f'gRPC client stub pinging {target}'\n")
	}
}
