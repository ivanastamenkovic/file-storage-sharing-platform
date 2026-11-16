CREATE KEYSPACE IF NOT EXISTS notifications
  WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};

CREATE TABLE IF NOT EXISTS notifications.notifications (
  user_id    text,
  created_at timestamp,
  id         uuid,
  type       text,
  message    text,
  is_read    boolean,
  PRIMARY KEY ((user_id), created_at, id)
) WITH CLUSTERING ORDER BY (created_at DESC, id ASC);
