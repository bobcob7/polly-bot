CREATE TABLE IF NOT EXISTS torrents (
	id INTEGER PRIMARY KEY NOT NULL,
	name STRING NOT NULL,
	percent_done FLOAT NOT NULL,
	total_size INTEGER NOT NULL,
	status INTEGER NOT NULL,
	left_until_done INTEGER NOT NULL,
	rate_downloaded INTEGER NOT NULL,
	is_stalled BOOLEAN NOT NULL
);