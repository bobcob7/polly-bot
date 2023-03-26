CREATE TABLE IF NOT EXISTS torrents (
	id BIGINT PRIMARY KEY NOT NULL,
	name VARCHAR(255) NOT NULL,
	friendly_name VARCHAR(255),
	created_at TIMESTAMP WITH TIME ZONE NOT NULL,
	started_at TIMESTAMP WITH TIME ZONE,
	updated_at TIMESTAMP WITH TIME ZONE,
	completed_at TIMESTAMP WITH TIME ZONE,
	deleted_at TIMESTAMP WITH TIME ZONE,
	status INTEGER NOT NULL,
	magnet_link TEXT NOT NULL,
	total_size BIGINT NOT NULL,
	downloaded BIGINT NOT NULL,
	uploaded BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS torrent_labels (
	torrent_id BIGINT NOT NULL,
	key VARCHAR(255) NOT NULL,
	value VARCHAR(255) NOT NULL,
	PRIMARY KEY(torrent_id, key),
	CONSTRAINT fk_torrent_id
      FOREIGN KEY(torrent_id) 
	  	REFERENCES torrents(id)
);

CREATE TABLE IF NOT EXISTS torrent_categories (
	torrent_id BIGINT NOT NULL,
	category VARCHAR(255) NOT NULL,
	PRIMARY KEY(torrent_id, category),
	CONSTRAINT fk_torrent_id
      FOREIGN KEY(torrent_id) 
	  	REFERENCES torrents(id)
);
