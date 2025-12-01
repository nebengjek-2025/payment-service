package config

import (
	kafkaPkgConfluent "payment-service/src/pkg/kafka/confluent"
	"payment-service/src/pkg/log"

	"github.com/spf13/viper"
)

func NewKafkaConfig(viper *viper.Viper) kafkaPkgConfluent.KafkaConfig {
	configKafka := kafkaPkgConfluent.Cfg{
		KafkaUrl:      viper.GetString("kafka.bootstrap.servers"),
		KafkaUsername: viper.GetString("kafka.username"),
		KafkaPassword: viper.GetString("kafka.password"),
		KafkaCaCert:   viper.GetString("kafka.cacert"),
		AppName:       viper.GetString("kafka.group.id"),
		Offset:        viper.GetString("kafka.auto.offset.reset"),
	}
	return kafkaPkgConfluent.InitKafkaConfig(configKafka)

}

func NewKafkaProducer(config *viper.Viper, log log.Log) kafkaPkgConfluent.Producer {
	if !config.GetBool("kafka.producer.enabled") {
		log.Info("kafka-config", "Kafka producer is disabled in configuration", "kafka", "")
		return nil
	}
	kafkaProducer, err := kafkaPkgConfluent.NewProducer(kafkaPkgConfluent.GetConfig().GetKafkaConfig(), log)
	if err != nil {
		panic(err)
	}

	return kafkaProducer
}

func NewKafkaConsumer(config *viper.Viper, log log.Log) kafkaPkgConfluent.Consumer {
	kafkaConsumer, err := kafkaPkgConfluent.NewConsumer(kafkaPkgConfluent.GetConfig().GetKafkaConfig(), log)
	if err != nil {
		panic(err)
	}

	return kafkaConsumer
}
