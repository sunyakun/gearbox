package task

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ThreeDotsLabs/watermill-sql/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors"
)

func defaultInsertArgs(topic string, msgs message.Messages) ([]interface{}, error) {
	var args []interface{}
	for _, msg := range msgs {
		metadata, err := json.Marshal(msg.Metadata)
		if err != nil {
			return nil, errors.Wrapf(err, "could not marshal metadata into JSON for message %s", msg.UUID)
		}

		args = append(args, topic, msg.UUID, []byte(msg.Payload), metadata)
	}

	return args, nil
}

type MySQLSchema struct {
	sql.DefaultMySQLSchema

	TaskHubName     string
	OffsetFieldName string
}

func (s MySQLSchema) SelectQuery(topic string, consumerGroup string, offsetsAdapter sql.OffsetsAdapter) (string, []interface{}) {
	nextOffsetQuery, nextOffsetArgs := offsetsAdapter.NextOffsetQuery(topic, consumerGroup)
	selectQuery := fmt.Sprintf(`
		SELECT %s as offset, uuid, payload, metadata FROM %s
		WHERE
			%s > (%s) AND topic='%s'
		ORDER BY
			%s ASC
		LIMIT 1`,
		s.OffsetFieldName, s.MessagesTable(s.TaskHubName),
		s.OffsetFieldName, nextOffsetQuery, topic,
		s.OffsetFieldName)

	return selectQuery, nextOffsetArgs
}

func (s MySQLSchema) InsertQuery(topic string, msgs message.Messages) (string, []interface{}, error) {
	insertQuery := fmt.Sprintf(
		`INSERT INTO %s (topic, uuid, payload, metadata) VALUES %s`,
		s.MessagesTable(s.TaskHubName),
		strings.TrimRight(strings.Repeat(`(?,?,?,?),`, len(msgs)), ","),
	)

	args, err := defaultInsertArgs(topic, msgs)
	if err != nil {
		return "", nil, err
	}

	return insertQuery, args, nil
}

type MySQLOffsetScheme struct {
	sql.DefaultMySQLOffsetsAdapter

	TaskHubName string
}

func (a MySQLOffsetScheme) AckMessageQuery(topic string, offset int, consumerGroup string) (string, []interface{}) {
	ackQuery := `UPDATE ` + a.MessagesOffsetsTable(a.TaskHubName) + ` SET offset_acked = ? WHERE consumer_group = ? AND topic = ?`
	return ackQuery, []interface{}{offset, consumerGroup, topic}
}

func (a MySQLOffsetScheme) NextOffsetQuery(topic, consumerGroup string) (string, []interface{}) {
	return `SELECT COALESCE(
				(SELECT offset_acked
				 FROM ` + a.MessagesOffsetsTable(a.TaskHubName) + `
				 WHERE consumer_group=? AND topic=? FOR UPDATE
				), 0)`,
		[]interface{}{consumerGroup, topic}
}

func (a MySQLOffsetScheme) ConsumedMessageQuery(
	topic string,
	offset int,
	consumerGroup string,
	consumerULID []byte,
) (string, []interface{}) {
	// offset_consumed is not queried anywhere, it's used only to detect race conditions with NextOffsetQuery.
	ackQuery := `INSERT INTO ` + a.MessagesOffsetsTable(a.TaskHubName) + ` (offset_consumed, consumer_group, topic)
		VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE offset_consumed=VALUES(offset_consumed)`
	return ackQuery, []interface{}{offset, consumerGroup, topic}
}
