CREATE TABLE IF NOT EXISTS torrent_notifications (
	id VARCHAR(255) PRIMARY KEY NOT NULL,
	torrent_id BIGINT NOT NULL,
	channel_id VARCHAR(255),
	recipient_id VARCHAR(255),
	CONSTRAINT fk_torrent_id
      FOREIGN KEY(torrent_id) 
	  	REFERENCES torrents(id)
);

CREATE TABLE IF NOT EXISTS private_channels (
	id VARCHAR(255) PRIMARY KEY NOT NULL,
	recipient_id VARCHAR(255) UNIQUE NOT NULL,
	created_at TIMESTAMP WITH TIME ZONE NOT NULL,
	last_message_at TIMESTAMP WITH TIME ZONE
);
