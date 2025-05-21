// kafkaProducer.go
package kafkalib

import (
	//"fmt"
	"strings"

	"github.com/IBM/sarama"
)

var (
	kafka_producer sarama.AsyncProducer
	//kafka_config   *sarama.Config
)

func Get_Kafka_Producer() sarama.AsyncProducer {
	return kafka_producer
}

func Init_Kafka_Producer(kafka_addr string) {
	if kafka_addr == "" {
		panic("kafka配置错误")
	}
	kafka_addr_arr := strings.Split(kafka_addr, ",")
	kafka_config := sarama.NewConfig()

	//是否等待服务器响应
	kafka_config.Producer.RequiredAcks = sarama.NoResponse //sarama.WaitForLocal

	//随机向partition发送消息
	kafka_config.Producer.Partitioner = sarama.NewRoundRobinPartitioner

	//如果设置成true必须把通道消费掉，要不会造成消息堵塞
	kafka_config.Producer.Return.Successes = false
	kafka_config.Producer.Return.Errors = false

	//注意，版本设置不对的话，kafka会返回很奇怪的错误，并且无法成功发送消息
	kafka_config.Version = sarama.V2_8_1_0

	//使用配置,新建一个异步生产者
	var err error
	kafka_producer, err = sarama.NewAsyncProducer(kafka_addr_arr, kafka_config)
	if err != nil {
		panic(err.Error())
	}

}

// 发送消息
func Sent_Msg(msgval, topic string) {
	msg := &sarama.ProducerMessage{
		Topic: topic,
	}

	//将字符串转化为字节数组
	msg.Value = sarama.ByteEncoder(msgval)

	//使用通道发送
	kafka_producer.Input() <- msg
	return
}
